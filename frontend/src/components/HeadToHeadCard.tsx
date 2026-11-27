import ChevronRightIcon from '@mui/icons-material/ChevronRight'
import Box from '@mui/material/Box'
import Paper from '@mui/material/Paper'
import Skeleton from '@mui/material/Skeleton'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { useTranslation } from 'react-i18next'
import { Link as RouterLink } from 'react-router-dom'
import { useHeadToHead } from '../api/hooks'
import type { HeadToHeadMeeting } from '../api/types'
import { formatDate } from '../utils/formatDate'
import { matchTeamColors } from '../utils/teamColors'
import { TeamBadge } from './TeamBadge'
import { TeamNameLink } from './TeamNameLink'

interface HeadToHeadCardProps {
  teamAId: number
  teamAName: string
  teamBId: number
  teamBName: string
}

interface Side {
  id: number
  name: string
  color: string
}

// Every block below the header shares these columns - team A's figures, a centered label, team B's
// figures - so the numbers line up from the summary down through every meeting. The outer columns
// of a meeting row are the same fixed width (date on the left, chevron on the right), which keeps
// its score centered under the summary's labels.
const SIDE_COLUMN = 64
const TABULAR = { fontVariantNumeric: 'tabular-nums' } as const

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

  // Distinct colors even when both teams' default badge colors collide, since the badges are what
  // tells the two apart in the meetings list.
  const colors = matchTeamColors(teamAId, teamBId)
  const teamA: Side = { id: teamAId, name: teamAName, color: colors.home }
  const teamB: Side = { id: teamBId, name: teamBName, color: colors.away }

  return (
    <Paper sx={{ p: { xs: 1.5, sm: 2 } }}>
      <Typography variant="subtitle1" sx={{ mb: 1 }}>
        {t('headToHead.title')}
      </Typography>

      {/* Team names appear once, here; everything below refers to them by side and badge. */}
      <Box sx={{ display: 'grid', gridTemplateColumns: '1fr 1fr', columnGap: 1, alignItems: 'center', mb: 1 }}>
        <TeamLegend team={teamA} align="left" />
        <TeamLegend team={teamB} align="right" />
      </Box>

      <ResultBar aWins={data.teamAWins} draws={data.draws} bWins={data.teamBWins} aColor={teamA.color} bColor={teamB.color} />

      <Box sx={{ display: 'grid', gridTemplateColumns: '1fr auto 1fr', alignItems: 'baseline', rowGap: 0.25, mt: 1 }}>
        <SummaryRow aValue={data.teamAWins} label={t('headToHead.wins')} bValue={data.teamBWins} />
        <SummaryRow aValue={data.teamAGoals} label={t('headToHead.goalsLabel')} bValue={data.teamBGoals} />
      </Box>
      <Typography variant="caption" color="text.secondary" sx={{ display: 'block', textAlign: 'center', mt: 0.5 }}>
        {t('headToHead.drawsCount', { count: data.draws })}
      </Typography>

      <Box sx={{ borderTop: 1, borderColor: 'divider', mt: 1.5, pt: 0.5 }}>
        {data.meetings.map((meeting) => (
          <MeetingRow key={meeting.matchId} meeting={meeting} teamA={teamA} teamB={teamB} />
        ))}
      </Box>
    </Paper>
  )
}

function TeamLegend({ team, align }: { team: Side; align: 'left' | 'right' }) {
  const badge = <TeamBadge teamId={team.id} teamName={team.name} color={team.color} size={28} />
  const name = (
    <Typography variant="body2" noWrap sx={{ minWidth: 0, fontWeight: 600, textAlign: align }}>
      <TeamNameLink teamId={team.id}>{team.name}</TeamNameLink>
    </Typography>
  )
  return (
    <Stack direction="row" spacing={1} sx={{ alignItems: 'center', minWidth: 0, justifyContent: align === 'left' ? 'flex-start' : 'flex-end' }}>
      {align === 'left' ? (
        <>
          {badge}
          {name}
        </>
      ) : (
        <>
          {name}
          {badge}
        </>
      )}
    </Stack>
  )
}

