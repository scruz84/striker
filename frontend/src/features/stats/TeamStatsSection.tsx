import { useState } from 'react'
import Chip from '@mui/material/Chip'
import MenuItem from '@mui/material/MenuItem'
import Skeleton from '@mui/material/Skeleton'
import Stack from '@mui/material/Stack'
import TextField from '@mui/material/TextField'
import { useTranslation } from 'react-i18next'
import { useTeamSeasonStats } from '../../api/hooks'
import type { TeamPlayerStats } from '../../api/types'
import { CountryFlag } from '../../components/CountryFlag'
import { EmptyState } from '../../components/EmptyState'
import { PlayerNameLink } from '../../components/PlayerNameLink'
import { ResponsiveTable } from '../../components/ResponsiveTable'
import { formatAverageRating } from './competition'
import { sortTeamStats, TEAM_STATS_SORTS, type TeamStatsSort } from './teamStatsSort'

// A team's squad stats for one season (openspec player-stats "Squad stats"): every player who
// played for it that season - still at the club, gone or retired - over every competition but
// club friendlies.
export function TeamStatsSection({ teamId }: { teamId: string | undefined }) {
  const { t } = useTranslation()
  const [season, setSeason] = useState<number | undefined>(undefined)
  const [sort, setSort] = useState<TeamStatsSort>('minutes')
  const { data, isLoading } = useTeamSeasonStats(teamId, season)

  if (isLoading || !data) {
    return <Skeleton variant="rectangular" height={200} />
  }
  if (data.seasons.length === 0) {
    return <EmptyState title={t('stats.team.empty')} />
  }
  const name = (p: TeamPlayerStats) => (
    <>
      <CountryFlag code={p.birthCountry} height={14} marginRight={0.5} />
      <PlayerNameLink playerId={p.playerId}>
        {p.firstName} {p.lastName}
      </PlayerNameLink>
      {!p.inSquad && <Chip size="small" sx={{ ml: 0.75 }} label={p.retired ? t('stats.team.retired') : t('stats.team.left')} />}
    </>
  )

  return (
    <Stack spacing={1.5}>
      <Stack direction="row" spacing={1} sx={{ flexWrap: 'wrap', rowGap: 1 }}>
        <TextField
          select
          size="small"
          value={data.season}
          onChange={(event) => setSeason(Number(event.target.value))}
          slotProps={{ htmlInput: { 'aria-label': t('stats.team.season') } }}
          sx={{ minWidth: 140 }}
        >
          {data.seasons.map((s) => (
            <MenuItem key={s} value={s}>
              {t('division.seasonOption', { season: s })}
            </MenuItem>
          ))}
        </TextField>
        <TextField
          select
          size="small"
          value={sort}
          onChange={(event) => setSort(event.target.value as TeamStatsSort)}
          slotProps={{ htmlInput: { 'aria-label': t('stats.team.sort') } }}
          sx={{ minWidth: 140 }}
        >
          {TEAM_STATS_SORTS.map((value) => (
            <MenuItem key={value} value={value}>
              {t(`stats.team.sorts.${value}`)}
            </MenuItem>
          ))}
        </TextField>
      </Stack>
      <ResponsiveTable
        ariaLabel={t('stats.team.title')}
        rows={sortTeamStats(data.players, sort)}
        getKey={(p) => p.playerId}
        columns={[
          { id: 'player', header: t('stats.columns.player'), cell: name },
          { id: 'apps', header: t('stats.columns.appearances'), align: 'right', cell: (p) => `${p.appearances} (${p.starts})` },
          { id: 'minutes', header: t('stats.columns.minutes'), align: 'right', cell: (p) => p.minutes },
          { id: 'goals', header: t('stats.columns.goals'), align: 'right', cell: (p) => p.goals },
          { id: 'penalties', header: t('stats.columns.penalties'), align: 'right', cell: (p) => p.penaltyGoals },
          { id: 'cards', header: t('stats.columns.cards'), align: 'right', cell: (p) => `${p.yellowCards} / ${p.redCards}` },
          { id: 'rating', header: t('stats.columns.rating'), align: 'right', cell: (p) => formatAverageRating(p.averageRating) },
        ]}
        mobile={{
          primary: name,
          secondary: (p) => t('stats.summary', { appearances: p.appearances, minutes: p.minutes, cards: p.yellowCards + p.redCards }),
          end: (p) => `${t('stats.goalsShort', { count: p.goals })} · ${formatAverageRating(p.averageRating)}`,
        }}
      />
    </Stack>
  )
}
