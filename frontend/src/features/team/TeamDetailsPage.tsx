import { useState } from 'react'
import ArrowDownwardIcon from '@mui/icons-material/ArrowDownward'
import ArrowUpwardIcon from '@mui/icons-material/ArrowUpward'
import Alert from '@mui/material/Alert'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import Chip from '@mui/material/Chip'
import FormControl from '@mui/material/FormControl'
import IconButton from '@mui/material/IconButton'
import InputLabel from '@mui/material/InputLabel'
import Link from '@mui/material/Link'
import MenuItem from '@mui/material/MenuItem'
import Paper from '@mui/material/Paper'
import Select, { type SelectChangeEvent } from '@mui/material/Select'
import Skeleton from '@mui/material/Skeleton'
import Stack from '@mui/material/Stack'
import Table from '@mui/material/Table'
import TableBody from '@mui/material/TableBody'
import TableCell from '@mui/material/TableCell'
import TableContainer from '@mui/material/TableContainer'
import TableHead from '@mui/material/TableHead'
import TableRow from '@mui/material/TableRow'
import Tooltip from '@mui/material/Tooltip'
import Typography from '@mui/material/Typography'
import { useTheme } from '@mui/material/styles'
import { Link as RouterLink, useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  useAcademy,
  useCupDetails,
  useLatestCupId,
  useLatestInternationalCupId,
  useMe,
  usePromoteFromAcademy,
  useReleasePlayer,
  useTeamDetails,
  useTeamHistory,
  useTeamMatches,
  useTeamTrainer,
  useUserTeam,
} from '../../api/hooks'
import {
  COUNTRY_NAME_KEYS,
  MAX_CONTRACT_YEARS_REMAINING,
  type CupDetails,
  type Player,
  type PlayerUnavailability,
  type SeasonHistoryEntry,
  type StadiumSummary,
} from '../../api/types'
import { CartoonAvatar } from '../../components/avatar/CartoonAvatar'
import { regionForCountry } from '../../components/avatar/traits'
import { useConfirm } from '../../components/confirmDialogContext'
import { ChallengeToFriendlyDialog } from '../../components/ChallengeToFriendlyDialog'
import { ContractRenewalDialog } from '../../components/ContractRenewalDialog'
import { TransferOfferDialog } from '../../components/TransferOfferDialog'
import { HeadToHeadCard } from '../../components/HeadToHeadCard'
import { SeasonPositionChart } from '../../components/SeasonPositionChart'
import { TeamNameLink } from '../../components/TeamNameLink'
import { PlayerChip } from '../lineup/PlayerChip'
import { UnavailabilityBadge } from '../lineup/UnavailabilityBadge'
import { useSquadUnavailability } from '../lineup/useSquadUnavailability'
import { useSquadTransferListed } from '../lineup/useSquadTransferListed'
import { TransferListedBadge } from '../lineup/TransferListedBadge'
import { isPast } from '../../api/useLiveMatch'
import { MatchesList } from './TeamMatchesSection'
import { countryFlag } from '../../utils/countryFlag'
import { formatCurrency } from '../../utils/formatCurrency'
import { formatDate, formatDateTime } from '../../utils/formatDate'
import { overallRating, ratingTierColor } from '../../utils/playerRating'

const RECENT_MATCHES_LIMIT = 5

type SortDirection = 'asc' | 'desc'
type SquadSortField = 'name' | 'age' | 'wage' | 'skill'

type CupStatus = 'none' | 'champion' | 'active' | 'eliminated'

// Derives whether a team is still alive in a cup from its ties - there's no dedicated
// "still in" field on Team/TeamDetails, so this reuses the same CupDetails.ties data the
// cup bracket pages (CupPage/InternationalCupPage) already fetch and render.
function teamCupStatus(cupDetails: CupDetails | undefined, teamId: number): CupStatus {
  if (!cupDetails) return 'none'
  const { cup, ties } = cupDetails
  const teamTies = ties.filter((tie) => tie.homeTeamId === teamId || tie.awayTeamId === teamId)
  if (teamTies.length === 0) return 'none'
  const eliminated = teamTies.some((tie) => tie.played && tie.winnerTeamId !== null && tie.winnerTeamId !== teamId)
  if (eliminated) return 'eliminated'
  if (cup.completed && cup.championTeamId === teamId) return 'champion'
  return 'active'
}

