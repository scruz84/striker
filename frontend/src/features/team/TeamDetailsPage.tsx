import { useState } from 'react'
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
import Typography from '@mui/material/Typography'
import { Link as RouterLink, useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  useAcademy,
  useMe,
  usePromoteFromAcademy,
  useReleasePlayer,
  useTeamDetails,
  useTeamHistory,
  useTeamTrainer,
  useUserTeam,
} from '../../api/hooks'
import { MAX_CONTRACT_YEARS_REMAINING, type Player, type SeasonHistoryEntry } from '../../api/types'
import { CartoonAvatar } from '../../components/avatar/CartoonAvatar'
import { useConfirm } from '../../components/confirmDialogContext'
import { ContractRenewalDialog } from '../../components/ContractRenewalDialog'
import { TransferOfferDialog } from '../../components/TransferOfferDialog'
import { HeadToHeadCard } from '../../components/HeadToHeadCard'
import { SeasonPositionChart } from '../../components/SeasonPositionChart'
import { formatCurrency } from '../../utils/formatCurrency'
import { formatDateTime } from '../../utils/formatDate'

export function TeamDetailsPage() {
  const { t } = useTranslation()
  const { id } = useParams<{ id: string }>()
  const { data: loggedUser } = useMe()
  const { data: teamDetails, isLoading } = useTeamDetails(id)
  const { data: userTeam } = useUserTeam(!!loggedUser)
  const { data: history, isLoading: historyLoading } = useTeamHistory(id)

  if (isLoading || !teamDetails) {
    return <div>{t('common.loading')}</div>
  }

  const isOwnTeam = userTeam?.team.id === teamDetails.team.id

  return (
    <Stack spacing={1}>
      <Typography variant="h4" gutterBottom>
        {teamDetails.team.name} ({teamDetails.team.id})
      </Typography>
      <Typography sx={{ fontWeight: 'bold' }}>{t('team.competitions')}</Typography>
      <Link component={RouterLink} to={`/division/${teamDetails.team.divisionId}`}>
        {teamDetails.team.divisionName}
      </Link>

      <Typography variant="h6" sx={{ mt: 2 }}>
        {t('team.squad')}
      </Typography>
      <TableContainer component={Paper}>
        <Table size="small">
          <TableHead>
            <TableRow>
              <TableCell>{t('team.player')}</TableCell>
              <TableCell align="right">{t('team.age')}</TableCell>
              {isOwnTeam && <TableCell align="right">{t('team.contract')}</TableCell>}
              <TableCell align="right" />
            </TableRow>
          </TableHead>
          <TableBody>
            {teamDetails.players.map((player) => (
              <PlayerRow key={player.id} player={player} isOwnTeam={isOwnTeam} />
            ))}
          </TableBody>
        </Table>
      </TableContainer>

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
                    <CartoonAvatar seed={prospect.id} size={32} />
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

function PlayerRow({ player, isOwnTeam }: { player: Player; isOwnTeam: boolean }) {
  const { t } = useTranslation()
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

  return (
    <TableRow>
      <TableCell>
        <Stack direction="row" spacing={1.5} sx={{ alignItems: 'center' }}>
          <CartoonAvatar seed={player.id} size={32} />
          <Link component={RouterLink} to={`/player/${player.id}`} underline="hover" color="inherit">
            {player.firstName} {player.lastName}
          </Link>
        </Stack>
      </TableCell>
      <TableCell align="right">{player.age}</TableCell>
      {isOwnTeam && (
        <TableCell align="right">{t('team.contractSeasons', { count: player.contractYearsRemaining })}</TableCell>
      )}
      {isOwnTeam ? (
        <TableCell align="right">
          <Stack direction="row" spacing={1} sx={{ justifyContent: 'flex-end' }}>
            <Button
              size="small"
              onClick={() => setRenewalOpen(true)}
              disabled={player.contractYearsRemaining >= MAX_CONTRACT_YEARS_REMAINING}
            >
              {t('team.renewContract')}
            </Button>
            <Button size="small" color="secondary" onClick={handleRelease} disabled={releasePlayer.isPending}>
              {t('team.releasePlayer')}
            </Button>
          </Stack>
          {releasePlayer.isError && (
            <Alert severity="error" sx={{ mt: 1 }}>
              {t('team.releaseError')}
            </Alert>
          )}
          <ContractRenewalDialog player={player} open={renewalOpen} onClose={() => setRenewalOpen(false)} />
        </TableCell>
      ) : (
        <TableCell align="right">
          <Button size="small" onClick={() => setOfferOpen(true)}>
            {t('team.makeOffer')}
          </Button>
          <TransferOfferDialog player={player} open={offerOpen} onClose={() => setOfferOpen(false)} />
        </TableCell>
      )}
    </TableRow>
  )
}
