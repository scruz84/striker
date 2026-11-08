import { useState, type ReactNode } from 'react'
import EmojiEventsOutlinedIcon from '@mui/icons-material/EmojiEventsOutlined'
import Alert from '@mui/material/Alert'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import Chip from '@mui/material/Chip'
import Link from '@mui/material/Link'
import Paper from '@mui/material/Paper'
import Skeleton from '@mui/material/Skeleton'
import Stack from '@mui/material/Stack'
import Table from '@mui/material/Table'
import TableBody from '@mui/material/TableBody'
import TableCell from '@mui/material/TableCell'
import TableContainer from '@mui/material/TableContainer'
import TableHead from '@mui/material/TableHead'
import TableRow from '@mui/material/TableRow'
import Tab from '@mui/material/Tab'
import Tabs from '@mui/material/Tabs'
import Tooltip from '@mui/material/Tooltip'
import Typography from '@mui/material/Typography'
import { useTheme } from '@mui/material/styles'
import { Link as RouterLink, useParams, useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  useAcademy,
  useCupDetails,
  useLatestCupId,
  useLatestInternationalCupId,
  useMe,
  usePromoteFromAcademy,
  useTeamDetails,
  useTeamHistory,
  useTeamMatches,
  useUserTeam,
} from '../../api/hooks'
import {
  COUNTRY_NAME_KEYS,
  type CupDetails,
  type SeasonHistoryEntry,
  type StadiumSummary,
  type Trainer,
} from '../../api/types'
import { CartoonAvatar } from '../../components/avatar/CartoonAvatar'
import { regionForCountry } from '../../components/avatar/traits'
import { ChallengeToFriendlyDialog } from '../../components/ChallengeToFriendlyDialog'
import { HeadToHeadCard } from '../../components/HeadToHeadCard'
import { SeasonPositionChart } from '../../components/SeasonPositionChart'
import { EmptyState } from '../../components/EmptyState'
import { ResponsiveTable } from '../../components/ResponsiveTable'
import { TeamBadge } from '../../components/TeamBadge'
import { TeamNameLink } from '../../components/TeamNameLink'
import { LoanedOutSection } from './LoanedOutSection'
import { RivalsSection } from './RivalsSection'
import { SquadList } from './SquadList'
import { useSquadUnavailability } from '../lineup/useSquadUnavailability'
import { useSquadTransferListed } from '../lineup/useSquadTransferListed'
import { isPast } from '../../api/useLiveMatch'
import { MatchesList } from './TeamMatchesSection'
import { CountryFlag } from '../../components/CountryFlag'
import { formatDivisionName } from '../../utils/divisionName'
import { formatCurrency, formatNumber } from '../../utils/formatCurrency'
import { formatDate, formatDateTime } from '../../utils/formatDate'
import { ratingTierColor } from '../../utils/playerRating'
import { NationalTeamMatchesTab } from '../nationalteams/NationalTeamMatchesTab'
import { teamTabs, type TeamTab } from './teamTabs'

const RECENT_MATCHES_LIMIT = 5

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

