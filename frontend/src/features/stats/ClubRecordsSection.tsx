import Grid from '@mui/material/Grid'
import Skeleton from '@mui/material/Skeleton'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import { useTeamRecords } from '../../api/hooks'
import { PlayerNameLink } from '../../components/PlayerNameLink'
import { TeamNameLink } from '../../components/TeamNameLink'
import { formatDivisionName } from '../../utils/divisionName'
import { formatCurrency, formatNumber } from '../../utils/formatCurrency'
import { RecordCard, RecordMatchLink, RecordPlayerList, RecordRow, RecordStreakValue } from './RecordRows'

// A club's records (openspec records "Club records"), on its history tab. A record nobody holds
// yet is left out; a club without any shows nothing.
export function ClubRecordsSection({ teamId }: { teamId: string | undefined }) {
  const { t } = useTranslation()
  const { data: records, isLoading } = useTeamRecords(teamId)
  if (isLoading || !records) return <Skeleton variant="rectangular" height={160} />

  const results = [
    records.biggestWin && (
      <RecordRow key="win" label={t('stats.records.biggestWin')}>
        <RecordMatchLink match={records.biggestWin} />
      </RecordRow>
    ),
    records.biggestDefeat && (
      <RecordRow key="defeat" label={t('stats.records.biggestDefeat')}>
        <RecordMatchLink match={records.biggestDefeat} />
      </RecordRow>
    ),
    records.highestScoring && (
      <RecordRow key="scoring" label={t('stats.records.highestScoring')}>
        <RecordMatchLink match={records.highestScoring} />
      </RecordRow>
    ),
    records.recordAttendance?.attendance !== undefined && (
      <RecordRow key="attendance" label={t('stats.records.attendance')}>
        {formatNumber(records.recordAttendance.attendance)}{' '}
        <Typography component="span" variant="caption" color="text.secondary">
          ({records.recordAttendance.homeTeamName} - {records.recordAttendance.awayTeamName})
        </Typography>
      </RecordRow>
    ),
  ].filter(Boolean)
  const streaks = [
    records.longestWinningStreak && (
      <RecordRow key="wins" label={t('stats.records.winningStreak')}>
        <RecordStreakValue streak={records.longestWinningStreak} />
      </RecordRow>
    ),
    records.longestUnbeatenStreak && (
      <RecordRow key="unbeaten" label={t('stats.records.unbeatenStreak')}>
        <RecordStreakValue streak={records.longestUnbeatenStreak} />
      </RecordRow>
    ),
    records.longestLosingStreak && (
      <RecordRow key="losses" label={t('stats.records.losingStreak')}>
        <RecordStreakValue streak={records.longestLosingStreak} />
      </RecordRow>
    ),
  ].filter(Boolean)
  const seasons = [
    records.bestFinish && (
      <RecordRow key="finish" label={t('stats.records.bestFinish')}>
        {t('stats.records.finish', {
          position: t('subheader.position', { count: records.bestFinish.position, ordinal: true }),
          league: formatDivisionName(t, records.bestFinish.leagueName),
          season: records.bestFinish.season,
        })}
      </RecordRow>
    ),
    records.mostPoints && (
      <RecordRow key="points" label={t('stats.records.mostPoints')}>
        {t('stats.records.points', { points: records.mostPoints.points, season: records.mostPoints.season })}
      </RecordRow>
    ),
    records.recordSigning && (
      <RecordRow key="signing" label={t('stats.records.recordSigning')}>
        <PlayerNameLink playerId={records.recordSigning.playerId}>{records.recordSigning.playerName}</PlayerNameLink>
        {' · '}
        {formatCurrency(records.recordSigning.fee)}
        {records.recordSigning.otherTeamId !== undefined && (
          <Typography component="span" variant="caption" color="text.secondary" sx={{ display: 'block' }}>
            {t('stats.records.from')} <TeamNameLink teamId={records.recordSigning.otherTeamId}>{records.recordSigning.otherTeamName}</TeamNameLink>
          </Typography>
        )}
      </RecordRow>
    ),
    records.recordSale && (
      <RecordRow key="sale" label={t('stats.records.recordSale')}>
        <PlayerNameLink playerId={records.recordSale.playerId}>{records.recordSale.playerName}</PlayerNameLink>
        {' · '}
        {formatCurrency(records.recordSale.fee)}
        {records.recordSale.otherTeamId !== undefined && (
          <Typography component="span" variant="caption" color="text.secondary" sx={{ display: 'block' }}>
            {t('stats.records.to')} <TeamNameLink teamId={records.recordSale.otherTeamId}>{records.recordSale.otherTeamName}</TeamNameLink>
          </Typography>
        )}
      </RecordRow>
    ),
  ].filter(Boolean)

  if (results.length + streaks.length + seasons.length + records.topScorers.length + records.mostAppearances.length === 0) {
    return null
  }
  return (
    <Stack spacing={1}>
      <Typography variant="subtitle1" component="h2" sx={{ fontWeight: 600 }}>
        {t('stats.records.clubTitle')}
      </Typography>
      <Grid container spacing={1.5}>
        {results.length > 0 && (
          <Grid size={{ xs: 12, md: 6 }}>
            <RecordCard title={t('stats.records.results')}>{results}</RecordCard>
          </Grid>
        )}
        {streaks.length > 0 && (
          <Grid size={{ xs: 12, md: 6 }}>
            <RecordCard title={t('stats.records.streaks')}>{streaks}</RecordCard>
          </Grid>
        )}
        {seasons.length > 0 && (
          <Grid size={{ xs: 12, md: 6 }}>
            <RecordCard title={t('stats.records.seasonsAndMarket')}>{seasons}</RecordCard>
          </Grid>
        )}
        {records.topScorers.length > 0 && (
          <Grid size={{ xs: 12, md: 6 }}>
            <RecordPlayerList title={t('stats.records.topScorers')} players={records.topScorers} figure="goals" />
          </Grid>
        )}
        {records.mostAppearances.length > 0 && (
          <Grid size={{ xs: 12, md: 6 }}>
            <RecordPlayerList title={t('stats.records.mostAppearances')} players={records.mostAppearances} figure="appearances" />
          </Grid>
        )}
      </Grid>
    </Stack>
  )
}
