import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import Card from '@mui/material/Card'
import CardContent from '@mui/material/CardContent'
import Grid from '@mui/material/Grid'
import Skeleton from '@mui/material/Skeleton'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { Link as RouterLink } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useMe, useUserTeam } from '../../api/hooks'
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
            <Typography variant="h6">{userTeam.team.name}</Typography>
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