export function TeamDetailsPage() {
  const { t } = useTranslation()
  const theme = useTheme()
  const { id } = useParams<{ id: string }>()
  const [searchParams, setSearchParams] = useSearchParams()
  const { data: loggedUser } = useMe()
  const { data: teamDetails, isLoading } = useTeamDetails(id)
  const { data: userTeam } = useUserTeam(!!loggedUser)
  const { data: history, isLoading: historyLoading } = useTeamHistory(id)
  const unavailability = useSquadUnavailability()
  const transferListed = useSquadTransferListed()
  const [challengeOpen, setChallengeOpen] = useState(false)

  // A national team has no league or club-cup campaign of its own (see Team.isNationalTeam) -
  // skip fetching any of that for one, rather than pulling its country's unrelated domestic/
  // international club-cup standings.
  const isNationalTeam = !!teamDetails?.team.isNationalTeam
  const teamCountry = !isNationalTeam ? teamDetails?.team.country : undefined
  const { data: latestCup } = useLatestCupId(teamCountry)
  const { data: cupDetails } = useCupDetails(latestCup?.cupId)
  const region = teamCountry ? regionForCountry(teamCountry) : undefined
  const { data: latestInternationalCup } = useLatestInternationalCupId(region)
  const { data: internationalCupDetails } = useCupDetails(latestInternationalCup?.cupId)

  if (isLoading || !teamDetails) {
    return <Skeleton variant="rectangular" height={200} />
  }

  const isOwnTeam = userTeam?.team.id === teamDetails.team.id
  const cupStatus = teamCupStatus(cupDetails, teamDetails.team.id)
  const internationalCupStatus = teamCupStatus(internationalCupDetails, teamDetails.team.id)
  const displayName = isNationalTeam
    ? t(COUNTRY_NAME_KEYS[teamDetails.team.country] ?? teamDetails.team.country)
    : teamDetails.team.name
  const canChallenge = !isOwnTeam && !!userTeam && !isNationalTeam

  // Tabs mirror the sections the single long page used to stack: its own tab per concern, kept in
  // the URL (?tab=squad) so a section can be linked to and survives a reload.
  const tabs = teamTabs(isNationalTeam, isOwnTeam)
  const requestedTab = searchParams.get('tab') as TeamTab | null
  const tab: TeamTab = requestedTab && tabs.includes(requestedTab) ? requestedTab : 'summary'

  return (
    <Stack spacing={1.5}>
      <Stack direction="row" spacing={1.5} sx={{ alignItems: 'center', minWidth: 0 }}>
        <TeamBadge teamId={teamDetails.team.id} teamName={displayName} size={48} />
        <Box sx={{ minWidth: 0, flex: 1 }}>
          <Stack direction="row" spacing={1} sx={{ alignItems: 'center', minWidth: 0 }}>
            <Typography variant="pageTitle" component="h1" noWrap title={displayName} sx={{ fontSize: { xs: '1.375rem', sm: '1.75rem' } }}>
              {displayName}
            </Typography>
            <CountryFlag code={teamDetails.team.country} height={22} />
          </Stack>
          {!isNationalTeam && (
            <Stack direction="row" spacing={1} sx={{ alignItems: 'center', flexWrap: 'wrap', rowGap: 0.5, mt: 0.5 }}>
              <Chip
                size="small"
                component={RouterLink}
                to={`/division/${teamDetails.team.divisionId}`}
                clickable
                label={`${formatDivisionName(t, teamDetails.team.divisionName)} · ${t('subheader.position', { count: teamDetails.team.classificationPosition, ordinal: true })}`}
              />
              <CupStatusChip label={t('team.cup.domestic')} status={cupStatus} />
              <CupStatusChip label={t('team.cup.international')} status={internationalCupStatus} />
            </Stack>
          )}
        </Box>
      </Stack>
      {canChallenge && (
        <>
          <Button variant="outlined" fullWidth onClick={() => setChallengeOpen(true)}>
            {t('friendlies.challenge.challengeTeam')}
          </Button>
          <ChallengeToFriendlyDialog
            opponentTeamId={teamDetails.team.id}
            opponentTeamName={teamDetails.team.name}
            open={challengeOpen}
            onClose={() => setChallengeOpen(false)}
          />
        </>
      )}

      <Tabs
        value={tab}
        onChange={(_event, value: TeamTab) => {
          const next = new URLSearchParams(searchParams)
          next.set('tab', value)
          setSearchParams(next, { replace: true })
        }}
        variant="fullWidth"
        sx={{ borderBottom: 1, borderColor: 'divider' }}
      >
        {tabs.map((value) => (
          <Tab key={value} value={value} label={t(`team.tabs.${value}`)} />
        ))}
      </Tabs>

      {tab === 'summary' && (
        <Stack spacing={2} sx={{ pt: 1 }}>
          {isNationalTeam ? (
            <Typography variant="body2" color="text.secondary">
              {t('team.nationalTeam.subtitle')}
            </Typography>
          ) : (
            <Stack direction="row" spacing={0.5} sx={{ alignItems: 'center' }}>
              <Typography variant="body2" color="text.secondary">
                {t('team.fanbase')}
              </Typography>
              <Typography variant="body2" sx={{ fontWeight: 700, color: ratingTierColor(teamDetails.fanbase, theme.palette) }}>
                {teamDetails.fanbase}
              </Typography>
            </Stack>
          )}
          {teamDetails.stadium && <StadiumCard stadium={teamDetails.stadium} />}
          <RecentMatchesCard teamId={teamDetails.team.id} />
          {!isNationalTeam && <RivalsSection teamId={teamDetails.team.id} teamName={teamDetails.team.name} />}
          {!isNationalTeam && <CoachCard trainer={teamDetails.trainer} isOwnTeam={isOwnTeam} />}
          {canChallenge && userTeam && (
            <HeadToHeadCard
              teamAId={userTeam.team.id}
              teamAName={userTeam.team.name}
              teamBId={teamDetails.team.id}
              teamBName={teamDetails.team.name}
            />
          )}
        </Stack>
      )}

      {tab === 'squad' && (
        <Box sx={{ pt: 1 }}>
          {isNationalTeam && (
            <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
              {t('team.nationalTeam.squadNote')}
            </Typography>
          )}
          <SquadList
            players={teamDetails.players}
            isOwnTeam={isOwnTeam}
            unavailability={unavailability}
            transferListed={transferListed}
            loanedIn={teamDetails.loanedIn}
          />
          <LoanedOutSection loanedOut={teamDetails.loanedOut ?? []} isOwnTeam={isOwnTeam} />
        </Box>
      )}

      {tab === 'matches' && isNationalTeam && <NationalTeamMatchesTab teamId={teamDetails.team.id} country={teamDetails.team.country} />}

      {tab === 'academy' && isOwnTeam && (
        <Box sx={{ pt: 1 }}>
          <AcademySection />
        </Box>
      )}

      {tab === 'history' && (
        <Stack spacing={2} sx={{ pt: 1 }}>
          <Typography variant="subtitle1" component="h2" sx={{ fontWeight: 600 }}>
            {t('team.trophyCabinet')}
          </Typography>
          {historyLoading || !history ? (
            <Skeleton variant="rectangular" height={100} />
          ) : (
            <>
              {history.filter((entry) => entry.finalPosition !== null).length >= 2 ? (
                <SeasonPositionChart history={history} />
              ) : (
                history.length > 0 && (
                  <Typography variant="caption" color="text.secondary">
                    {t('team.noHistoryChart')}
                  </Typography>
                )
              )}
              <TrophyCabinet history={history} />
            </>
          )}
        </Stack>
      )}
    </Stack>
  )
}

