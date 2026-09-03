import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import Card from '@mui/material/Card'
import CardContent from '@mui/material/CardContent'
import Skeleton from '@mui/material/Skeleton'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { Link as RouterLink } from 'react-router-dom'
import { useMe, useUserTeam } from '../../api/hooks'
import { PlayerCard } from '../../components/PlayerCard'
import { SeasonRecap } from './SeasonRecap'

export function DashboardPage() {
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
              Edit lineup
            </Button>
          </Stack>
        </CardContent>
      </Card>
      {userTeam.players.map((player) => (
        <PlayerCard key={player.id} player={player} />
      ))}
    </Box>
  )
}
