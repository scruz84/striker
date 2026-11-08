import { useState } from 'react'
import Chip from '@mui/material/Chip'
import Divider from '@mui/material/Divider'
import Paper from '@mui/material/Paper'
import Skeleton from '@mui/material/Skeleton'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import SportsSoccerOutlinedIcon from '@mui/icons-material/SportsSoccerOutlined'
import { useTranslation } from 'react-i18next'
import { useNationalTeamRound, useUserTeam } from '../../api/hooks'
import type { NationalMatch } from '../../api/types'
import { EmptyState } from '../../components/EmptyState'
import { NationalMatchFilters } from './NationalMatchFilters'
import { NationalMatchRow } from './NationalMatchRow'
import { EMPTY_FILTER, formatKickoffHeader, matchesFilter, splitRound, type KickoffGroup } from './nationalMatches'

/** The current international round: every national-team match still to play plus those of the
 * last day - live first, then upcoming and just-finished matches grouped by kickoff in the
 * viewer's timezone (see backend common.GetNationalTeamRound). */
export function RoundTab() {
  const { t } = useTranslation()
  const { data, isLoading, isError } = useNationalTeamRound(true)
  const { data: userTeam } = useUserTeam(true)
  const [filter, setFilter] = useState(EMPTY_FILTER)

  if (isLoading) {
    return <Skeleton variant="rectangular" height={240} />
  }
  if (isError || !data) {
    return <EmptyState compact title={t('nationalTeams.round.error')} />
  }

  const visible = data.filter((m) => matchesFilter(t, m, filter))
  const { live, upcoming, finished } = splitRound(visible)

  return (
    <Stack spacing={2}>
      <NationalMatchFilters filter={filter} onChange={setFilter} showMineToggle={!!userTeam} />
      <Stack direction="row" sx={{ flexWrap: 'wrap', gap: 0.75 }}>
        <Chip size="small" color={live.length > 0 ? 'error' : 'default'} label={t('nationalTeams.round.liveCount', { count: live.length })} />
        <Chip size="small" label={t('nationalTeams.round.upcomingCount', { count: upcoming.reduce((n, g) => n + g.matches.length, 0) })} />
        <Chip size="small" label={t('nationalTeams.round.finishedCount', { count: finished.reduce((n, g) => n + g.matches.length, 0) })} />
      </Stack>

      {visible.length === 0 && (
        <EmptyState
          icon={<SportsSoccerOutlinedIcon fontSize="inherit" />}
          title={data.length === 0 ? t('nationalTeams.round.emptyTitle') : t('nationalTeams.filters.noMatches')}
          text={data.length === 0 ? t('nationalTeams.round.empty') : undefined}
        />
      )}

      {live.length > 0 && (
        <Paper variant="outlined" sx={{ p: 1, borderColor: 'error.main' }}>
          <SectionTitle title={t('nationalTeams.round.live')} live />
          <MatchList matches={live} />
        </Paper>
      )}
      {upcoming.length > 0 && (
        <Paper variant="outlined" sx={{ p: 1 }}>
          <SectionTitle title={t('nationalTeams.round.upcoming')} />
          <KickoffGroups groups={upcoming} />
        </Paper>
      )}
      {finished.length > 0 && (
        <Paper variant="outlined" sx={{ p: 1 }}>
          <SectionTitle title={t('nationalTeams.round.finished')} />
          <KickoffGroups groups={finished} />
        </Paper>
      )}
    </Stack>
  )
}

function SectionTitle({ title, live = false }: { title: string; live?: boolean }) {
  const { t } = useTranslation()
  return (
    <Stack direction="row" spacing={1} sx={{ alignItems: 'center', px: 1, pt: 0.5, pb: 1 }}>
      {live && <Chip label={t('match.live')} color="error" size="small" sx={{ fontWeight: 700 }} />}
      <Typography variant="subtitle1" component="h2">
        {title}
      </Typography>
    </Stack>
  )
}

function MatchList({ matches }: { matches: NationalMatch[] }) {
  return (
    <Stack divider={<Divider flexItem />}>
      {matches.map((match) => (
        <NationalMatchRow key={`${match.kind}-${match.id}`} match={match} />
      ))}
    </Stack>
  )
}

function KickoffGroups({ groups }: { groups: KickoffGroup[] }) {
  return (
    <Stack spacing={1.5}>
      {groups.map((group) => (
        <Stack key={group.key} spacing={0.5}>
          <Typography variant="overline" color="text.secondary" sx={{ px: 1, lineHeight: 1.6 }}>
            {formatKickoffHeader(group.kickoff)}
          </Typography>
          <MatchList matches={group.matches} />
        </Stack>
      ))}
    </Stack>
  )
}
