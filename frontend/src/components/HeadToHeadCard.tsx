import Box from '@mui/material/Box'
import Paper from '@mui/material/Paper'
import Skeleton from '@mui/material/Skeleton'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import { useHeadToHead } from '../api/hooks'
import { TeamNameLink } from './TeamNameLink'
import { formatDateTime } from '../utils/formatDate'

interface HeadToHeadCardProps {
  teamAId: number
  teamAName: string
  teamBId: number
  teamBName: string
}

export function HeadToHeadCard({ teamAId, teamAName, teamBId, teamBName }: HeadToHeadCardProps) {
  const { t } = useTranslation()
  const { data, isLoading } = useHeadToHead(teamAId, teamBId)

  if (isLoading) {
    return <Skeleton variant="rectangular" height={120} />
  }

  if (!data || data.totalMatches === 0) {
    return (
      <Paper sx={{ p: 2 }}>
        <Typography variant="subtitle1" gutterBottom>
          {t('headToHead.title')}
        </Typography>
        <Typography variant="body2" color="text.secondary">
          {t('headToHead.noMeetings')}
        </Typography>
      </Paper>
    )
  }

  return (
    <Paper sx={{ p: 2 }}>
      <Typography variant="subtitle1" gutterBottom>
        {t('headToHead.title')}
      </Typography>
      <Stack direction="row" spacing={2} sx={{ mb: 2, alignItems: 'baseline', flexWrap: 'wrap' }}>
        <Typography variant="h6">
          {data.teamAWins} - {data.draws} - {data.teamBWins}
        </Typography>
        <Typography variant="body2" color="text.secondary">
          {t('headToHead.record', { teamA: teamAName, teamB: teamBName })}
        </Typography>
        <Typography variant="body2" color="text.secondary">
          {t('headToHead.goals', { teamA: teamAName, teamAGoals: data.teamAGoals, teamB: teamBName, teamBGoals: data.teamBGoals })}
        </Typography>
      </Stack>
      <Stack spacing={0.5}>
        {data.meetings.map((meeting) => {
          const homeIsTeamA = meeting.homeTeamId === teamAId
          const homeId = homeIsTeamA ? teamAId : teamBId
          const homeName = homeIsTeamA ? teamAName : teamBName
          const awayId = homeIsTeamA ? teamBId : teamAId
          const awayName = homeIsTeamA ? teamBName : teamAName
          return (
            <Box key={meeting.matchId} sx={{ display: 'flex', justifyContent: 'space-between', gap: 1 }}>
              <Typography variant="body2">
                <TeamNameLink teamId={homeId}>{homeName}</TeamNameLink> {meeting.homeGoals} - {meeting.awayGoals}{' '}
                <TeamNameLink teamId={awayId}>{awayName}</TeamNameLink>
              </Typography>
              <Typography variant="caption" color="text.secondary">
                {formatDateTime(meeting.matchDate)}
              </Typography>
            </Box>
          )
        })}
      </Stack>
    </Paper>
  )
}
