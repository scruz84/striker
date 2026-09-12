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
import TextField from '@mui/material/TextField'
import Typography from '@mui/material/Typography'
import { Link as RouterLink, useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  useMakeOffer,
  useMe,
  useReleasePlayer,
  useTeamDetails,
  useTeamHistory,
  useUserTeam,
} from '../../api/hooks'
import {
  AI_LEVEL_CARETAKER,
  AI_LEVEL_DIRECTOR,
  AI_LEVEL_MANAGER,
  MAX_CONTRACT_YEARS_REMAINING,
  type Player,
  type SeasonHistoryEntry,
} from '../../api/types'
import { CartoonAvatar } from '../../components/avatar/CartoonAvatar'
import { useConfirm } from '../../components/confirmDialogContext'
import { ContractRenewalDialog } from '../../components/ContractRenewalDialog'
import { HeadToHeadCard } from '../../components/HeadToHeadCard'
import { SeasonPositionChart } from '../../components/SeasonPositionChart'

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
      <Stack direction="row" spacing={1} sx={{ alignItems: 'center' }}>
        <Typography variant="h4" gutterBottom>
          {teamDetails.team.name} ({teamDetails.team.id})
        </Typography>
        <AiLevelBadge aiLevel={teamDetails.team.aiLevel} />
      </Stack>
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

// AiLevelBadge shows an unowned team's AI difficulty tier - undefined/0 (an owned team, or one
// GET /v1/teams/team/:id/ simply didn't populate) renders nothing at all, since the value is
// meaningless there.
function AiLevelBadge({ aiLevel }: { aiLevel: number | undefined }) {
  const { t } = useTranslation()
  if (!aiLevel) {
    return null
  }
  const labels: Record<number, string> = {
    [AI_LEVEL_CARETAKER]: t('team.aiLevel.caretaker'),
    [AI_LEVEL_MANAGER]: t('team.aiLevel.manager'),
    [AI_LEVEL_DIRECTOR]: t('team.aiLevel.director'),
  }
  const colors: Record<number, 'default' | 'info' | 'success'> = {
    [AI_LEVEL_CARETAKER]: 'default',
    [AI_LEVEL_MANAGER]: 'info',
    [AI_LEVEL_DIRECTOR]: 'success',
  }
  return <Chip size="small" color={colors[aiLevel] ?? 'default'} label={labels[aiLevel] ?? t('team.aiLevel.unknown')} />
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

function PlayerRow({ player, isOwnTeam }: { player: Player; isOwnTeam: boolean }) {
  const { t } = useTranslation()
  const confirm = useConfirm()
  const makeOffer = useMakeOffer()
  const releasePlayer = useReleasePlayer()
  const [amount, setAmount] = useState('')
  const [showForm, setShowForm] = useState(false)
  const [renewalOpen, setRenewalOpen] = useState(false)

  const handleSubmit = () => {
    const amountNum = Number(amount)
    if (!amountNum || amountNum <= 0) {
      return
    }
    makeOffer.mutate({ playerId: player.id, amount: amountNum }, { onSuccess: () => setShowForm(false) })
  }

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
          {showForm ? (
            <Stack direction="row" spacing={1} sx={{ justifyContent: 'flex-end', alignItems: 'center' }}>
              <TextField
                size="small"
                label={t('team.amount')}
                value={amount}
                onChange={(e) => setAmount(e.target.value)}
                sx={{ width: 120 }}
              />
              <Button size="small" variant="contained" onClick={handleSubmit} disabled={makeOffer.isPending}>
                {t('team.send')}
              </Button>
              <Button size="small" onClick={() => setShowForm(false)}>
                {t('common.cancel')}
              </Button>
            </Stack>
          ) : (
            <Button size="small" onClick={() => setShowForm(true)}>
              {t('team.makeOffer')}
            </Button>
          )}
          {makeOffer.isError && (
            <Alert severity="error" sx={{ mt: 1 }}>
              {t('team.offerError')}
            </Alert>
          )}
        </TableCell>
      )}
    </TableRow>
  )
}
