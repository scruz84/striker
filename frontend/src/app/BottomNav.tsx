import BottomNavigation from '@mui/material/BottomNavigation'
import BottomNavigationAction from '@mui/material/BottomNavigationAction'
import Paper from '@mui/material/Paper'
import MenuIcon from '@mui/icons-material/Menu'
import { Link as RouterLink, useLocation } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { navIcon, usePrimaryDestinations } from './navItems'

/** The navigation bar's height, without the bottom safe area - pages keep this much clear. */
export const BOTTOM_NAV_HEIGHT = 80

/** M3 navigation bar for phones: the four screens used every day (dashboard, squad, lineup,
 * market) one tap away from anywhere, plus "More" for the rest of the navigation. Its look (80px,
 * pill indicator, 12px labels) comes from the theme's MuiBottomNavigation overrides. */
export function BottomNav({ onMore }: { onMore: () => void }) {
  const { t } = useTranslation()
  const location = useLocation()
  const destinations = usePrimaryDestinations()
  const selected = destinations.find((d) => d.to === location.pathname)?.to ?? 'more'

  return (
    <Paper
      component="nav"
      aria-label={t('nav.bottom.more')}
      elevation={0}
      square
      sx={{
        position: 'fixed',
        left: 0,
        right: 0,
        bottom: 0,
        zIndex: (theme) => theme.zIndex.appBar,
        pb: 'env(safe-area-inset-bottom)',
        bgcolor: 'surfaceContainer',
      }}
    >
      <BottomNavigation showLabels value={selected}>
        {destinations.map((destination) => (
          <BottomNavigationAction
            key={destination.to}
            value={destination.to}
            label={destination.label}
            icon={navIcon(destination, destination.to === selected)}
            component={RouterLink}
            to={destination.to}
          />
        ))}
        <BottomNavigationAction value="more" label={t('nav.bottom.more')} icon={<MenuIcon />} onClick={onMore} />
      </BottomNavigation>
    </Paper>
  )
}