// Wins / draws / wins as one proportional strip in the teams' colors - the figures underneath carry
// the exact numbers, so the bar itself is decorative.
function ResultBar({ aWins, draws, bWins, aColor, bColor }: { aWins: number; draws: number; bWins: number; aColor: string; bColor: string }) {
  const segments = [
    { key: 'a', value: aWins, color: aColor },
    { key: 'draws', value: draws, color: 'action.disabled' },
    { key: 'b', value: bWins, color: bColor },
  ].filter((segment) => segment.value > 0)
  return (
    <Box aria-hidden sx={{ display: 'flex', gap: '2px', height: 6, borderRadius: 3, overflow: 'hidden' }}>
      {segments.map((segment) => (
        <Box key={segment.key} sx={{ flex: segment.value, bgcolor: segment.color }} />
      ))}
    </Box>
  )
}

function SummaryRow({ aValue, label, bValue }: { aValue: number; label: string; bValue: number }) {
  const valueSx = { fontWeight: 700, textAlign: 'center', lineHeight: 1.3, ...TABULAR } as const
  return (
    <>
      <Typography variant="h6" sx={valueSx}>
        {aValue}
      </Typography>
      <Typography variant="caption" color="text.secondary" sx={{ textAlign: 'center', px: 1 }}>
        {label}
      </Typography>
      <Typography variant="h6" sx={valueSx}>
        {bValue}
      </Typography>
    </>
  )
}

// One past meeting as a scoreboard line: home always on the left, away on the right, each side
// identified by its badge rather than its name (the legend above maps badges to names). The winner's
// goals are bold; a draw leaves both plain.
function MeetingRow({ meeting, teamA, teamB }: { meeting: HeadToHeadMeeting; teamA: Side; teamB: Side }) {
  const { t } = useTranslation()
  const homeIsA = meeting.homeTeamId === teamA.id
  const home = homeIsA ? teamA : teamB
  const away = homeIsA ? teamB : teamA
  const date = formatDate(meeting.matchDate)
  const homeWon = meeting.homeGoals > meeting.awayGoals
  const awayWon = meeting.awayGoals > meeting.homeGoals
  const goalSx = (won: boolean, lost: boolean) => ({
    fontWeight: won ? 800 : 400,
    color: lost ? 'text.secondary' : 'text.primary',
    ...TABULAR,
  })

  return (
    <Box
      component={RouterLink}
      to={`/match/${meeting.matchId}`}
      aria-label={t('headToHead.meetingLabel', {
        home: home.name,
        homeGoals: meeting.homeGoals,
        awayGoals: meeting.awayGoals,
        away: away.name,
        date,
      })}
      sx={{
        display: 'grid',
        gridTemplateColumns: `${SIDE_COLUMN}px 1fr auto 1fr ${SIDE_COLUMN}px`,
        columnGap: 1,
        alignItems: 'center',
        minHeight: 44,
        mx: -1,
        px: 1,
        borderRadius: 1,
        color: 'inherit',
        textDecoration: 'none',
        '&:hover': { bgcolor: 'action.hover' },
      }}
    >
      <Typography variant="caption" color="text.secondary" sx={TABULAR}>
        {date}
      </Typography>
      <Box sx={{ justifySelf: 'end' }}>
        <TeamBadge teamId={home.id} teamName={home.name} color={home.color} size={22} />
      </Box>
      <Box sx={{ display: 'grid', gridTemplateColumns: '2ch auto 2ch', columnGap: 0.75, alignItems: 'baseline' }}>
        <Typography variant="body1" sx={{ textAlign: 'right', ...goalSx(homeWon, awayWon) }}>
          {meeting.homeGoals}
        </Typography>
        <Typography variant="body1" color="text.secondary">
          –
        </Typography>
        <Typography variant="body1" sx={{ textAlign: 'left', ...goalSx(awayWon, homeWon) }}>
          {meeting.awayGoals}
        </Typography>
      </Box>
      <Box sx={{ justifySelf: 'start' }}>
        <TeamBadge teamId={away.id} teamName={away.name} color={away.color} size={22} />
      </Box>
      <ChevronRightIcon fontSize="small" sx={{ justifySelf: 'end', color: 'text.secondary' }} />
    </Box>
  )
}
