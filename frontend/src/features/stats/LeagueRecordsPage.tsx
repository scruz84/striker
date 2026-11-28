import ArrowBackIcon from '@mui/icons-material/ArrowBack'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import Grid from '@mui/material/Grid'
import Skeleton from '@mui/material/Skeleton'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import { Link as RouterLink, useParams } from 'react-router-dom'
import { useDivisionDetails, useLeagueRecords } from '../../api/hooks'
import { EmptyState } from '../../components/EmptyState'
import { PageHeader } from '../../components/PageHeader'
import { PlayerNameLink } from '../../components/PlayerNameLink'
import { ResponsiveTable } from '../../components/ResponsiveTable'
import { TeamNameLink } from '../../components/TeamNameLink'
import { formatDivisionName } from '../../utils/divisionName'
import { RecordCard, RecordMatchLink, RecordPlayerList, RecordRow } from './RecordRows'

// A league's history (openspec records "League records"): its champions season by season, the
// clubs with most titles, its Golden Boots and its all-time records.
export function LeagueRecordsPage() {
  const { t } = useTranslation()
  const { id } = useParams<{ id: string }>()
  const { data: division } = useDivisionDetails(id)
  const { data: records, isLoading } = useLeagueRecords(id)
  const title = division ? formatDivisionName(t, division.name) : t('stats.records.leagueTitle')

  return (
    <Box>
      <PageHeader
        title={title}
        titleText={division?.name}
        subtitle={t('stats.records.leagueTitle')}
        action={
          <Button component={RouterLink} to={`/division/${id}`} size="small" startIcon={<ArrowBackIcon />}>
            {t('stats.records.backToLeague')}
          </Button>
        }
      />
      {isLoading || !records ? (
        <Skeleton variant="rectangular" height={300} />
      ) : records.champions.length === 0 && !records.biggestWin && records.topScorers.length === 0 ? (
        <EmptyState title={t('stats.records.leagueEmpty')} />
      ) : (
        <Grid container spacing={2}>
          <Grid size={{ xs: 12, md: 7 }}>
            <Stack spacing={1}>
              <Typography variant="h6">{t('stats.records.champions')}</Typography>
              {records.champions.length === 0 ? (
                <Typography variant="body2" color="text.secondary">
                  {t('stats.records.noChampions')}
                </Typography>
              ) : (
                <ResponsiveTable
                  ariaLabel={t('stats.records.champions')}
                  rows={records.champions}
                  getKey={(row) => row.season}
                  tableFrom="sm"
                  columns={[
                    { id: 'season', header: t('stats.columns.season'), cell: (row) => row.season },
                    { id: 'team', header: t('stats.records.champion'), cell: (row) => <TeamNameLink teamId={row.teamId}>{row.teamName}</TeamNameLink> },
                    { id: 'points', header: t('stats.records.pointsColumn'), align: 'right', cell: (row) => row.points ?? '—' },
                  ]}
                  mobile={{
                    primary: (row) => <TeamNameLink teamId={row.teamId}>{row.teamName}</TeamNameLink>,
                    secondary: (row) => t('stats.seasonShort', { season: row.season }),
                    end: (row) => (row.points !== undefined ? t('stats.records.pointsShort', { points: row.points }) : ''),
                  }}
                />
              )}
            </Stack>
          </Grid>
          <Grid size={{ xs: 12, md: 5 }}>
            <Stack spacing={1.5}>
              {records.mostTitles.length > 0 && (
                <RecordCard title={t('stats.records.mostTitles')}>
                  {records.mostTitles.slice(0, 5).map((row, index) => (
                    <RecordRow key={row.teamId} label={`${index + 1}.`}>
                      <TeamNameLink teamId={row.teamId}>{row.teamName}</TeamNameLink>
                      {' · '}
                      {t('stats.records.titles', { count: row.titles })}
                    </RecordRow>
                  ))}
                </RecordCard>
              )}
              {(records.mostPoints || records.biggestWin || records.highestScoring) && (
                <RecordCard title={t('stats.records.results')}>
                  {records.mostPoints && (
                    <RecordRow label={t('stats.records.mostPoints')}>
                      <TeamNameLink teamId={records.mostPoints.teamId}>{records.mostPoints.teamName}</TeamNameLink>
                      {' · '}
                      {t('stats.records.points', { points: records.mostPoints.points, season: records.mostPoints.season })}
                    </RecordRow>
                  )}
                  {records.biggestWin && (
                    <RecordRow label={t('stats.records.biggestWin')}>
                      <RecordMatchLink match={records.biggestWin} />
                    </RecordRow>
                  )}
                  {records.highestScoring && (
                    <RecordRow label={t('stats.records.highestScoring')}>
                      <RecordMatchLink match={records.highestScoring} />
                    </RecordRow>
                  )}
                </RecordCard>
              )}
              {records.goldenBoots.length > 0 && (
                <RecordCard title={t('stats.records.goldenBoots')}>
                  {records.goldenBoots.map((row) => (
                    <RecordRow key={row.season} label={t('stats.seasonShort', { season: row.season })}>
                      <PlayerNameLink playerId={row.playerId}>
                        {row.firstName} {row.lastName}
                      </PlayerNameLink>
                      {' · '}
                      {t('stats.goalsShort', { count: row.goals })}
                    </RecordRow>
                  ))}
                </RecordCard>
              )}
              <RecordPlayerList title={t('stats.records.allTimeScorers')} players={records.topScorers} figure="goals" />
            </Stack>
          </Grid>
        </Grid>
      )}
    </Box>
  )
}
