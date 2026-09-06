import { useState, type ReactNode } from 'react'
import AccountBalanceWalletTwoToneIcon from '@mui/icons-material/AccountBalanceWalletTwoTone'
import CalendarTodayTwoToneIcon from '@mui/icons-material/CalendarTodayTwoTone'
import EditIcon from '@mui/icons-material/Edit'
import GroupsTwoToneIcon from '@mui/icons-material/GroupsTwoTone'
import SportsSoccerTwoToneIcon from '@mui/icons-material/SportsSoccerTwoTone'
import Alert from '@mui/material/Alert'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import Card from '@mui/material/Card'
import CardActions from '@mui/material/CardActions'
import CardContent from '@mui/material/CardContent'
import Chip from '@mui/material/Chip'
import Grid from '@mui/material/Grid'
import IconButton from '@mui/material/IconButton'
import Link from '@mui/material/Link'
import Skeleton from '@mui/material/Skeleton'
import Stack from '@mui/material/Stack'
import TextField from '@mui/material/TextField'
import Tooltip from '@mui/material/Tooltip'
import Typography from '@mui/material/Typography'
import { Link as RouterLink } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useDivisionDetails, useFinances, useMe, useRenameTeam, useUserTeam } from '../../api/hooks'
import type { ClassificationPosition, Team, WeekMatch } from '../../api/types'
import { TeamBadge } from '../../components/TeamBadge'
import { formatCurrency } from '../../utils/formatCurrency'
import { formatDateTime } from '../../utils/formatDate'
import { SeasonRecap } from './SeasonRecap'

export function DashboardPage() {
  const { data: loggedUser } = useMe()
  const { data: userTeam, isLoading } = useUserTeam(!!loggedUser)
  const { data: finances } = useFinances(!!userTeam)
  const { data: division } = useDivisionDetails(userTeam ? String(userTeam.team.divisionId) : undefined)

  if (isLoading || !userTeam) {
    return <Skeleton variant="rectangular" height={320} sx={{ borderRadius: 2 }} />
  }

  const { team, players } = userTeam
  const nextMatch = division?.nextWeekMatches.find((m) => m.home === team.id || m.away === team.id)
  const lastMatch = division?.previousWeekMatches.find((m) => m.home === team.id || m.away === team.id)
  const standing = division?.classification.positions.find((p) => p.teamId === team.id)

  return (
    <Box>
      <SeasonRecap enabled={!!loggedUser} />
      <TeamHeaderCard team={team} standing={standing} />
      <Grid container spacing={2}>
        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <NextMatchCard match={nextMatch} teamId={team.id} />
        </Grid>
        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <LastResultCard match={lastMatch} teamId={team.id} />
        </Grid>
        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <BudgetCard budget={finances?.budget} />
        </Grid>
        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <SquadCard teamId={team.id} squadSize={players.length} />
        </Grid>
      </Grid>
    </Box>
  )
}

function TeamHeaderCard({ team, standing }: { team: Team; standing: ClassificationPosition | undefined }) {
  const { t } = useTranslation()
  return (
    <Card sx={{ mb: 2 }}>
      <CardContent>
        <Stack
          direction={{ xs: 'column', sm: 'row' }}
          spacing={2}
          sx={{ alignItems: { sm: 'center' }, justifyContent: 'space-between' }}
        >
          <Stack direction="row" spacing={2} sx={{ alignItems: 'center' }}>
            <TeamBadge teamId={team.id} teamName={team.name} size={56} />
            <Box>
              <TeamNameHeader team={team} />
              <Stack direction="row" spacing={1} sx={{ alignItems: 'center', mt: 0.5 }}>
                <Link
                  component={RouterLink}
                  to={`/division/${team.divisionId}`}
                  underline="hover"
                  variant="body2"
                  color="text.secondary"
                >
                  {team.divisionName}
                </Link>
                {standing && (
                  <Chip
                    size="small"
                    variant="outlined"
                    label={t('dashboard.leaguePosition', { position: standing.position })}
                  />
                )}
              </Stack>
            </Box>
          </Stack>
          <Button component={RouterLink} to="/dashboard/lineup" variant="contained" size="small">
            {t('dashboard.editLineup')}
          </Button>
        </Stack>
      </CardContent>
    </Card>
  )
}

function StatCard({
  icon,
  title,
  action,
  children,
}: {
  icon: ReactNode
  title: string
  action?: ReactNode
  children: ReactNode
}) {
  return (
    <Card variant="outlined" sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <CardContent sx={{ flexGrow: 1 }}>
        <Stack direction="row" spacing={1} sx={{ alignItems: 'center', mb: 1.5 }}>
          {icon}
          <Typography variant="overline" color="text.secondary" sx={{ lineHeight: 1 }}>
            {title}
          </Typography>
        </Stack>
        {children}
      </CardContent>
      {action && <CardActions sx={{ pt: 0 }}>{action}</CardActions>}
    </Card>
  )
}

function OpponentLine({ match, teamId }: { match: WeekMatch; teamId: number }) {
  const { t } = useTranslation()
  const isHome = match.home === teamId
  const opponentName = isHome ? match.awayName : match.homeName
  const opponentId = isHome ? match.away : match.home

  return (
    <Stack direction="row" spacing={1.5} sx={{ alignItems: 'center' }}>
      <TeamBadge teamId={opponentId} teamName={opponentName} size={32} />
      <Box sx={{ minWidth: 0 }}>
        <Typography variant="body2" noWrap sx={{ fontWeight: 600 }}>
          {opponentName}
        </Typography>
        <Typography variant="caption" color="text.secondary">
          {isHome ? t('dashboard.home') : t('dashboard.away')}
        </Typography>
      </Box>
    </Stack>
  )
}