function CupStatusChip({ label, status }: { label: string; status: CupStatus }) {
  const { t } = useTranslation()
  if (status === 'none') return null

  const color = status === 'champion' ? 'success' : status === 'active' ? 'primary' : 'default'
  const statusLabel =
    status === 'champion' ? t('team.cup.champion') : status === 'active' ? t('team.cup.stillIn') : t('team.cup.eliminated')

  return <Chip size="small" color={color} label={`${label}: ${statusLabel}`} />
}

function compareSquadPlayers(a: Player, b: Player, field: SquadSortField): number {
  switch (field) {
    case 'age':
      return a.age - b.age
    case 'wage':
      return a.wage - b.wage
    case 'skill':
      return (a.averageSkill ?? overallRating(a)) - (b.averageSkill ?? overallRating(b))
    default:
      return `${a.firstName} ${a.lastName}`.localeCompare(`${b.firstName} ${b.lastName}`)
  }
}

export function TeamDetailsPage() {
  const { t } = useTranslation()
  const { id } = useParams<{ id: string }>()
  const { data: loggedUser } = useMe()
  const { data: teamDetails, isLoading } = useTeamDetails(id)
  const { data: userTeam } = useUserTeam(!!loggedUser)
  const { data: history, isLoading: historyLoading } = useTeamHistory(id)
  const unavailability = useSquadUnavailability()
  const transferListed = useSquadTransferListed()
  const [sortField, setSortField] = useState<SquadSortField>('name')
  const [sortDir, setSortDir] = useState<SortDirection>('asc')
  const [challengeOpen, setChallengeOpen] = useState(false)

  const teamCountry = teamDetails?.team.country
  const { data: latestCup } = useLatestCupId(teamCountry)
  const { data: cupDetails } = useCupDetails(latestCup?.cupId)
  const region = teamCountry ? regionForCountry(teamCountry) : undefined
  const { data: latestInternationalCup } = useLatestInternationalCupId(region)
  const { data: internationalCupDetails } = useCupDetails(latestInternationalCup?.cupId)

  if (isLoading || !teamDetails) {
    return <div>{t('common.loading')}</div>
  }

  const isOwnTeam = userTeam?.team.id === teamDetails.team.id
  const flag = countryFlag(teamDetails.team.country)
  const cupStatus = teamCupStatus(cupDetails, teamDetails.team.id)
  const internationalCupStatus = teamCupStatus(internationalCupDetails, teamDetails.team.id)

  const handleSort = (field: SquadSortField) => {
    if (field === sortField) {
      setSortDir((prev) => (prev === 'asc' ? 'desc' : 'asc'))
    } else {
      setSortField(field)
      setSortDir('asc')
    }
  }

  const players = [...teamDetails.players].sort((a, b) => (sortDir === 'asc' ? 1 : -1) * compareSquadPlayers(a, b, sortField))

  return (
    <Stack spacing={1}>
      <Stack direction="row" spacing={1} sx={{ alignItems: 'center', justifyContent: 'space-between', flexWrap: 'wrap' }}>
        <Stack direction="row" spacing={1} sx={{ alignItems: 'center' }}>
          <Typography variant="h4" gutterBottom sx={{ mb: 0 }}>
            {teamDetails.team.name} ({teamDetails.team.id})
          </Typography>
          {flag && (
            <Tooltip title={t(COUNTRY_NAME_KEYS[teamDetails.team.country] ?? teamDetails.team.country)}>
              <Typography component="span" sx={{ fontSize: '1.5rem', lineHeight: 1 }}>
                {flag}
              </Typography>
            </Tooltip>
          )}
        </Stack>
        {!isOwnTeam && userTeam && (
          <Button variant="outlined" size="small" onClick={() => setChallengeOpen(true)}>
            {t('friendlies.challenge.challengeTeam')}
          </Button>
        )}
      </Stack>
      {!isOwnTeam && userTeam && (
        <ChallengeToFriendlyDialog
          opponentTeamId={teamDetails.team.id}
          opponentTeamName={teamDetails.team.name}
          open={challengeOpen}
          onClose={() => setChallengeOpen(false)}
        />
      )}
      <Typography sx={{ fontWeight: 'bold' }}>{t('team.competitions')}</Typography>
      <Stack direction="row" spacing={1} sx={{ alignItems: 'center', flexWrap: 'wrap' }}>
        <Link component={RouterLink} to={`/division/${teamDetails.team.divisionId}`}>
          {teamDetails.team.divisionName}
        </Link>
        <Typography variant="body2" color="text.secondary">
          {t('team.classificationPosition', { position: teamDetails.team.classificationPosition })}
        </Typography>
      </Stack>
      <Stack direction="row" spacing={1} sx={{ flexWrap: 'wrap' }}>
        <CupStatusChip label={t('team.cup.domestic')} status={cupStatus} />
        <CupStatusChip label={t('team.cup.international')} status={internationalCupStatus} />
      </Stack>

      {teamDetails.stadium && (
        <Box sx={{ mt: 1 }}>
          <StadiumCard stadium={teamDetails.stadium} />
        </Box>
      )}

      <Box sx={{ mt: 2 }}>
        <RecentMatchesCard teamId={teamDetails.team.id} />
      </Box>

      <Typography variant="h6" sx={{ mt: 2 }}>
        {t('team.squad')}
      </Typography>
      <Stack direction="row" spacing={1} sx={{ alignItems: 'center' }}>
        <FormControl size="small" sx={{ minWidth: 160 }}>
          <InputLabel id="squad-sort-label">{t('team.sortBy')}</InputLabel>
          <Select
            labelId="squad-sort-label"
            label={t('team.sortBy')}
            value={sortField}
            onChange={(event: SelectChangeEvent) => handleSort(event.target.value as SquadSortField)}
          >
            <MenuItem value="name">{t('team.player')}</MenuItem>
            <MenuItem value="age">{t('team.age')}</MenuItem>
            <MenuItem value="wage">{t('team.wage')}</MenuItem>
            <MenuItem value="skill">{t('team.averageSkill')}</MenuItem>
          </Select>
        </FormControl>
        <IconButton size="small" onClick={() => setSortDir((prev) => (prev === 'asc' ? 'desc' : 'asc'))}>
          {sortDir === 'asc' ? <ArrowUpwardIcon fontSize="small" /> : <ArrowDownwardIcon fontSize="small" />}
        </IconButton>
      </Stack>
      <Stack spacing={1}>
        {players.map((player) => (
          <SquadPlayerCard
            key={player.id}
            player={player}
            isOwnTeam={isOwnTeam}
            unavailability={isOwnTeam ? unavailability.get(player.id) : undefined}
            forSale={isOwnTeam && transferListed.has(player.id)}
          />
        ))}
      </Stack>

      {isOwnTeam && (
        <Box sx={{ mt: 2 }}>
          <CoachCard />
        </Box>
      )}

      {isOwnTeam && (
        <Box sx={{ mt: 2 }}>
          <Typography variant="h6">{t('team.academy.title')}</Typography>
          <AcademySection />
        </Box>
      )}

      <Typography variant="h6" sx={{ mt: 2 }}>
        {t('team.trophyCabinet')}
      </Typography>
      {historyLoading || !history ? (
        <Skeleton variant="rectangular" height={100} />
      ) : (
        <>
          <SeasonPositionChart history={history} />
          <TrophyCabinet history={history} />
        </>
      )}

      {!isOwnTeam && userTeam && (
        <Box sx={{ mt: 2 }}>
          <HeadToHeadCard
            teamAId={userTeam.team.id}
            teamAName={userTeam.team.name}
            teamBId={teamDetails.team.id}
            teamBName={teamDetails.team.name}
          />
        </Box>
      )}
    </Stack>
  )
}

