import BottomNavigation from '@mui/material/BottomNavigation'
import BottomNavigationAction from '@mui/material/BottomNavigationAction'
import Paper from '@mui/material/Paper'
import DashboardTwoToneIcon from '@mui/icons-material/DashboardTwoTone'
import MenuIcon from '@mui/icons-material/Menu'
import PersonTwoToneIcon from '@mui/icons-material/PersonTwoTone'
import SportsSoccerTwoToneIcon from '@mui/icons-material/SportsSoccerTwoTone'
import SwapHorizTwoToneIcon from '@mui/icons-material/SwapHorizTwoTone'
import { Link as RouterLink, useLocation } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useUserTeam } from '../api/hooks'

export const BOTTOM_NAV_HEIGHT = 56

/** Fixed bottom bar for phones: the four screens used every day (dashboard, squad, lineup,
 * market) one tap away from anywhere, plus "More" for the rest of the navigation. */
export function BottomNav({ onMore }: { onMore: () => void }) {
  const { t } = useTranslation()
  const location = useLocation()
  const { data: userTeam } = useUserTeam(true)

  const squadPath = userTeam ? `/team/${userTeam.team.id}` : undefined
  const destinations = [
    { value: '/dashboard', label: t('nav.bottom.dashboard'), icon: <DashboardTwoToneIcon /> },
    ...(squadPath ? [{ value: squadPath, label: t('nav.bottom.squad'), icon: <PersonTwoToneIcon /> }] : []),
    { value: '/dashboard/lineup', label: t('nav.bottom.lineup'), icon: <SportsSoccerTwoToneIcon /> },
    { value: '/transfers', label: t('nav.bottom.market'), icon: <SwapHorizTwoToneIcon /> },
  ]
  const selected = destinations.find((d) => d.value === location.pathname)?.value ?? 'more'

  return (
    <Paper
      component="nav"
      aria-label={t('nav.bottom.more')}
      elevation={3}
      square
      sx={{
        position: 'fixed',
        left: 0,
        right: 0,
        bottom: 0,
        zIndex: (theme) => theme.zIndex.appBar,
        pb: 'env(safe-area-inset-bottom)',
        borderTop: 1,
        borderColor: 'divider',
      }}
    >
      <BottomNavigation showLabels value={selected} sx={{ height: BOTTOM_NAV_HEIGHT, bgcolor: 'transparent' }}>
        {destinations.map((destination) => (
          <BottomNavigationAction
            key={destination.value}
            value={destination.value}
            label={destination.label}
            icon={destination.icon}
            component={RouterLink}
            to={destination.value}
            sx={{ minWidth: 0, '& .MuiBottomNavigationAction-label': { fontSize: '0.6875rem' } }}
          />
        ))}
        <BottomNavigationAction
          value="more"
          label={t('nav.bottom.more')}
          icon={<MenuIcon />}
          onClick={onMore}
          sx={{ minWidth: 0, '& .MuiBottomNavigationAction-label': { fontSize: '0.6875rem' } }}
        />
      </BottomNavigation>
    </Paper>
  )
}
