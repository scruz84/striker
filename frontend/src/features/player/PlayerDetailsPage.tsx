import Chip from '@mui/material/Chip'
import Grid from '@mui/material/Grid'
import Paper from '@mui/material/Paper'
import Skeleton from '@mui/material/Skeleton'
import Stack from '@mui/material/Stack'
import Table from '@mui/material/Table'
import TableBody from '@mui/material/TableBody'
import TableCell from '@mui/material/TableCell'
import TableContainer from '@mui/material/TableContainer'
import TableHead from '@mui/material/TableHead'
import TableRow from '@mui/material/TableRow'
import Typography from '@mui/material/Typography'
import { Link as RouterLink, useParams, useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { usePlayerDetails, useUserTeam } from '../../api/hooks'
import type { NationalTeamCallUp, PlayerMatchSummary } from '../../api/types'
import { localizedTeamName } from '../../utils/nationalTeamName'
import { PlayerAttributesPanel } from '../../components/PlayerAttributesPanel'
import { PlayerConditionPanel } from '../../components/PlayerConditionPanel'
import { PlayerEvolutionCard } from '../../components/PlayerEvolutionCard'
import { PlayerProfileRail } from '../../components/PlayerProfileRail'
import { TeamNameLink } from '../../components/TeamNameLink'
import { formatDateTime, formatDayTime } from '../../utils/formatDate'
import { matchRatingColor } from '../../utils/matchRating'
import { PlayerTransferHistory } from './PlayerTransferHistory'

export function PlayerDetailsPage() {
  const { t } = useTranslation()
  const { id } = useParams<{ id: string }>()
  const [searchParams] = useSearchParams()
  const { data: player, isLoading } = usePlayerDetails(id)
  const { data: userTeam } = useUserTeam(true)

  if (isLoading || !player) {
    return <Skeleton variant="rectangular" height={300} />
  }

  // A rostered player's contract length and financial history are the owning manager's business
  // - the backend already redacts them for anyone else (see common.Player.RedactFinancials), so
  // this decides whether to render the transaction history table below (PlayerProfileRail
  // handles the contract-length spec row itself). Wage stays visible regardless - the backend
  // keeps it populated even for someone else's rostered player. A free agent (teamId 0) has no
  // current team to protect that information for, same reasoning as the transfer market listing.
  const canSeeFinancials = player.teamId === 0 || userTeam?.team.id === player.teamId
  // A rostered player's skill ratings are likewise the owning manager's business, unless this
  // specific player is listed on the transfer market - the backend already zeroes the underlying
  // fields for anyone else (see common.Player.RedactSkills), so this just decides whether the
  // attribute panel/traits are worth rendering at all rather than showing an all-zero radar chart.
  const canSeeSkills = canSeeFinancials || player.transferListingAskingPrice !== undefined

  return (
    <Stack spacing={{ xs: 1.5, sm: 2 }}>
      <Grid container spacing={{ xs: 1.5, sm: 2 }}>
        <Grid size={{ xs: 12, md: 4 }}>
          <PlayerProfileRail
            player={player}
            canSeeFinancials={canSeeFinancials}
            canSeeSkills={canSeeSkills}
            autoOpenOffer={searchParams.get('renegotiate') === '1'}
          />
        </Grid>
        {canSeeSkills && (
          <Grid size={{ xs: 12, md: 8 }}>
            <Stack spacing={{ xs: 1.5, sm: 2 }}>
              <PlayerAttributesPanel player={player} />
              <PlayerConditionPanel player={player} />
            </Stack>
          </Grid>
        )}
      </Grid>

      {canSeeSkills && player.latestEvolution && <PlayerEvolutionCard evolution={player.latestEvolution} />}

      {player.clubHistory.length > 0 && (
        <Stack spacing={1}>
          <Typography variant="h6">{t('playerPage.clubHistory.title')}</Typography>
          <TableContainer component={Paper}>
            <Table size="small" aria-label="Club history">
              <TableHead>
                <TableRow>
                  <TableCell>{t('playerPage.clubHistory.club')}</TableCell>
                  <TableCell align="right">{t('playerPage.clubHistory.joined')}</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {player.clubHistory.map((entry) => (
                  <TableRow key={`${entry.teamId}-${entry.joinedAt}`}>
                    <TableCell>
                      <TeamNameLink teamId={entry.teamId}>{entry.teamName}</TeamNameLink>
                    </TableCell>
                    <TableCell align="right" sx={{ whiteSpace: 'nowrap' }}>
                      {formatDateTime(entry.joinedAt)}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        </Stack>
      )}

      {player.teamId !== 0 && (
        <Stack spacing={1}>
          <Typography variant="h6">{t('playerPage.recentMatches.title')}</Typography>
          <TableContainer component={Paper}>
            <Table size="small" aria-label="Recent matches">
              <TableHead>
                <TableRow>
                  <TableCell>{t('playerPage.recentMatches.date')}</TableCell>
                  <TableCell>{t('playerPage.recentMatches.opponent')}</TableCell>
                  <TableCell align="right">{t('playerPage.recentMatches.result')}</TableCell>
                  <TableCell align="right">{t('playerPage.recentMatches.rating')}</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {player.recentMatches.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={4}>
                      <Typography variant="body2" color="text.secondary">
                        {t('playerPage.recentMatches.none')}
                      </Typography>
                    </TableCell>
                  </TableRow>
                )}
                {player.recentMatches.map((match) => (
                  <RecentMatchRow key={match.matchId} match={match} />
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        </Stack>
      )}

      {player.recentCallUps.length > 0 && (
        <Stack spacing={1}>
          <Typography variant="h6">{t('playerPage.callUps.title')}</Typography>
          <TableContainer component={Paper}>
            <Table size="small" aria-label="National team call-ups">
              <TableHead>
                <TableRow>
                  <TableCell>{t('playerPage.callUps.date')}</TableCell>
                  <TableCell>{t('playerPage.callUps.fixture')}</TableCell>
                  <TableCell align="right">{t('playerPage.callUps.result')}</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {player.recentCallUps.map((callUp) => (
                  <CallUpRow key={`${callUp.kind}-${callUp.matchId}`} callUp={callUp} />
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        </Stack>
      )}

      <PlayerTransferHistory transfers={player.transfers} />
    </Stack>
  )
}

// Rows link to the fixture itself - a friendly's own detail page, or (for a World Cup knockout
// tie) the cup tie page - rather than a static read-only row, so a manager can jump straight from
// "my player got called up" to the match.
function CallUpRow({ callUp }: { callUp: NationalTeamCallUp }) {
  const { t } = useTranslation()
  const to = callUp.kind === 'friendly' ? `/friendlies/${callUp.matchId}` : `/cup/ties/${callUp.matchId}`
  return (
    <TableRow component={RouterLink} to={to} hover sx={{ cursor: 'pointer', '& td': { border: 0 } }}>
      <TableCell>{formatDayTime(callUp.matchDate)}</TableCell>
      <TableCell>
        {localizedTeamName(t, callUp.teamName)} {t('playerPage.callUps.vs')} {localizedTeamName(t, callUp.opponentName)}
      </TableCell>
      <TableCell align="right">
        {callUp.played ? `${callUp.goalsFor} - ${callUp.goalsAgainst}` : <Chip size="small" label={t('playerPage.callUps.upcoming')} />}
      </TableCell>
    </TableRow>
  )
}

// Rows link through to the match report - the same place the "Calificaciones" tab that this
// rating comes from lives (see MatchRatingsTab/matchRatingColor) - so a manager can jump straight
// from "how has this player been doing" to the full match detail.
function RecentMatchRow({ match }: { match: PlayerMatchSummary }) {
  const { t } = useTranslation()
  return (
    <TableRow component={RouterLink} to={`/match/${match.matchId}`} hover sx={{ cursor: 'pointer', '& td': { border: 0 } }}>
      <TableCell>{formatDayTime(match.matchDate)}</TableCell>
      <TableCell>
        <Chip
          size="small"
          variant="outlined"
          label={t(match.isHome ? 'playerPage.recentMatches.homeTag' : 'playerPage.recentMatches.awayTag')}
          sx={{ height: 18, mr: 0.75, fontSize: '0.65rem', '& .MuiChip-label': { px: 0.5 } }}
        />
        {match.opponentName}
        {(match.goals > 0 || match.yellowCards > 0 || match.redCards > 0) && (
          <Typography component="span" variant="caption" color="text.secondary" sx={{ ml: 1 }}>
            {match.goals > 0 && `⚽×${match.goals}`}
            {match.yellowCards > 0 && ' 🟨'}
            {match.redCards > 0 && ' 🟥'}
          </Typography>
        )}
      </TableCell>
      <TableCell align="right">
        {match.homeGoals} - {match.awayGoals}
      </TableCell>
      <TableCell align="right">
        {match.rating !== undefined ? <Chip size="small" color={matchRatingColor(match.rating)} label={match.rating.toFixed(1)} /> : '—'}
      </TableCell>
    </TableRow>
  )
}
