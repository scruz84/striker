import Box from '@mui/material/Box'
import Breadcrumbs from '@mui/material/Breadcrumbs'
import ChevronRightIcon from '@mui/icons-material/ChevronRight'
import Link from '@mui/material/Link'
import { Link as RouterLink, useLocation } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useMe, useUserTeam } from '../api/hooks'
import { TeamBadge } from '../components/TeamBadge'
import { formatDivisionName } from '../utils/divisionName'
import { isClubPage } from './navigation'

/** Club context strip: team and readable division with the current league position. Only shown on
 * pages about the user's own club - elsewhere (a match, another club, a cup) it would be noise. */
export function Subheader() {
  const { t } = useTranslation()
  const { pathname } = useLocation()
  const { data: loggedUser } = useMe()
  const { data: userTeam } = useUserTeam(!!loggedUser)

  if (!userTeam || !isClubPage(pathname, userTeam.team.id)) {
    return null
  }

  const position = userTeam.team.classificationPosition
  const divisionLabel = `${formatDivisionName(t, userTeam.team.divisionName)}${
    position > 0 ? ` · ${t('subheader.position', { count: position, ordinal: true })}` : ''
  }`

  return (
    <Box
      sx={{
        px: { xs: 2, sm: 3 },
        py: { xs: 0.5, md: 1 },
        display: 'flex',
        alignItems: 'center',
        gap: 1.5,
        bgcolor: 'background.paper',
        borderBottom: 1,
        borderColor: 'divider',
        '@media (orientation: landscape) and (max-height: 500px)': { py: 0 },
      }}
    >
      <TeamBadge teamId={userTeam.team.id} teamName={userTeam.team.name} size={28} />
      <Breadcrumbs
        separator={<ChevronRightIcon sx={{ fontSize: 16 }} color="disabled" />}
        aria-label="breadcrumb"
        sx={{ minWidth: 0, '& .MuiBreadcrumbs-ol': { flexWrap: 'nowrap' }, '& .MuiBreadcrumbs-li': { minWidth: 0 } }}
      >
        <Link
          component={RouterLink}
          to="/dashboard"
          underline="hover"
          color="text.primary"
          noWrap
          sx={{ fontWeight: 600, fontSize: '0.9rem', display: 'inline-flex', alignItems: 'center', minHeight: 44, maxWidth: '100%' }}
        >
          {userTeam.team.name}
        </Link>
        <Link
          component={RouterLink}
          to={`/division/${userTeam.team.divisionId}`}
          underline="hover"
          color="text.secondary"
          noWrap
          sx={{ fontSize: '0.9rem', display: 'inline-flex', alignItems: 'center', minHeight: 44 }}
        >
          {divisionLabel}
        </Link>
      </Breadcrumbs>
    </Box>
  )
}
