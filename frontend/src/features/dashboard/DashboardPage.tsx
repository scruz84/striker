import { useState } from 'react'
import Alert from '@mui/material/Alert'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import Card from '@mui/material/Card'
import CardContent from '@mui/material/CardContent'
import EditIcon from '@mui/icons-material/Edit'
import Grid from '@mui/material/Grid'
import IconButton from '@mui/material/IconButton'
import Skeleton from '@mui/material/Skeleton'
import Stack from '@mui/material/Stack'
import TextField from '@mui/material/TextField'
import Tooltip from '@mui/material/Tooltip'
import Typography from '@mui/material/Typography'
import { Link as RouterLink } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useMe, useRenameTeam, useUserTeam } from '../../api/hooks'
import type { Team } from '../../api/types'
import { PlayerCard } from '../../components/PlayerCard'
import { SeasonRecap } from './SeasonRecap'

export function DashboardPage() {
  const { t } = useTranslation()
  const { data: loggedUser } = useMe()
  const { data: userTeam, isLoading } = useUserTeam(!!loggedUser)

  if (isLoading || !userTeam) {
    return <Skeleton variant="rectangular" height={200} />
  }

  return (
    <Box>
      <SeasonRecap enabled={!!loggedUser} />
      <Card sx={{ mb: 2 }}>
        <CardContent>
          <Stack direction="row" sx={{ justifyContent: 'space-between', alignItems: 'center' }}>
            <TeamNameHeader team={userTeam.team} />
            <Button component={RouterLink} to="/dashboard/lineup" variant="outlined" size="small">
              {t('dashboard.editLineup')}
            </Button>
          </Stack>
        </CardContent>
      </Card>
      <Grid container spacing={2}>
        {userTeam.players.map((player) => (
          <Grid key={player.id} size={{ xs: 12, lg: 6 }}>
            <PlayerCard player={player} />
          </Grid>
        ))}
      </Grid>
    </Box>
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
      <Stack sx={{ width: '100%' }}>
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
