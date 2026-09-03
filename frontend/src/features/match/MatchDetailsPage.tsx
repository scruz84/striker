import { useState, type SyntheticEvent } from 'react'
import Box from '@mui/material/Box'
import Divider from '@mui/material/Divider'
import Grid from '@mui/material/Grid'
import Paper from '@mui/material/Paper'
import Skeleton from '@mui/material/Skeleton'
import Tab from '@mui/material/Tab'
import Tabs from '@mui/material/Tabs'
import Typography from '@mui/material/Typography'
import { useParams } from 'react-router-dom'
import { useMatchDetails } from '../../api/hooks'
import {
  EVENT_TYPE_GOAL,
  EVENT_TYPE_INJURY,
  EVENT_TYPE_RED_CARD,
  EVENT_TYPE_YELLOW_CARD,
  POSITION_LABELS,
  type Lineup,
  type MatchEvent,
  type Player,
  type WeekMatch,
} from '../../api/types'
import { formatDateTime } from '../../utils/formatDate'

export function MatchDetailsPage() {
  const { id } = useParams<{ id: string }>()
  const { data: match, isLoading } = useMatchDetails(id)

  if (isLoading || !match) {
    return <Skeleton variant="rectangular" height={200} />
  }

  if (!match.played) {
    return <Typography>This match hasn't been played yet.</Typography>
  }

  return <PlayedMatch match={match} />
}

function PlayedMatch({ match }: { match: WeekMatch }) {
  const [tab, setTab] = useState(0)
  const handleChange = (_event: SyntheticEvent, newValue: number) => setTab(newValue)

  return (
    <Box>
      <Paper sx={{ p: 2, mb: 2 }}>
        <Typography variant="caption" color="text.secondary">
          {formatDateTime(match.matchDate)}
        </Typography>
        <Grid container sx={{ mt: 1, alignItems: 'center' }}>
          <Grid size={5}>
            <Typography variant="h6">{match.homeName}</Typography>
          </Grid>
          <Grid size={2}>
            <Typography variant="h6" align="center">
              {match.homeGoals} - {match.awayGoals}
            </Typography>
          </Grid>
          <Grid size={5}>
            <Typography variant="h6" align="right">
              {match.awayName}
            </Typography>
          </Grid>
        </Grid>
      </Paper>

      <Paper>
        <Tabs value={tab} onChange={handleChange}>
          <Tab label="Events" />
          <Tab label="Lineups" />
        </Tabs>
        <Box sx={{ p: 2 }}>
          {tab === 0 && <Events events={match.events ?? []} />}
          {tab === 1 && (
            <Lineups
              homeName={match.homeName}
              awayName={match.awayName}
              homeLineup={match.homeLineup}
              awayLineup={match.awayLineup}
              homePlayers={match.homePlayers}
              awayPlayers={match.awayPlayers}
            />
          )}
        </Box>
      </Paper>
    </Box>
  )
}

const NOTABLE_EVENT_TYPES: number[] = [EVENT_TYPE_GOAL, EVENT_TYPE_YELLOW_CARD, EVENT_TYPE_RED_CARD, EVENT_TYPE_INJURY]

function Events({ events }: { events: WeekMatch['events'] }) {
  const notable = (events ?? [])
    .filter((event) => NOTABLE_EVENT_TYPES.includes(event.eventType))
    .sort((a, b) => a.minute - b.minute)

  if (notable.length === 0) {
    return <Typography color="text.secondary">No notable events in this match.</Typography>
  }

  return (
    <Box>
      {notable.map((event, idx) => (
        <Grid container key={idx} spacing={1} sx={{ py: 0.5, alignItems: 'center' }}>
          <Grid size={2}>
            <EventMarker event={event} />
          </Grid>
          <Grid size={1}>
            <Divider orientation="vertical" />
          </Grid>
          <Grid size={7}>
            <Typography variant="body2">{(event.teamOwnerPlayerNames ?? []).join(', ')}</Typography>
          </Grid>
          <Grid size={2}>
            <Typography variant="body2" align="right">
              {event.minute}&apos;
            </Typography>
          </Grid>
        </Grid>
      ))}
    </Box>
  )
}

function EventMarker({ event }: { event: MatchEvent }) {
  switch (event.eventType) {
    case EVENT_TYPE_GOAL:
      return (
        <Typography variant="body2">
          {event.homeGoals} - {event.awayGoals}
        </Typography>
      )
    case EVENT_TYPE_YELLOW_CARD:
      return (
        <Typography variant="body2" component="span" aria-label="Yellow card" title="Yellow card">
          🟨
        </Typography>
      )
    case EVENT_TYPE_RED_CARD:
      return (
        <Typography variant="body2" component="span" aria-label="Red card" title="Red card">
          🟥
        </Typography>
      )
    case EVENT_TYPE_INJURY:
      return (
        <Typography variant="body2" component="span" aria-label="Injury" title="Injury">
          🚑
        </Typography>
      )
    default:
      return null
  }
}

interface LineupsProps {
  homeName: string
  awayName: string
  homeLineup: Lineup | null
  awayLineup: Lineup | null
  homePlayers: Record<string, Player> | null
  awayPlayers: Record<string, Player> | null
}

function Lineups({ homeName, awayName, homeLineup, awayLineup, homePlayers, awayPlayers }: LineupsProps) {
  return (
    <Grid container spacing={2}>
      <Grid size={{ xs: 12, sm: 6 }}>
        <Typography variant="subtitle1" gutterBottom>
          {homeName}
        </Typography>
        <TeamLineup lineup={homeLineup} players={homePlayers} />
      </Grid>
      <Grid size={{ xs: 12, sm: 6 }}>
        <Typography variant="subtitle1" gutterBottom>
          {awayName}
        </Typography>
        <TeamLineup lineup={awayLineup} players={awayPlayers} />
      </Grid>
    </Grid>
  )
}

function TeamLineup({ lineup, players }: { lineup: Lineup | null; players: Record<string, Player> | null }) {
  if (!lineup || !players || lineup.positions.length === 0) {
    return <Typography color="text.secondary">No lineup data.</Typography>
  }

  const sorted = [...lineup.positions].sort((a, b) => a.position - b.position)

  return (
    <Box>
      {sorted.map((pos) => {
        const player = players[String(pos.playerId)]
        return (
          <Typography variant="body2" key={pos.playerId}>
            {POSITION_LABELS[pos.position] ?? pos.position}:{' '}
            {player ? `${player.firstName} ${player.lastName}` : `#${pos.playerId}`}
          </Typography>
        )
      })}
    </Box>
  )
}
