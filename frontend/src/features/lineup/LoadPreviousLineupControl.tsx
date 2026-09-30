import { useEffect, useState } from 'react'
import Alert from '@mui/material/Alert'
import Box from '@mui/material/Box'
import CircularProgress from '@mui/material/CircularProgress'
import FormControl from '@mui/material/FormControl'
import InputLabel from '@mui/material/InputLabel'
import MenuItem from '@mui/material/MenuItem'
import Select, { type SelectChangeEvent } from '@mui/material/Select'
import Stack from '@mui/material/Stack'
import { useTranslation } from 'react-i18next'
import { useMatchDetails, useTeamMatches } from '../../api/hooks'
import { MATCH_KIND_LEAGUE, type Lineup, type WeekMatch } from '../../api/types'
import { formatDate } from '../../utils/formatDate'

// How far back the dropdown offers - a manager reusing an old lineup almost always means a
// recent one; older results are still reachable from the team's own match history page.
const MAX_PREVIOUS_MATCHES = 15

export type LoadOutcome = { action: 'wait' } | { action: 'apply'; lineup: Lineup } | { action: 'error' }

/** Decides what a pending "copy this match's lineup" pick should do once useMatchDetails settles,
 * given a match id is actually pending (the caller only calls this while pendingMatchId is set).
 * Pulled out of the effect below so the one subtle bit of logic here - a failed *background*
 * refetch of an already-cached match must win over stale cached data, not get silently overridden
 * by it - has a direct, render-free unit test instead of only being reachable by rendering the
 * whole component through React Query. */
export function resolveLoadOutcome(
  teamId: number,
  isFetching: boolean,
  fetchError: boolean,
  match: WeekMatch | undefined,
): LoadOutcome {
  if (isFetching) {
    return { action: 'wait' }
  }
  // Checked before `match`: React Query keeps the last-good data around through a failed
  // *background* refetch of an already-cached match (isFetching flips back to false, data stays
  // truthy, but so does isError) - without this check first, a transient refetch failure would
  // silently re-apply that stale-but-present data while also showing the error banner.
  if (fetchError) {
    return { action: 'error' }
  }
  if (!match) {
    return { action: 'wait' }
  }
  const lineup = match.home === teamId ? match.homeLineup : match.awayLineup
  return lineup ? { action: 'apply', lineup } : { action: 'error' }
}

interface LoadPreviousLineupControlProps {
  teamId: number
  onLoad: (lineup: Lineup) => void
}

/** Lets a manager copy one of their own team's past played league matches' lineup (formation +
 * XI) straight into the editor, instead of rebuilding it slot by slot - "use the same lineup as
 * that match". Scoped to league matches: cup ties and friendlies store their lineups through
 * separate endpoints (see CupTieDetailsPage/FriendlyMatchPage) this control doesn't fetch from.
 * A one-shot action, not a persistent selection - the dropdown resets to its placeholder once the
 * lineup loads (or fails to), the same way MatchLineupForm's own inputs aren't "bound" to a match.
 * Renders nothing once the team has no played league match yet to offer. */
export function LoadPreviousLineupControl({ teamId, onLoad }: LoadPreviousLineupControlProps) {
  const { t } = useTranslation()
  const { data: matches } = useTeamMatches(String(teamId))
  const [pendingMatchId, setPendingMatchId] = useState<string | undefined>(undefined)
  const [loadError, setLoadError] = useState(false)
  const { data: match, isFetching, isError: fetchError } = useMatchDetails(pendingMatchId)

  const pastMatches = (matches ?? [])
    .filter((m) => m.kind === MATCH_KIND_LEAGUE && m.played)
    .sort((a, b) => b.matchDate.localeCompare(a.matchDate))
    .slice(0, MAX_PREVIOUS_MATCHES)

  useEffect(() => {
    if (!pendingMatchId) {
      return
    }
    const outcome = resolveLoadOutcome(teamId, isFetching, fetchError, match)
    if (outcome.action === 'wait') {
      return
    }
    if (outcome.action === 'apply') {
      onLoad(outcome.lineup)
    } else {
      setLoadError(true)
    }
    setPendingMatchId(undefined)
    // onLoad intentionally excluded - it's a fresh closure every render (useLineupSelections'
    // loadLineup isn't memoized), and including it would only make this effect re-run redundantly
    // right after it already reset pendingMatchId to undefined, not change what it does.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [match, isFetching, fetchError, pendingMatchId, teamId])

  if (pastMatches.length === 0) {
    return null
  }

  const handleChange = (event: SelectChangeEvent) => {
    setLoadError(false)
    setPendingMatchId(event.target.value)
  }

  return (
    <Box sx={{ mb: { xs: 1, sm: 2 } }}>
      <Stack direction="row" spacing={1} sx={{ alignItems: 'center' }}>
        <FormControl size="small" sx={{ minWidth: { xs: '100%', sm: 280 } }} disabled={isFetching}>
          <InputLabel id="load-previous-lineup-label">{t('lineup.loadPrevious.label')}</InputLabel>
          <Select
            labelId="load-previous-lineup-label"
            label={t('lineup.loadPrevious.label')}
            value={pendingMatchId ?? ''}
            displayEmpty
            onChange={handleChange}
          >
            {pastMatches.map((m) => (
              <MenuItem key={m.matchId} value={String(m.matchId)}>
                {t('lineup.loadPrevious.option', {
                  opponent: m.opponentName,
                  date: formatDate(m.matchDate),
                  ownGoals: m.isHome ? m.homeGoals : m.awayGoals,
                  opponentGoals: m.isHome ? m.awayGoals : m.homeGoals,
                })}
              </MenuItem>
            ))}
          </Select>
        </FormControl>
        {isFetching && <CircularProgress size={18} />}
      </Stack>
      {loadError && (
        <Alert severity="error" sx={{ mt: 1 }}>
          {t('lineup.loadPrevious.error')}
        </Alert>
      )}
    </Box>
  )
}
