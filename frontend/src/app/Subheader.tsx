import Box from '@mui/material/Box'
import Breadcrumbs from '@mui/material/Breadcrumbs'
import ChevronRightIcon from '@mui/icons-material/ChevronRight'
import Link from '@mui/material/Link'
import { Link as RouterLink } from 'react-router-dom'
import { useMe, useUserTeam } from '../api/hooks'
import { TeamBadge } from '../components/TeamBadge'

export function Subheader() {
  const { data: loggedUser } = useMe()
  const { data: userTeam } = useUserTeam(!!loggedUser)

  if (!userTeam) {
    return null
  }

  return (
    <Box
      sx={{
        px: { xs: 2, sm: 3 },
        py: 1,
        display: 'flex',
        alignItems: 'center',
        gap: 1.5,
        bgcolor: 'background.paper',
        borderBottom: 1,
        borderColor: 'divider',
      }}
    >
      <TeamBadge teamId={userTeam.team.id} teamName={userTeam.team.name} size={28} />
      <Breadcrumbs separator={<ChevronRightIcon sx={{ fontSize: 16 }} color="disabled" />} aria-label="breadcrumb">
        <Link
          component={RouterLink}
          to="/dashboard"
          underline="hover"
          color="text.primary"
          sx={{ fontWeight: 600, fontSize: '0.9rem' }}
        >
          {userTeam.team.name}
        </Link>
        <Link
          component={RouterLink}
          to={`/division/${userTeam.team.divisionId}`}
          underline="hover"
          color="text.secondary"
          sx={{ fontSize: '0.9rem' }}
        >
          {userTeam.team.divisionName}
        </Link>
      </Breadcrumbs>
    </Box>
  )
}
