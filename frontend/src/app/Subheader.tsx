import Breadcrumbs from '@mui/material/Breadcrumbs'
import Link from '@mui/material/Link'
import NavigateNextIcon from '@mui/icons-material/NavigateNext'
import Stack from '@mui/material/Stack'
import { Link as RouterLink } from 'react-router-dom'
import { useMe, useUserTeam } from '../api/hooks'

export function Subheader() {
  const { data: loggedUser } = useMe()
  const { data: userTeam } = useUserTeam(!!loggedUser)

  if (!userTeam) {
    return null
  }

  return (
    <Stack spacing={2} sx={{ px: 2, py: 1 }}>
      <Breadcrumbs separator={<NavigateNextIcon fontSize="small" />} aria-label="breadcrumb">
        <Link component={RouterLink} to="/dashboard" underline="none" color="inherit">
          {userTeam.team.name}
        </Link>
        <Link component={RouterLink} to={`/division/${userTeam.team.divisionId}`} underline="none" color="inherit">
          {userTeam.team.divisionName}
        </Link>
      </Breadcrumbs>
    </Stack>
  )
}
