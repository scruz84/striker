import Box from '@mui/material/Box'
import Chip from '@mui/material/Chip'
import ButtonBase from '@mui/material/ButtonBase'
import Paper from '@mui/material/Paper'
import Skeleton from '@mui/material/Skeleton'
import Stack from '@mui/material/Stack'
import Tooltip from '@mui/material/Tooltip'
import Typography from '@mui/material/Typography'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useCupDetails, useLatestCupId, useMe } from '../../api/hooks'
import type { CupTie } from '../../api/types'
import { isPast, mightStillBeLive } from '../../api/useLiveMatch'
import { NestedTeamNameLink } from '../../components/TeamNameLink'
import { formatDateTime } from '../../utils/formatDate'
import { localizedTeamName } from '../../utils/nationalTeamName'

export function CupPage() {
  const { t } = useTranslation()
  const { data: loggedUser } = useMe()
  const { data: latestCup, isLoading: cupIdLoading } = useLatestCupId(loggedUser?.country)
  const { data: cupDetails, isLoading: detailsLoading } = useCupDetails(latestCup?.cupId)

  if (cupIdLoading || detailsLoading) {
    return <Skeleton variant="rectangular" height={300} />
  }

  if (!cupDetails) {
    return <Typography color="text.secondary">{t('cup.noCup')}</Typography>
  }

  const { cup, ties } = cupDetails
  const rounds = groupByRound(ties)

  return (
    <Box>
      <Typography variant="h5" gutterBottom>
        {t('cup.titleWithSeason', { name: cup.name, season: cup.season })}
      </Typography>
      {cup.completed && (
        <Chip color="success" label={t('cup.champion', { name: cup.championTeamName })} sx={{ mb: 2 }} />
      )}

      {rounds.map(({ round, ties: roundTies }) => (
        <Paper key={round} sx={{ mb: 2, p: 2 }}>
          <Typography variant="subtitle1" gutterBottom>
            {roundName(t, round, rounds.length)}
          </Typography>
          <Stack spacing={1}>
            {roundTies.map((tie) => (
              <TieRow key={tie.tieId} tie={tie} />
            ))}
          </Stack>
        </Paper>
      ))}
    </Box>
  )
}

// A tie decided by a penalty shootout ends level on the scoreboard (0-0, 1-1, ...), so unlike a
// normally-decided tie, the score alone can't tell a reader who actually advances - only
// tie.winnerTeamId (set by the coin-flip shootout in backend engine.simulateCupTie) says that.
// Returns undefined for a tie that wasn't decided on penalties, or isn't played/resolved yet, so
// callers can fall back to the plain "(pens)" caption (issue: bracket/tie page said penalties
// happened but never said who won).
export function cupTiePenaltyWinnerName(tie: CupTie): string | undefined {
  if (!tie.wentToPenalties || tie.winnerTeamId === null) return undefined
  return tie.winnerTeamId === tie.homeTeamId ? tie.homeTeamName : tie.awayTeamName
}

// Exported for InternationalCupPage - identical bracket rendering, just keyed by region instead
// of country to find its cup id (see useLatestInternationalCupId).
export function TieRow({ tie }: { tie: CupTie }) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const isBye = tie.awayTeamId === null
  // See DivisionDetailsPage's identical mightStillBeLive check - a tie flips played=true the
  // instant it's simulated, well before its result finishes broadcasting live.
  const stillLive = tie.played && mightStillBeLive(tie.matchDate ?? '')
  const penaltyWinnerName = cupTiePenaltyWinnerName(tie)
  const homeWonPenalties = tie.wentToPenalties && tie.winnerTeamId === tie.homeTeamId
  const awayWonPenalties = tie.wentToPenalties && tie.winnerTeamId === tie.awayTeamId
  // This bracket is shared by regular club cups and the World Cup knockout stage alike (see
  // NationalTeamsPage's WorldCupSection) - localizedTeamName is a no-op for an ordinary club name,
  // so it's always safe to apply here regardless of which one a given tie belongs to.

  return (
    <ButtonBase
      disabled={isBye}
      onClick={() => navigate(`/cup/ties/${tie.tieId}`)}
      sx={{ width: '100%', justifyContent: 'stretch', borderRadius: 1 }}
    >
      <Stack direction="row" spacing={1} sx={{ py: 0.5, alignItems: 'center', width: '100%' }}>
        <Typography variant="body2" sx={{ flex: 1, textAlign: 'left' }}>
          {tie.homeTeamId ? (
            <NestedTeamNameLink teamId={tie.homeTeamId} sx={{ fontWeight: homeWonPenalties ? 700 : undefined }}>
              {localizedTeamName(t, tie.homeTeamName)}
            </NestedTeamNameLink>
          ) : (
            t('cup.tbd')
          )}
        </Typography>
        {stillLive ? (
          <Chip label={t('match.live')} color="error" size="small" sx={{ fontWeight: 700 }} />
        ) : (
          <Typography variant="body2" sx={{ minWidth: 80, textAlign: 'center' }}>
            {isBye ? t('cup.bye') : tie.played && isPast(tie.matchDate ?? '') ? `${tie.homeGoals} - ${tie.awayGoals}` : formatDateTime(tie.matchDate ?? '')}
          </Typography>
        )}
        <Typography variant="body2" sx={{ flex: 1, textAlign: 'right' }}>
          {tie.awayTeamId ? (
            <NestedTeamNameLink teamId={tie.awayTeamId} sx={{ fontWeight: awayWonPenalties ? 700 : undefined }}>
              {localizedTeamName(t, tie.awayTeamName)}
            </NestedTeamNameLink>
          ) : isBye ? (
            ''
          ) : (
            t('cup.tbd')
          )}
        </Typography>
        {!stillLive && tie.wentToPenalties && (
          <Tooltip title={penaltyWinnerName ? t('cup.pensWinner', { team: localizedTeamName(t, penaltyWinnerName) }) : ''}>
            <Typography variant="caption" color="text.secondary">
              {t('cup.pens')}
            </Typography>
          </Tooltip>
        )}
      </Stack>
    </ButtonBase>
  )
}

export function groupByRound(ties: CupTie[]): { round: number; ties: CupTie[] }[] {
  const byRound = new Map<number, CupTie[]>()
  for (const tie of ties) {
    const list = byRound.get(tie.round) ?? []
    list.push(tie)
    byRound.set(tie.round, list)
  }
  return Array.from(byRound.entries())
    .sort(([a], [b]) => a - b)
    .map(([round, roundTies]) => ({
      round,
      ties: [...roundTies].sort((a, b) => a.slotNumber - b.slotNumber),
    }))
}

export function roundName(t: (key: string, options?: Record<string, unknown>) => string, round: number, totalRounds: number): string {
  const roundsFromEnd = totalRounds - round
  if (roundsFromEnd === 0) return t('cup.final')
  if (roundsFromEnd === 1) return t('cup.semifinal')
  if (roundsFromEnd === 2) return t('cup.quarterfinal')
  return t('cup.round', { round })
}