function TrophyCabinet({ history }: { history: SeasonHistoryEntry[] }) {
  const { t } = useTranslation()
  if (history.length === 0) {
    return (
      <Typography variant="body2" color="text.secondary">
        {t('team.noSeasons')}
      </Typography>
    )
  }

  return (
    <TableContainer component={Paper}>
      <Table size="small">
        <TableHead>
          <TableRow>
            <TableCell>{t('team.table.season')}</TableCell>
            <TableCell>{t('team.table.league')}</TableCell>
            <TableCell align="right">{t('team.table.finalPosition')}</TableCell>
            <TableCell>{t('team.table.outcome')}</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {history.map((entry) => (
            <TableRow key={entry.season}>
              <TableCell>{entry.season}</TableCell>
              <TableCell>{entry.leagueName ?? '-'}</TableCell>
              <TableCell align="right">{entry.finalPosition ?? '-'}</TableCell>
              <TableCell>
                <Stack direction="row" spacing={1}>
                  {entry.promoted && <Chip size="small" color="success" label={t('team.promoted')} />}
                  {entry.relegated && <Chip size="small" color="warning" label={t('team.relegated')} />}
                  {entry.cupResult && <Chip size="small" color="primary" label={entry.cupResult} />}
                </Stack>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </TableContainer>
  )
}

// StadiumCard shows this team's stadium name, capacity, and (when it's played one) the most
// recent completed league match there with its attendance - public information, shown on any
// team's page, unlike ticket price and pending-upgrade details, which stay owner-only on
// FinancesPage.
function StadiumCard({ stadium }: { stadium: StadiumSummary }) {
  const { t } = useTranslation()
  const match = stadium.lastHomeMatch

  return (
    <Paper sx={{ p: 2 }}>
      <Typography variant="overline" color="text.secondary" sx={{ display: 'block' }}>
        {t('team.stadium.title')}
      </Typography>
      <Typography variant="subtitle1" sx={{ fontWeight: 600 }}>
        {stadium.name}
      </Typography>
      <Typography variant="body2" color="text.secondary">
        {t('team.stadium.capacity', { capacity: stadium.capacity.toLocaleString() })}
      </Typography>
      {match ? (
        <Box sx={{ mt: 1 }}>
          <Typography variant="caption" color="text.secondary" component="div">
            {t('team.stadium.lastMatch')} · {formatDate(match.matchDate)}
          </Typography>
          <Typography variant="body2">
            {t('team.stadium.lastMatchResult', { homeGoals: match.homeGoals, awayGoals: match.awayGoals })}{' '}
            <TeamNameLink teamId={match.opponentId}>{match.opponentName}</TeamNameLink>
          </Typography>
          <Typography variant="caption" color="text.secondary">
            {t('team.stadium.attendance', { count: match.attendance.toLocaleString() })}
          </Typography>
        </Box>
      ) : (
        <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
          {t('team.stadium.noMatches')}
        </Typography>
      )}
    </Paper>
  )
}

// RecentMatchesCard shows this team's most recently played matches (league, cup, and friendly
// alike) right on any team's page, with a link through to the full fixture/results list - so a
// visitor doesn't have to jump to the "own team" matches page (only reachable via the sidebar for
// the caller's own team) just to see how an arbitrary team has been doing lately.
function RecentMatchesCard({ teamId }: { teamId: number }) {
  const { t } = useTranslation()
  const { data: matches, isLoading } = useTeamMatches(String(teamId))

  if (isLoading || !matches) {
    return <Skeleton variant="rectangular" height={100} />
  }

  const played = matches
    .filter((m) => m.played && isPast(m.matchDate))
    .sort((a, b) => b.matchDate.localeCompare(a.matchDate))
    .slice(0, RECENT_MATCHES_LIMIT)

  return (
    <Stack spacing={0.5}>
      <Stack direction="row" spacing={1} sx={{ alignItems: 'center', justifyContent: 'space-between' }}>
        <Typography variant="h6">{t('team.matches.recent')}</Typography>
        <Link component={RouterLink} to={`/team/${teamId}/matches`} underline="hover">
          {t('team.matches.viewAll')}
        </Link>
      </Stack>
      <MatchesList matches={played} emptyKey="team.matches.noPlayed" />
    </Stack>
  )
}

// CoachCard shows the caller's own team's currently employed head coach (if any) right on the
// team page - hiring/firing itself still only happens from the Coaching tab of the transfer
// market (see TransferMarketPage.CoachingTab), which this links straight to via `?tab=coaching`
// rather than duplicating that flow here (issue #58: a hired coach was otherwise only ever
// visible on the market page, nowhere on the team that actually employs them).
function CoachCard() {
  const { t } = useTranslation()
  const { data: trainer, isLoading } = useTeamTrainer(true)

  if (isLoading) {
    return <Skeleton variant="rectangular" height={72} />
  }

  return (
    <Paper sx={{ p: 2 }}>
      <Typography variant="overline" color="text.secondary" sx={{ display: 'block' }}>
        {t('team.coach.title')}
      </Typography>
      {trainer ? (
        <Stack direction="row" spacing={4} sx={{ alignItems: 'center', mt: 0.5, flexWrap: 'wrap' }}>
          <Typography variant="subtitle1" sx={{ fontWeight: 600 }}>
            {trainer.firstName} {trainer.lastName}
          </Typography>
          <Box>
            <Typography variant="caption" color="text.secondary" component="div">
              {t('transfers.coaching.skill')}
            </Typography>
            <Typography variant="body1">{trainer.skill}</Typography>
          </Box>
          <Box>
            <Typography variant="caption" color="text.secondary" component="div">
              {t('transfers.coaching.wage')}
            </Typography>
            <Typography variant="body1">{formatCurrency(trainer.wage)}</Typography>
          </Box>
          <Link component={RouterLink} to="/transfers?tab=coaching" underline="hover">
            {t('team.coach.manage')}
          </Link>
        </Stack>
      ) : (
        <Stack direction="row" spacing={2} sx={{ alignItems: 'center', mt: 0.5 }}>
          <Typography variant="body2" color="text.secondary">
            {t('team.coach.none')}
          </Typography>
          <Link component={RouterLink} to="/transfers?tab=coaching" underline="hover">
            {t('team.coach.hire')}
          </Link>
        </Stack>
      )}
    </Paper>
  )
}

// AcademySection shows the caller's own team's youth academy ("cantera") pool - up to a few
// young prospects, one of which can be promoted to the senior squad at most once per real
// calendar week (see backend common.PromoteFromAcademy).
function AcademySection() {
  const { t } = useTranslation()
  const { data: academy, isLoading } = useAcademy()
  const promote = usePromoteFromAcademy()

  if (isLoading || !academy) {
    return <Skeleton variant="rectangular" height={100} />
  }

  return (
    <Stack spacing={1}>
      <Typography variant="body2" color="text.secondary">
        {academy.canPromote
          ? t('team.academy.available')
          : t('team.academy.nextAvailable', {
              date: academy.nextAvailableAt ? formatDateTime(new Date(academy.nextAvailableAt).toISOString()) : '',
            })}
      </Typography>
      <TableContainer component={Paper}>
        <Table size="small">
          <TableHead>
            <TableRow>
              <TableCell>{t('team.player')}</TableCell>
              <TableCell align="right">{t('team.age')}</TableCell>
              <TableCell align="right" />
            </TableRow>
          </TableHead>
          <TableBody>
            {academy.players.map((prospect) => (
              <TableRow key={prospect.id}>
                <TableCell>
                  <Stack direction="row" spacing={1.5} sx={{ alignItems: 'center' }}>
                    <CartoonAvatar seed={prospect.id} birthCountry={prospect.birthCountry} size={32} />
                    <Link component={RouterLink} to={`/player/${prospect.id}`} underline="hover" color="inherit">
                      {prospect.firstName} {prospect.lastName}
                    </Link>
                  </Stack>
                </TableCell>
                <TableCell align="right">{prospect.age}</TableCell>
                <TableCell align="right">
                  <Button
                    size="small"
                    variant="contained"
                    disabled={!academy.canPromote || promote.isPending}
                    onClick={() => promote.mutate(prospect.id)}
                  >
                    {t('team.academy.promote')}
                  </Button>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>
      {promote.isError && (
        <Alert severity="error" sx={{ mt: 1 }}>
          {t('team.academy.promoteError')}
        </Alert>
      )}
    </Stack>
  )
}

// The caller's own squad gets the full lineup-bench card (PlayerChip), with the per-skill/
// condition breakdown and renew/release controls as its footer. A rival's squad stays a plain
// summary row instead - common.AverageSkill (see api/types.ts) is the only rating the backend
// sends for someone else's rostered, unlisted player, with the seven individual skills and
// Stamina/Form all zeroed out, so PlayerChip's skill chips would show a misleading wall of zeros
// rather than actually-hidden data.
function SquadPlayerCard({
  player,
  isOwnTeam,
  unavailability,
  forSale,
}: {
  player: Player
  isOwnTeam: boolean
  unavailability?: PlayerUnavailability
  forSale?: boolean
}) {
  const { t } = useTranslation()
  const theme = useTheme()
  const confirm = useConfirm()
  const releasePlayer = useReleasePlayer()
  const [renewalOpen, setRenewalOpen] = useState(false)
  const [offerOpen, setOfferOpen] = useState(false)

  const handleRelease = async () => {
    const ok = await confirm({
      message: t('team.confirmRelease', { name: `${player.firstName} ${player.lastName}` }),
      confirmLabel: t('team.releasePlayer'),
      destructive: true,
    })
    if (ok) {
      releasePlayer.mutate(player.id)
    }
  }

  if (isOwnTeam) {
    const footer = (
      <Stack spacing={0.5}>
        <Stack direction="row" spacing={1} sx={{ alignItems: 'center', justifyContent: 'space-between', flexWrap: 'wrap' }}>
          <Typography variant="body2" color="text.secondary">
            {formatCurrency(player.wage)}
          </Typography>
          <Stack direction="row" spacing={1} sx={{ alignItems: 'center' }}>
            <Button
              size="small"
              onClick={() => setRenewalOpen(true)}
              disabled={player.contractYearsRemaining >= MAX_CONTRACT_YEARS_REMAINING}
            >
              {t('team.renewContract')}
            </Button>
            <Tooltip title={t('team.contractSeasons', { count: player.contractYearsRemaining })}>
              <Typography variant="body2" color="text.secondary" component="span">
                ({player.contractYearsRemaining})
              </Typography>
            </Tooltip>
            <Button size="small" color="secondary" onClick={handleRelease} disabled={releasePlayer.isPending}>
              {t('team.releasePlayer')}
            </Button>
          </Stack>
        </Stack>
        {releasePlayer.isError && <Alert severity="error">{t('team.releaseError')}</Alert>}
      </Stack>
    )

    return (
      <>
        <PlayerChip player={player} unavailability={unavailability} forSale={forSale} footer={footer} />
        <ContractRenewalDialog player={player} open={renewalOpen} onClose={() => setRenewalOpen(false)} />
      </>
    )
  }

  const flag = countryFlag(player.birthCountry)
  const rating = player.averageSkill ?? overallRating(player)

  return (
    <Paper sx={{ display: 'flex', alignItems: 'center', gap: 1.5, p: 1 }}>
      <CartoonAvatar seed={player.id} birthCountry={player.birthCountry} size={32} />
      <Box sx={{ minWidth: 0, flex: 1 }}>
        <Stack direction="row" spacing={1} sx={{ alignItems: 'center' }}>
          {flag && (
            <Tooltip title={t(COUNTRY_NAME_KEYS[player.birthCountry!] ?? player.birthCountry!)}>
              <Typography component="span" sx={{ flexShrink: 0, fontSize: '0.9rem', lineHeight: 1 }}>
                {flag}
              </Typography>
            </Tooltip>
          )}
          <Link component={RouterLink} to={`/player/${player.id}`} underline="hover" color="inherit" noWrap>
            {player.firstName} {player.lastName}
          </Link>
          {unavailability && <UnavailabilityBadge unavailability={unavailability} size={16} />}
          {forSale && <TransferListedBadge size={16} />}
        </Stack>
        <Typography variant="caption" color="text.secondary">
          {player.age}
        </Typography>
      </Box>
      <Typography variant="body2" sx={{ fontWeight: 700, color: ratingTierColor(rating, theme.palette) }}>
        {rating}
      </Typography>
      <Button size="small" onClick={() => setOfferOpen(true)}>
        {t('team.makeOffer')}
      </Button>
      <TransferOfferDialog player={player} open={offerOpen} onClose={() => setOfferOpen(false)} />
    </Paper>
  )
}
