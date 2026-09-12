import type { ReactNode } from 'react'
import AccountBalanceWalletTwoToneIcon from '@mui/icons-material/AccountBalanceWalletTwoTone'
import AdminPanelSettingsTwoToneIcon from '@mui/icons-material/AdminPanelSettingsTwoTone'
import DashboardTwoToneIcon from '@mui/icons-material/DashboardTwoTone'
import EmojiEventsTwoToneIcon from '@mui/icons-material/EmojiEventsTwoTone'
import GroupsTwoToneIcon from '@mui/icons-material/GroupsTwoTone'
import HandshakeTwoToneIcon from '@mui/icons-material/HandshakeTwoTone'
import PersonTwoToneIcon from '@mui/icons-material/PersonTwoTone'
import PublicTwoToneIcon from '@mui/icons-material/PublicTwoTone'
import SportsSoccerTwoToneIcon from '@mui/icons-material/SportsSoccerTwoTone'
import SwapHorizTwoToneIcon from '@mui/icons-material/SwapHorizTwoTone'
import Box from '@mui/material/Box'
import Drawer from '@mui/material/Drawer'
import List from '@mui/material/List'
import ListItemButton from '@mui/material/ListItemButton'
import ListItemIcon from '@mui/material/ListItemIcon'
import ListItemText from '@mui/material/ListItemText'
import { useTheme } from '@mui/material/styles'
import useMediaQuery from '@mui/material/useMediaQuery'
import { Link as RouterLink, useLocation } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useMe, useUserTeam } from '../api/hooks'

export const NAV_DRAWER_WIDTH = 232

interface NavItem {
  to: string
  label: string
  icon: ReactNode
}

/** The app's primary site navigation - every section that used to be hidden behind the account
 * avatar menu now lives here instead, as a permanent sidebar on desktop and a slide-out drawer
 * (toggled from the header's menu button) on narrow viewports. */
export function NavDrawer({ mobileOpen, onClose }: { mobileOpen: boolean; onClose: () => void }) {
  const { t } = useTranslation()
  const theme = useTheme()
  const isDesktop = useMediaQuery(theme.breakpoints.up('md'))
  const location = useLocation()
  const { data: loggedUser } = useMe()
  const { data: userTeam } = useUserTeam(!!loggedUser)

  // Nothing here applies before a team exists - a pending user only ever sees the onboarding
  // country picker itself (see Layout's redirect), so the site nav would just be dead weight.
  if (!loggedUser || loggedUser.pendingCountrySelection) {
    return null
  }

  const items: NavItem[] = [
    { to: '/dashboard', label: t('header.nav.dashboard'), icon: <DashboardTwoToneIcon /> },
    ...(userTeam
      ? [{ to: `/team/${userTeam.team.id}`, label: t('header.nav.squad'), icon: <PersonTwoToneIcon /> }]
      : []),
    { to: '/dashboard/lineup', label: t('header.nav.lineup'), icon: <SportsSoccerTwoToneIcon /> },
    { to: '/transfers', label: t('header.nav.transferMarket'), icon: <SwapHorizTwoToneIcon /> },
    { to: '/dashboard/finances', label: t('header.nav.finances'), icon: <AccountBalanceWalletTwoToneIcon /> },
    { to: '/leagues', label: t('header.nav.leagues'), icon: <PublicTwoToneIcon /> },
    { to: '/cup', label: t('header.nav.cup'), icon: <EmojiEventsTwoToneIcon /> },
    { to: '/friendlies', label: t('header.nav.friendlies'), icon: <HandshakeTwoToneIcon /> },
    { to: '/groups', label: t('header.nav.friendGroups'), icon: <GroupsTwoToneIcon /> },
    ...(loggedUser.isAdmin
      ? [{ to: '/admin', label: t('header.nav.admin'), icon: <AdminPanelSettingsTwoToneIcon /> }]
      : []),
  ]

  const content = (
    <Box sx={{ width: NAV_DRAWER_WIDTH }} role="presentation">
      <List>
        {items.map((item) => (
          <ListItemButton
            key={item.to}
            component={RouterLink}
            to={item.to}
            selected={location.pathname === item.to}
            onClick={onClose}
          >
            <ListItemIcon>{item.icon}</ListItemIcon>
            <ListItemText primary={item.label} />
          </ListItemButton>
        ))}
      </List>
    </Box>
  )

  if (isDesktop) {
    return (
      <Drawer
        variant="permanent"
        sx={{
          width: NAV_DRAWER_WIDTH,
          flexShrink: 0,
          '& .MuiDrawer-paper': { position: 'relative', width: NAV_DRAWER_WIDTH, boxSizing: 'border-box' },
        }}
      >
        {content}
      </Drawer>
    )
  }

  return (
    <Drawer
      variant="temporary"
      open={mobileOpen}
      onClose={onClose}
      ModalProps={{ keepMounted: true }}
      sx={{ '& .MuiDrawer-paper': { width: NAV_DRAWER_WIDTH, boxSizing: 'border-box' } }}
    >
      {content}
    </Drawer>
  )
}
