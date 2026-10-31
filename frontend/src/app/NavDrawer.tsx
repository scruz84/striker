import type { ReactNode } from 'react'
import AccountBalanceWalletTwoToneIcon from '@mui/icons-material/AccountBalanceWalletTwoTone'
import AdminPanelSettingsTwoToneIcon from '@mui/icons-material/AdminPanelSettingsTwoTone'
import CalendarMonthTwoToneIcon from '@mui/icons-material/CalendarMonthTwoTone'
import DashboardTwoToneIcon from '@mui/icons-material/DashboardTwoTone'
import EmojiEventsTwoToneIcon from '@mui/icons-material/EmojiEventsTwoTone'
import FlagTwoToneIcon from '@mui/icons-material/FlagTwoTone'
import GroupsTwoToneIcon from '@mui/icons-material/GroupsTwoTone'
import HandshakeTwoToneIcon from '@mui/icons-material/HandshakeTwoTone'
import HelpTwoToneIcon from '@mui/icons-material/HelpTwoTone'
import ManageAccountsTwoToneIcon from '@mui/icons-material/ManageAccountsTwoTone'
import PersonTwoToneIcon from '@mui/icons-material/PersonTwoTone'
import PublicTwoToneIcon from '@mui/icons-material/PublicTwoTone'
import SearchTwoToneIcon from '@mui/icons-material/SearchTwoTone'
import SportsSoccerTwoToneIcon from '@mui/icons-material/SportsSoccerTwoTone'
import SwapHorizTwoToneIcon from '@mui/icons-material/SwapHorizTwoTone'
import WorkspacePremiumTwoToneIcon from '@mui/icons-material/WorkspacePremiumTwoTone'
import Box from '@mui/material/Box'
import Drawer from '@mui/material/Drawer'
import List from '@mui/material/List'
import ListItemButton from '@mui/material/ListItemButton'
import ListItemIcon from '@mui/material/ListItemIcon'
import ListItemText from '@mui/material/ListItemText'
import ListSubheader from '@mui/material/ListSubheader'
import { useTheme } from '@mui/material/styles'
import useMediaQuery from '@mui/material/useMediaQuery'
import { Link as RouterLink, useLocation } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useMe, useUserTeam } from '../api/hooks'
import { LegalLinks } from '../features/legal/LegalLinks'
import { isNavSelected } from './navigation'

export const NAV_DRAWER_WIDTH = 232

interface NavItem {
  to: string
  label: string
  icon: ReactNode
  /** Extra path prefixes that keep this entry selected (e.g. /cup/ties for Cup). */
  alsoMatches?: string[]
}

interface NavGroup {
  key: string
  label?: string
  items: NavItem[]
}

/** The app's full site navigation, grouped (My club / Competitions / Social / Other). It is a
 * permanent sidebar on desktop; on a phone it opens from the bottom bar's "More" entry (the
 * four daily destinations live in the bottom bar itself). On a phone it slides in from the left,
 * the same side as the desktop sidebar, so the navigation lives in one place at every width - the
 * right edge is the notification panel's. */
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

  const groups: NavGroup[] = [
    {
      key: 'club',
      label: t('nav.groups.club'),
      items: [
        { to: '/dashboard', label: t('header.nav.dashboard'), icon: <DashboardTwoToneIcon /> },
        ...(userTeam
          ? [
              { to: `/team/${userTeam.team.id}`, label: t('header.nav.squad'), icon: <PersonTwoToneIcon /> },
              {
                to: `/team/${userTeam.team.id}/matches`,
                label: t('header.nav.matches'),
                icon: <CalendarMonthTwoToneIcon />,
              },
            ]
          : []),
        { to: '/dashboard/lineup', label: t('header.nav.lineup'), icon: <SportsSoccerTwoToneIcon /> },
        { to: '/transfers', label: t('header.nav.transferMarket'), icon: <SwapHorizTwoToneIcon /> },
        { to: '/dashboard/finances', label: t('header.nav.finances'), icon: <AccountBalanceWalletTwoToneIcon /> },
      ],
    },
    {
      key: 'competitions',
      label: t('nav.groups.competitions'),
      items: [
        { to: '/leagues', label: t('header.nav.leagues'), icon: <PublicTwoToneIcon />, alsoMatches: ['/division'] },
        { to: '/cup', label: t('header.nav.cup'), icon: <EmojiEventsTwoToneIcon />, alsoMatches: ['/cup/ties'] },
        { to: '/international-cup', label: t('header.nav.internationalCup'), icon: <WorkspacePremiumTwoToneIcon /> },
        { to: '/national-teams', label: t('header.nav.nationalTeams'), icon: <FlagTwoToneIcon /> },
      ],
    },
    {
      key: 'social',
      label: t('nav.groups.social'),
      items: [
        { to: '/friendlies', label: t('header.nav.friendlies'), icon: <HandshakeTwoToneIcon />, alsoMatches: ['/friendlies'] },
        { to: '/groups', label: t('header.nav.friendGroups'), icon: <GroupsTwoToneIcon /> },
      ],
    },
    {
      key: 'other',
      label: t('nav.groups.other'),
      items: [
        { to: '/search', label: t('header.nav.search'), icon: <SearchTwoToneIcon /> },
        { to: '/help', label: t('header.nav.help'), icon: <HelpTwoToneIcon /> },
        { to: '/account', label: t('header.nav.account'), icon: <ManageAccountsTwoToneIcon /> },
        ...(loggedUser.isAdmin
          ? [{ to: '/admin', label: t('header.nav.admin'), icon: <AdminPanelSettingsTwoToneIcon /> }]
          : []),
      ],
    },
  ]

  const content = (
    <Box sx={{ width: NAV_DRAWER_WIDTH, pb: 'env(safe-area-inset-bottom)' }} role="presentation">
      {groups.map((group) => (
        <List
          key={group.key}
          dense={false}
          subheader={
            group.label ? (
              <ListSubheader disableSticky sx={{ bgcolor: 'transparent', lineHeight: '36px', typography: 'overline' }}>
                {group.label}
              </ListSubheader>
            ) : undefined
          }
        >
          {group.items.map((item) => (
            <ListItemButton
              key={item.to}
              component={RouterLink}
              to={item.to}
              selected={isNavSelected(item.to, location.pathname, item.alsoMatches)}
              onClick={onClose}
            >
              <ListItemIcon>{item.icon}</ListItemIcon>
              <ListItemText primary={item.label} />
            </ListItemButton>
          ))}
        </List>
      ))}
      <LegalLinks onNavigate={onClose} sx={{ px: 2, py: 1.5 }} />
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
      anchor="left"
      open={mobileOpen}
      onClose={onClose}
      ModalProps={{ keepMounted: true }}
      sx={{ '& .MuiDrawer-paper': { width: NAV_DRAWER_WIDTH, boxSizing: 'border-box' } }}
    >
      {content}
    </Drawer>
  )
}