// Same gold used for the 1st-place rank badge on DivisionDetailsPage, so a league title reads
// consistently as "gold" wherever it shows up in the UI.
const CHAMPION_COLOR = '#D4AF37'

export function TrophyCabinet({ history }: { history: SeasonHistoryEntry[] }) {
  const { t } = useTranslation()
  if (history.length === 0) {
    return <EmptyState compact title={t('team.noSeasons')} />
  }

  const outcomeChips = (entry: SeasonHistoryEntry) => (
    <Stack direction="row" spacing={0.5} sx={{ flexWrap: 'wrap', rowGap: 0.5 }}>
      {entry.promoted && <Chip size="small" color="success" label={t('team.promoted')} />}
      {entry.relegated && <Chip size="small" color="warning" label={t('team.relegated')} />}
      {entry.cupResult && <Chip size="small" color="primary" label={entry.cupResult} />}
    </Stack>
  )
  // A season with a recorded league links to that league's table for that season.
  const seasonLink = (entry: SeasonHistoryEntry, label: ReactNode) =>
    entry.leagueId === null ? (
      label
    ) : (
      <Link component={RouterLink} to={`/division/${entry.leagueId}?season=${entry.season}`} underline="hover">
        {label}
      </Link>
    )
  const position = (entry: SeasonHistoryEntry) => (
    <Stack direction="row" spacing={0.5} sx={{ alignItems: 'center', justifyContent: 'flex-end' }}>
      <span>{entry.finalPosition ?? '-'}</span>
      {entry.finalPosition === 1 && (
        <Tooltip title={t('team.leagueChampion')}>
          <EmojiEventsOutlinedIcon sx={{ color: CHAMPION_COLOR }} fontSize="small" />
        </Tooltip>
      )}
    </Stack>
  )

  return (
    <ResponsiveTable
      rows={history}
      getKey={(entry) => entry.season}
      ariaLabel={t('team.trophyCabinet')}
      columns={[
        { id: 'season', header: t('team.table.season'), cell: (entry) => seasonLink(entry, entry.season) },
        { id: 'league', header: t('team.table.league'), cell: (entry) => seasonLink(entry, entry.leagueName ?? '-') },
        { id: 'position', header: t('team.table.finalPosition'), align: 'right', cell: position },
        { id: 'outcome', header: t('team.table.outcome'), cell: outcomeChips },
      ]}
      mobile={{
        primary: (entry) => seasonLink(entry, `${t('team.table.season')} ${entry.season} · ${entry.leagueName ?? '-'}`),
        secondary: (entry) => (entry.promoted || entry.relegated || entry.cupResult ? outcomeChips(entry) : null),
        end: (entry) => (
          <Box component="span" sx={{ display: 'inline-flex', alignItems: 'center', gap: 0.5 }}>
            {t('team.table.finalPosition')} {position(entry)}
          </Box>
        ),
      }}
    />
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
        {t('team.stadium.capacity', { capacity: formatNumber(stadium.capacity) })}
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
            {t('team.stadium.attendance', { count: formatNumber(match.attendance) })}
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

// CoachCard shows a team's currently employed head coach (if any) right on the team page - any
// team's, not just the caller's own (#67: a rival's page showed no coach at all, even though the
// backend already resolves one generically - see common.GetTeamDetailsById). Hiring/firing itself
// still only happens from the Coaching tab of the transfer market (see
// TransferMarketPage.CoachingTab), which this links straight to via `?tab=coaching` rather than
// duplicating that flow here (issue #58) - and only for the caller's own team, since that's the
// only one they're allowed to hire/fire for.
function CoachCard({ trainer, isOwnTeam }: { trainer?: Trainer; isOwnTeam: boolean }) {
  const { t } = useTranslation()

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
          {isOwnTeam && (
            <Link component={RouterLink} to="/transfers?tab=coaching" underline="hover">
              {t('team.coach.manage')}
            </Link>
          )}
        </Stack>
      ) : (
        <Stack direction="row" spacing={2} sx={{ alignItems: 'center', mt: 0.5 }}>
          <Typography variant="body2" color="text.secondary">
            {t('team.coach.none')}
          </Typography>
          {isOwnTeam && (
            <Link component={RouterLink} to="/transfers?tab=coaching" underline="hover">
              {t('team.coach.hire')}
            </Link>
          )}
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
