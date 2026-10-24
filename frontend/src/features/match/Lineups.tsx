import Box from '@mui/material/Box'
import Grid from '@mui/material/Grid'
import Typography from '@mui/material/Typography'
import { POSITION_GOALKEEPER, type Lineup, type Player, type MatchInstant } from '../../api/types'
import { TeamNameLink } from '../../components/TeamNameLink'
import { MatchHeadToHeadPitch } from './MatchHeadToHeadPitch'
import { MatchMinuteZoneComparison } from './MatchMinuteZoneComparison'

export interface LineupsProps {
  homeId: number
  homeName: string
  awayId: number
  awayName: string
  homeLineup: Lineup | null
  awayLineup: Lineup | null
  homePlayers: Record<string, Player> | null
  awayPlayers: Record<string, Player> | null
  matchInstants: MatchInstant[] | null
  ratings?: Record<string, number>
}

// The starting goalkeeper's own rating - not part of TeamQualification (see its own comment),
// but this is exactly what engine.processChance reads live for goalkeeping strength. Both the live
// and the finished view of the match viewer pass it to the zone duel, so a minute shows the same
// figures on either side of the full-time switch.
export function goalkeeperRating(lineup: Lineup | null, players: Record<string, Player> | null): number {
  const gk = lineup?.positions.find((p) => p.position === POSITION_GOALKEEPER)
  if (!gk || !players) return 0
  return players[String(gk.playerId)]?.goalKeeper ?? 0
}

// The finished match's "Tácticas" tab, shared by every competition: the minute-scrubbable zone duel
// when the match has stored per-minute instants, and the lineups pitch.
export function Lineups({
  homeId,
  homeName,
  awayId,
  awayName,
  homeLineup,
  awayLineup,
  homePlayers,
  awayPlayers,
  matchInstants,
  ratings,
}: LineupsProps) {
  // The full per-minute history (see backend common.MatchInstant) lets a viewer scrub the
  // zone-battle panel through the whole match, the same way MatchMinuteRatings already lets them
  // scrub player ratings - not just the kickoff snapshot this used to be frozen at. A match without
  // stored instants (one played before they were stored) simply skips this panel and shows only the
  // pitch.
  const zoneHistory = matchInstants && matchInstants.length > 0 ? matchInstants : null

  return (
    <Grid container spacing={3}>
      {zoneHistory && (
        <Grid size={{ xs: 12, md: 5 }}>
          <MatchMinuteZoneComparison
            homeId={homeId}
            homeName={homeName}
            awayId={awayId}
            awayName={awayName}
            homeGoalkeeperRating={goalkeeperRating(homeLineup, homePlayers)}
            awayGoalkeeperRating={goalkeeperRating(awayLineup, awayPlayers)}
            matchInstants={zoneHistory}
            currentMinute={zoneHistory[zoneHistory.length - 1].minute}
          />
        </Grid>
      )}
      <Grid size={{ xs: 12, md: zoneHistory ? 7 : 12 }}>
        <Box sx={{ maxWidth: 380, mx: 'auto' }}>
          <Typography variant="subtitle1" gutterBottom sx={{ textAlign: 'center' }}>
            <TeamNameLink teamId={awayId}>{awayName}</TeamNameLink>
          </Typography>
          <MatchHeadToHeadPitch
            homeLineup={homeLineup}
            homePlayers={homePlayers}
            awayLineup={awayLineup}
            awayPlayers={awayPlayers}
            ratings={ratings}
          />
          <Typography variant="subtitle1" sx={{ textAlign: 'center' }}>
            <TeamNameLink teamId={homeId}>{homeName}</TeamNameLink>
          </Typography>
        </Box>
      </Grid>
    </Grid>
  )
}
