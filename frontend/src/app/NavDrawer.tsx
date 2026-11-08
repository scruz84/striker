import Box from '@mui/material/Box'
import Drawer from '@mui/material/Drawer'
import List from '@mui/material/List'
import ListItemButton from '@mui/material/ListItemButton'
import ListItemIcon from '@mui/material/ListItemIcon'
import ListItemText from '@mui/material/ListItemText'
import ListSubheader from '@mui/material/ListSubheader'
import { Link as RouterLink, useLocation } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useMe } from '../api/hooks'
import { LegalLinks } from '../features/legal/LegalLinks'
import { isNavSelected, STICKY_NAV_COLUMN_SX } from './navigation'
import { navIcon, useNavGroups } from './navItems'

export const NAV_DRAWER_WIDTH = 256

/** M3 navigation drawer item: a 56px pill, the current one in secondaryContainer. Styled here
 * rather than in the theme because other lists use ListItemButton with their own look. */
const ITEM_SX = {
  minHeight: 56,
  mx: 1.5,
  px: 2,
  borderRadius: 7,
  color: 'onSurfaceVariant',
  '& .MuiListItemIcon-root': { color: 'inherit', minWidth: 40 },
  '& .MuiListItemText-primary': { fontWeight: 500 },
  '&.Mui-selected, &.Mui-selected:hover': { bgcolor: 'secondaryContainer', color: 'onSecondaryContainer' },
} as const

/** The app's full site navigation, grouped (My club / Competitions / Social / Other).
 * - `standard`: the permanent drawer from 1200px, a column that stays under the header while the
 *   page scrolls (it is never drawn over the header).
 * - `modal`: opened from the navigation bar's "More" (phones) or the rail's menu button (laptop
 *   widths); it slides in from the left, the same side as the rail and the standard drawer - the
 *   right edge is the notification panel's. */
export function NavDrawer({ variant, open = false, onClose }: { variant: 'modal' | 'standard'; open?: boolean; onClose: () => void }) {
  const { t } = useTranslation()
  const location = useLocation()
  const { data: loggedUser } = useMe()
  const groups = useNavGroups()

  // Nothing here applies before a team exists - a pending user only ever sees the onboarding
  // country picker itself (see Layout's redirect), so the site nav would just be dead weight.
  if (!loggedUser || loggedUser.pendingCountrySelection) {
    return null
  }

  const content = (
    <Box sx={{ width: NAV_DRAWER_WIDTH, py: 1, pb: 'max(8px, env(safe-area-inset-bottom))' }} role="presentation">
      {groups.map((group) => (
        <List
          key={group.key}
          subheader={
            <ListSubheader disableSticky sx={{ bgcolor: 'transparent', px: 3.5, lineHeight: '40px', typography: 'overline', color: 'onSurfaceVariant' }}>
              {group.label}
            </ListSubheader>
          }
          sx={{ py: 0.5 }}
        >
          {group.items.map((item) => {
            const selected = isNavSelected(item.to, location.pathname, item.alsoMatches)
            return (
              <ListItemButton key={item.to} component={RouterLink} to={item.to} selected={selected} onClick={onClose} sx={ITEM_SX}>
                <ListItemIcon>{navIcon(item, selected)}</ListItemIcon>
                <ListItemText primary={item.label} />
              </ListItemButton>
            )
          })}
        </List>
      ))}
      <LegalLinks onNavigate={onClose} sx={{ px: 3.5, py: 1.5 }} />
    </Box>
  )

  if (variant === 'standard') {
    return (
      <Box component="nav" aria-label={t('nav.bottom.more')} sx={{ ...STICKY_NAV_COLUMN_SX, width: NAV_DRAWER_WIDTH }}>
        {content}
      </Box>
    )
  }

  return (
    <Drawer
      variant="temporary"
      anchor="left"
      open={open}
      onClose={onClose}
      ModalProps={{ keepMounted: true }}
      slotProps={{
        paper: {
          sx: { bgcolor: 'surfaceContainerLow', borderRadius: '0 16px 16px 0' },
        },
      }}
    >
      {content}
    </Drawer>
  )
}
