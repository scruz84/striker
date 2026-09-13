import { useEffect, useState, type SyntheticEvent } from 'react'
import Box from '@mui/material/Box'
import Grid from '@mui/material/Grid'
import Paper from '@mui/material/Paper'
import Skeleton from '@mui/material/Skeleton'
import Tab from '@mui/material/Tab'
import Tabs from '@mui/material/Tabs'
import Typography from '@mui/material/Typography'
import { useQueryClient } from '@tanstack/react-query'
import { useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { queryKeys, useMatchDetails } from '../../api/hooks'
import { POSITION_GOALKEEPER, type WeekMatch } from '../../api/types'
import { useLiveMatch } from '../../api/useLiveMatch'
import { formatDateTime } from '../../utils/formatDate'
import { HeadToHeadCard } from '../../components/HeadToHeadCard'
import { MatchHeadToHeadPitch } from './MatchHeadToHeadPitch'
import { MatchPositionComparison } from './MatchPositionComparison'
import { Events, Scoreboard } from './MatchScoreboard'

export function MatchDetailsPage() {
  const { id } = useParams<{ id: string }>()
  const { data: match, isLoading } = useMatchDetails(id)

  if (isLoading || !match) {
    return <Skeleton variant="rectangular" height={200} />
  }

  if (!match.played) {
    return (
      <LiveMatch
        key={id}
        matchId={id}
        homeId={match.home}
        homeName={match.homeName}
        awayId={match.away}
        awayName={match.awayName}
      />
    )
  }

  return <PlayedMatch match={match} />
}

function LiveMatch({
  matchId,
  homeId,
  homeName,
  awayId,
  awayName,
}: {
  matchId: string | undefined
  homeId: number
  homeName: string
  awayId: number
  awayName: string
}) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const live = useLiveMatch(matchId ? `/v1/teams/match/${matchId}/live` : undefined, !!matchId)

  useEffect(() => {
    if (live.complete && matchId) {
      void queryClient.invalidateQueries({ queryKey: queryKeys.match(matchId) })
    }
  }, [live.complete, matchId, queryClient])

  if (live.complete) {
    return <Typography color="text.secondary">{t('match.matchFinished')}</Typography>
  }

  return (
    <Box>
      <Scoreboard
        homeId={homeId}
        homeName={homeName}
        awayId={awayId}
        awayName={awayName}
        homeGoals={live.homeGoals}
        awayGoals={live.awayGoals}
        status={live.connected ? 'live' : 'connecting'}
        minute={live.minute}
      />
      <Paper variant="outlined" sx={{ p: 2, mb: 2 }}>
        <Events events={live.events} homeId={homeId} homeName={homeName} awayName={awayName} />
      </Paper>
      <HeadToHeadCard teamAId={homeId} teamAName={homeName} teamBId={awayId} teamBName={awayName} />
    </Box>
  )
}

function PlayedMatch({ match }: { match: WeekMatch }) {
  const { t } = useTranslation()
  const [tab, setTab] = useState(0)
  const handleChange = (_event: SyntheticEvent, newValue: number) => setTab(newValue)

  return (
    <Box>
      <Scoreboard
        homeId={match.home}
        homeName={match.homeName}
        awayId={match.away}
        awayName={match.awayName}
        homeGoals={match.homeGoals}
        awayGoals={match.awayGoals}
        status="played"
        subtitle={formatDateTime(match.matchDate)}
      />

      <Paper variant="outlined" sx={{ mb: 2 }}>
        <Tabs value={tab} onChange={handleChange}>
          <Tab label={t('match.tabs.events')} />
          <Tab label={t('match.tabs.lineups')} />
        </Tabs>
        <Box sx={{ p: 2 }}>
          {tab === 0 && (
            <Events events={match.events ?? []} homeId={match.home} homeName={match.homeName} awayName={match.awayName} />
          )}
          {tab === 1 && (
            <Lineups
              homeName={match.homeName}
              awayName={match.awayName}
              homeLineup={match.homeLineup}
              awayLineup={match.awayLineup}
              homePlayers={match.homePlayers}
              awayPlayers={match.awayPlayers}
              matchInstants={match.matchInstants}
            />
          )}
        </Box>
      </Paper>
      <HeadToHeadCard teamAId={match.home} teamAName={match.homeName} teamBId={match.away} teamBName={match.awayName} />
    </Box>
  )
}

interface LineupsProps {
  homeName: string
  awayName: string
  homeLineup: WeekMatch['homeLineup']
  awayLineup: WeekMatch['awayLineup']
  homePlayers: WeekMatch['homePlayers']
  awayPlayers: WeekMatch['awayPlayers']
  matchInstants: WeekMatch['matchInstants']
}

// The starting goalkeeper's own rating - not part of TeamQualification (see its own comment),
// but this is exactly what engine.processChance reads live for goalkeeping strength.
function goalkeeperRating(lineup: WeekMatch['homeLineup'], players: WeekMatch['homePlayers']): number {
  const gk = lineup?.positions.find((p) => p.position === POSITION_GOALKEEPER)
  if (!gk || !players) return 0
  return players[String(gk.playerId)]?.goalKeeper ?? 0
}

function Lineups({ homeName, awayName, homeLineup, awayLineup, homePlayers, awayPlayers, matchInstants }: LineupsProps) {
  // The kickoff instant (minute 1) reflects the same starting XI the pitch below shows - later
  // instants would drift as the match itself changes who's on the pitch (cards, injuries, subs).
  const kickoff = matchInstants?.[0]

  return (
    <Grid container spacing={3}>
      {kickoff && (
        <Grid size={{ xs: 12, md: 5 }}>
          <MatchPositionComparison
            homeQua={kickoff.homeQua}
            awayQua={kickoff.awayQua}
            homeGoalkeeperRating={goalkeeperRating(homeLineup, homePlayers)}
            awayGoalkeeperRating={goalkeeperRating(awayLineup, awayPlayers)}
            homeName={homeName}
            awayName={awayName}
          />
        </Grid>
      )}
      <Grid size={{ xs: 12, md: kickoff ? 7 : 12 }}>
        <Box sx={{ maxWidth: 380, mx: 'auto' }}>
          <Typography variant="subtitle1" gutterBottom sx={{ textAlign: 'center' }}>
            {awayName}
          </Typography>
          <MatchHeadToHeadPitch
            homeLineup={homeLineup}
            homePlayers={homePlayers}
            awayLineup={awayLineup}
            awayPlayers={awayPlayers}
          />
          <Typography variant="subtitle1" sx={{ textAlign: 'center' }}>
            {homeName}
          </Typography>
        </Box>
      </Grid>
    </Grid>
  )
}
