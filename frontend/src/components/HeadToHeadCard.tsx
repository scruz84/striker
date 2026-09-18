import Box from '@mui/material/Box'
import Paper from '@mui/material/Paper'
import Skeleton from '@mui/material/Skeleton'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import { useHeadToHead } from '../api/hooks'
import { TeamNameLink } from './TeamNameLink'
import { formatDate } from '../utils/formatDate'

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
    <Paper sx={{ p: { xs: 1.5, sm: 2 } }}>
      <Typography variant="subtitle1" sx={{ mb: 1 }}>
        {t('headToHead.title')}
      </Typography>
      {/* Three stat cells (wins/draws/wins) instead of a wordy "X wins - draws - Y wins" sentence
          - that sentence's three clauses each end up on their own line once the team names are
          long enough not to all fit on one row at phone width, tripling this section's height
          for no extra information (the numbers are already in the same order as the labels). */}
      <Stack direction="row" sx={{ textAlign: 'center', mb: 0.5 }}>
        <StatCell value={data.teamAWins} label={teamAName} />
        <StatCell value={data.draws} label={t('headToHead.draws')} />
        <StatCell value={data.teamBWins} label={teamBName} />
      </Stack>
      <Typography variant="caption" color="text.secondary" sx={{ display: 'block', textAlign: 'center', mb: 1.5 }}>
        {t('headToHead.goals', { teamA: teamAName, teamAGoals: data.teamAGoals, teamB: teamBName, teamBGoals: data.teamBGoals })}
      </Typography>
      <Stack spacing={0.75}>
        {data.meetings.map((meeting) => {
          const homeIsTeamA = meeting.homeTeamId === teamAId
          const homeId = homeIsTeamA ? teamAId : teamBId
          const homeName = homeIsTeamA ? teamAName : teamBName
          const awayId = homeIsTeamA ? teamBId : teamAId
          const awayName = homeIsTeamA ? teamBName : teamAName
          return (
            <Stack key={meeting.matchId} direction="row" spacing={1} sx={{ alignItems: 'baseline' }}>
              <Typography variant="caption" color="text.secondary" sx={{ flexShrink: 0, width: 56 }}>
                {formatDate(meeting.matchDate)}
              </Typography>
              {/* Left to wrap onto a second line for an unusually long pair of team names, rather
                  than the old flex row that fought the date for space and forced the date onto
                  its own line underneath instead - a rare wrap here is still more compact than
                  that always happened. */}
              <Typography variant="body2" sx={{ minWidth: 0 }}>
                <TeamNameLink teamId={homeId}>{homeName}</TeamNameLink> {meeting.homeGoals} - {meeting.awayGoals}{' '}
                <TeamNameLink teamId={awayId}>{awayName}</TeamNameLink>
              </Typography>
            </Stack>
          )
        })}
      </Stack>
    </Paper>
  )
}

function StatCell({ value, label }: { value: number; label: string }) {
  return (
    <Box sx={{ flex: 1, minWidth: 0 }}>
      <Typography variant="h6" sx={{ fontWeight: 700, lineHeight: 1.2 }}>
        {value}
      </Typography>
      <Typography variant="caption" color="text.secondary" noWrap sx={{ display: 'block', px: 0.5 }}>
        {label}
      </Typography>
    </Box>
  )
}
