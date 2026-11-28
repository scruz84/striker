import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import type { PlayerSeasonStats } from '../../api/types'
import { ResponsiveTable } from '../../components/ResponsiveTable'
import { TeamNameLink } from '../../components/TeamNameLink'
import { competitionLabel, formatAverageRating } from './competition'

// A player's lines per season, competition and team (openspec player-stats "Season lines on the
// profile"), newest season first. Nothing is shown before his first appearance.
export function PlayerSeasonStatsSection({ lines }: { lines: PlayerSeasonStats[] }) {
  const { t } = useTranslation()
  if (lines.length === 0) return null
  const key = (line: PlayerSeasonStats) => `${line.season}-${line.competition}-${line.competitionId ?? 0}-${line.teamId}`
  return (
    <Stack spacing={1}>
      <Typography variant="h6">{t('stats.playerSeasons.title')}</Typography>
      <ResponsiveTable
        ariaLabel={t('stats.playerSeasons.title')}
        rows={lines}
        getKey={key}
        columns={[
          { id: 'season', header: t('stats.columns.season'), cell: (line) => line.season },
          { id: 'competition', header: t('stats.columns.competition'), cell: (line) => competitionLabel(t, line) },
          {
            id: 'team',
            header: t('stats.columns.team'),
            cell: (line) => <TeamNameLink teamId={line.teamId}>{line.teamName}</TeamNameLink>,
          },
          { id: 'apps', header: t('stats.columns.appearances'), align: 'right', cell: (line) => `${line.appearances} (${line.starts})` },
          { id: 'minutes', header: t('stats.columns.minutes'), align: 'right', cell: (line) => line.minutes },
          { id: 'goals', header: t('stats.columns.goals'), align: 'right', cell: (line) => line.goals },
          { id: 'cards', header: t('stats.columns.cards'), align: 'right', cell: (line) => `${line.yellowCards} / ${line.redCards}` },
          { id: 'rating', header: t('stats.columns.rating'), align: 'right', cell: (line) => formatAverageRating(line.averageRating) },
        ]}
        mobile={{
          primary: (line) => `${t('stats.seasonShort', { season: line.season })} · ${competitionLabel(t, line)}`,
          secondary: (line) =>
            `${line.teamName} · ${t('stats.summary', { appearances: line.appearances, minutes: line.minutes, cards: line.yellowCards + line.redCards })}`,
          end: (line) => (
            <Stack sx={{ alignItems: 'flex-end' }}>
              <Typography variant="body2" sx={{ fontWeight: 700 }}>
                {t('stats.goalsShort', { count: line.goals })}
              </Typography>
              <Typography variant="caption" color="text.secondary">
                {formatAverageRating(line.averageRating)}
              </Typography>
            </Stack>
          ),
        }}
      />
    </Stack>
  )
}
