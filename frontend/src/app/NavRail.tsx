import Box from '@mui/material/Box'
import ButtonBase from '@mui/material/ButtonBase'
import IconButton from '@mui/material/IconButton'
import Typography from '@mui/material/Typography'
import MenuIcon from '@mui/icons-material/Menu'
import { Link as RouterLink, useLocation } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { NAV_RAIL_WIDTH, STICKY_NAV_COLUMN_SX } from './navigation'
import { navIcon, usePrimaryDestinations } from './navItems'

/** M3 navigation rail for laptop widths (900-1199px, see useNavMode): a menu button that opens
 * the full navigation drawer, then the four daily destinations with the navigation bar's pill
 * indicator. */
export function NavRail({ menuOpen, onMenu }: { menuOpen: boolean; onMenu: () => void }) {
  const { t } = useTranslation()
  const location = useLocation()
  const destinations = usePrimaryDestinations()

  return (
    <Box
      component="nav"
      aria-label={t('nav.bottom.more')}
      sx={{ ...STICKY_NAV_COLUMN_SX, width: NAV_RAIL_WIDTH, display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 1.5, pt: 1.5 }}
    >
      <IconButton onClick={onMenu} aria-label={t('nav.bottom.more')} aria-expanded={menuOpen} sx={{ width: 48, height: 48, mb: 1 }}>
        <MenuIcon />
      </IconButton>
      {destinations.map((destination) => {
        const selected = destination.to === location.pathname
        return (
          <ButtonBase
            key={destination.to}
            component={RouterLink}
            to={destination.to}
            aria-current={selected ? 'page' : undefined}
            sx={{
              width: NAV_RAIL_WIDTH,
              minHeight: 56,
              display: 'flex',
              flexDirection: 'column',
              gap: 0.5,
              borderRadius: 2,
              color: selected ? 'onSurface' : 'onSurfaceVariant',
              // M3 state layer on the pill, not the whole entry.
              '&:hover .nav-rail-pill': { bgcolor: selected ? 'secondaryContainer' : 'action.hover' },
            }}
          >
            <Box
              className="nav-rail-pill"
              sx={{
                width: 56,
                height: 32,
                borderRadius: 4,
                display: 'grid',
                placeItems: 'center',
                bgcolor: selected ? 'secondaryContainer' : 'transparent',
                color: selected ? 'onSecondaryContainer' : 'inherit',
                transition: 'background-color 200ms',
              }}
            >
              {navIcon(destination, selected)}
            </Box>
            <Typography component="span" sx={{ fontSize: '0.75rem', lineHeight: '1rem', fontWeight: selected ? 700 : 500 }}>
              {destination.label}
            </Typography>
          </ButtonBase>
        )
      })}
    </Box>
  )
}