function NextMatchCard({ match, teamId }: { match: WeekMatch | undefined; teamId: number }) {
  const { t } = useTranslation()
  return (
    <StatCard
      icon={<CalendarTodayTwoToneIcon color="primary" fontSize="small" />}
      title={t('dashboard.nextMatch')}
      action={
        match && (
          <Button size="small" component={RouterLink} to={`/match/${match.matchId}`}>
            {t('dashboard.viewMatch')}
          </Button>
        )
      }
    >
      {match ? (
        <Stack spacing={0.5}>
          <OpponentLine match={match} teamId={teamId} />
          <Typography variant="caption" color="text.secondary">
            {formatDateTime(match.matchDate)}
          </Typography>
        </Stack>
      ) : (
        <Typography variant="body2" color="text.secondary">
          {t('dashboard.noUpcomingMatch')}
        </Typography>
      )}
    </StatCard>
  )
}

function LastResultCard({ match, teamId }: { match: WeekMatch | undefined; teamId: number }) {
  const { t } = useTranslation()

  if (!match) {
    return (
      <StatCard icon={<SportsSoccerTwoToneIcon color="primary" fontSize="small" />} title={t('dashboard.lastResult')}>
        <Typography variant="body2" color="text.secondary">
          {t('dashboard.noRecentMatch')}
        </Typography>
      </StatCard>
    )
  }

  const isHome = match.home === teamId
  const ownGoals = isHome ? match.homeGoals : match.awayGoals
  const oppGoals = isHome ? match.awayGoals : match.homeGoals
  const outcome = ownGoals > oppGoals ? 'win' : ownGoals < oppGoals ? 'loss' : 'draw'
  const color = outcome === 'win' ? 'success' : outcome === 'loss' ? 'error' : 'default'

  return (
    <StatCard
      icon={<SportsSoccerTwoToneIcon color="primary" fontSize="small" />}
      title={t('dashboard.lastResult')}
      action={
        <Button size="small" component={RouterLink} to={`/match/${match.matchId}`}>
          {t('dashboard.viewMatch')}
        </Button>
      }
    >
      <Stack direction="row" spacing={1} sx={{ alignItems: 'center' }}>
        <Chip label={t(`dashboard.result.${outcome}`)} color={color} size="small" sx={{ fontWeight: 700 }} />
        <Typography variant="body2" sx={{ fontWeight: 700 }}>
          {ownGoals} - {oppGoals}
        </Typography>
      </Stack>
      <Box sx={{ mt: 1 }}>
        <OpponentLine match={match} teamId={teamId} />
      </Box>
    </StatCard>
  )
}

function BudgetCard({ budget }: { budget: number | undefined }) {
  const { t } = useTranslation()
  return (
    <StatCard
      icon={<AccountBalanceWalletTwoToneIcon color="primary" fontSize="small" />}
      title={t('finances.budget')}
      action={
        <Button size="small" component={RouterLink} to="/dashboard/finances">
          {t('dashboard.viewFinances')}
        </Button>
      }
    >
      {budget === undefined ? (
        <Skeleton width={100} height={32} />
      ) : (
        <Typography variant="h5" sx={{ fontWeight: 700 }}>
          {formatCurrency(budget)}
        </Typography>
      )}
    </StatCard>
  )
}

function SquadCard({ teamId, squadSize }: { teamId: number; squadSize: number }) {
  const { t } = useTranslation()
  return (
    <StatCard
      icon={<GroupsTwoToneIcon color="primary" fontSize="small" />}
      title={t('header.nav.squad')}
      action={
        <Button size="small" component={RouterLink} to={`/team/${teamId}`}>
          {t('dashboard.viewSquad')}
        </Button>
      }
    >
      <Typography variant="h5" sx={{ fontWeight: 700 }}>
        {squadSize}
      </Typography>
      <Typography variant="caption" color="text.secondary">
        {t('dashboard.players')}
      </Typography>
    </StatCard>
  )
}

function TeamNameHeader({ team }: { team: Team }) {
  const { t } = useTranslation()
  const renameTeam = useRenameTeam()
  const [editing, setEditing] = useState(false)
  const [name, setName] = useState(team.name)

  const handleSave = () => {
    const trimmed = name.trim()
    if (!trimmed || trimmed === team.name) {
      setEditing(false)
      return
    }
    renameTeam.mutate(trimmed, { onSuccess: () => setEditing(false) })
  }

  if (editing) {
    return (
      <Stack>
        <Stack direction="row" spacing={1} sx={{ alignItems: 'center' }}>
          <TextField
            size="small"
            autoFocus
            value={name}
            onChange={(e) => setName(e.target.value)}
            disabled={renameTeam.isPending}
          />
          <Button size="small" variant="contained" onClick={handleSave} disabled={renameTeam.isPending}>
            {t('dashboard.renameSave')}
          </Button>
          <Button
            size="small"
            onClick={() => {
              setEditing(false)
              setName(team.name)
            }}
          >
            {t('common.cancel')}
          </Button>
        </Stack>
        {renameTeam.isError && (
          <Alert severity="error" sx={{ mt: 1 }}>
            {t('dashboard.renameError')}
          </Alert>
        )}
      </Stack>
    )
  }

  return (
    <Stack direction="row" spacing={0.5} sx={{ alignItems: 'center' }}>
      <Typography variant="h6">{team.name}</Typography>
      {!team.nameChanged && (
        <Tooltip title={t('dashboard.renameTeam')}>
          <IconButton size="small" onClick={() => setEditing(true)}>
            <EditIcon fontSize="small" />
          </IconButton>
        </Tooltip>
      )}
    </Stack>
  )
}
