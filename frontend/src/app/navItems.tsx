import type { ReactElement } from 'react'
import AccountBalanceWalletIcon from '@mui/icons-material/AccountBalanceWallet'
import AccountBalanceWalletOutlinedIcon from '@mui/icons-material/AccountBalanceWalletOutlined'
import AdminPanelSettingsIcon from '@mui/icons-material/AdminPanelSettings'
import AdminPanelSettingsOutlinedIcon from '@mui/icons-material/AdminPanelSettingsOutlined'
import CalendarMonthIcon from '@mui/icons-material/CalendarMonth'
import CalendarMonthOutlinedIcon from '@mui/icons-material/CalendarMonthOutlined'
import DashboardIcon from '@mui/icons-material/Dashboard'
import DashboardOutlinedIcon from '@mui/icons-material/DashboardOutlined'
import EmojiEventsIcon from '@mui/icons-material/EmojiEvents'
import EmojiEventsOutlinedIcon from '@mui/icons-material/EmojiEventsOutlined'
import FlagIcon from '@mui/icons-material/Flag'
import FlagOutlinedIcon from '@mui/icons-material/FlagOutlined'
import GroupsIcon from '@mui/icons-material/Groups'
import GroupsOutlinedIcon from '@mui/icons-material/GroupsOutlined'
import HandshakeIcon from '@mui/icons-material/Handshake'
import HandshakeOutlinedIcon from '@mui/icons-material/HandshakeOutlined'
import HelpIcon from '@mui/icons-material/Help'
import HelpOutlineIcon from '@mui/icons-material/HelpOutlineOutlined'
import ManageAccountsIcon from '@mui/icons-material/ManageAccounts'
import ManageAccountsOutlinedIcon from '@mui/icons-material/ManageAccountsOutlined'
import PersonIcon from '@mui/icons-material/Person'
import PersonOutlinedIcon from '@mui/icons-material/PersonOutlined'
import PublicIcon from '@mui/icons-material/Public'
import PublicOutlinedIcon from '@mui/icons-material/PublicOutlined'
import SearchIcon from '@mui/icons-material/Search'
import SearchOutlinedIcon from '@mui/icons-material/SearchOutlined'
import SportsSoccerIcon from '@mui/icons-material/SportsSoccer'
import SportsSoccerOutlinedIcon from '@mui/icons-material/SportsSoccerOutlined'
import SwapHorizIcon from '@mui/icons-material/SwapHoriz'
import SwapHorizOutlinedIcon from '@mui/icons-material/SwapHorizOutlined'
import WorkspacePremiumIcon from '@mui/icons-material/WorkspacePremium'
import WorkspacePremiumOutlinedIcon from '@mui/icons-material/WorkspacePremiumOutlined'
import { useTranslation } from 'react-i18next'
import { useMe, useUserTeam } from '../api/hooks'

/** One navigation destination. Every navigation surface (bar, rail, both drawers) shows `icon`
 * (outlined) and swaps to `selectedIcon` (filled, same glyph) while it is the current entry. */
export interface NavItem {
  to: string
  label: string
  icon: ReactElement
  selectedIcon: ReactElement
  /** Extra path prefixes that keep this entry selected in the drawers (e.g. /cup/ties for Cup). */
  alsoMatches?: string[]
}

export interface NavGroup {
  key: string
  label: string
  items: NavItem[]
}

/** The entry's filled icon while it is the current one, its outlined icon otherwise. */
export function navIcon(item: Pick<NavItem, 'icon' | 'selectedIcon'>, selected: boolean): ReactElement {
  return selected ? item.selectedIcon : item.icon
}

/** The app's full site navigation, grouped (My club / Competitions / Social / Other) - what the
 * navigation drawers list. Empty until the user is signed in. */
export function useNavGroups(): NavGroup[] {
  const { t } = useTranslation()
  const { data: loggedUser } = useMe()
  const { data: userTeam } = useUserTeam(!!loggedUser)
  if (!loggedUser) {
    return []
  }
  const teamId = userTeam?.team.id

  return [
    {
      key: 'club',
      label: t('nav.groups.club'),
      items: [
        { to: '/dashboard', label: t('header.nav.dashboard'), icon: <DashboardOutlinedIcon />, selectedIcon: <DashboardIcon /> },
        ...(teamId !== undefined
          ? [
              { to: `/team/${teamId}`, label: t('header.nav.squad'), icon: <PersonOutlinedIcon />, selectedIcon: <PersonIcon /> },
              { to: `/team/${teamId}/matches`, label: t('header.nav.matches'), icon: <CalendarMonthOutlinedIcon />, selectedIcon: <CalendarMonthIcon /> },
            ]
          : []),
        { to: '/dashboard/lineup', label: t('header.nav.lineup'), icon: <SportsSoccerOutlinedIcon />, selectedIcon: <SportsSoccerIcon /> },
        { to: '/transfers', label: t('header.nav.transferMarket'), icon: <SwapHorizOutlinedIcon />, selectedIcon: <SwapHorizIcon /> },
        { to: '/dashboard/finances', label: t('header.nav.finances'), icon: <AccountBalanceWalletOutlinedIcon />, selectedIcon: <AccountBalanceWalletIcon /> },
      ],
    },
    {
      key: 'competitions',
      label: t('nav.groups.competitions'),
      items: [
        { to: '/leagues', label: t('header.nav.leagues'), icon: <PublicOutlinedIcon />, selectedIcon: <PublicIcon />, alsoMatches: ['/division'] },
        { to: '/cup', label: t('header.nav.cup'), icon: <EmojiEventsOutlinedIcon />, selectedIcon: <EmojiEventsIcon />, alsoMatches: ['/cup/ties'] },
        { to: '/international-cup', label: t('header.nav.internationalCup'), icon: <WorkspacePremiumOutlinedIcon />, selectedIcon: <WorkspacePremiumIcon /> },
        { to: '/national-teams', label: t('header.nav.nationalTeams'), icon: <FlagOutlinedIcon />, selectedIcon: <FlagIcon /> },
      ],
    },
    {
      key: 'social',
      label: t('nav.groups.social'),
      items: [
        { to: '/friendlies', label: t('header.nav.friendlies'), icon: <HandshakeOutlinedIcon />, selectedIcon: <HandshakeIcon />, alsoMatches: ['/friendlies'] },
        { to: '/groups', label: t('header.nav.friendGroups'), icon: <GroupsOutlinedIcon />, selectedIcon: <GroupsIcon /> },
      ],
    },
    {
      key: 'other',
      label: t('nav.groups.other'),
      items: [
        { to: '/search', label: t('header.nav.search'), icon: <SearchOutlinedIcon />, selectedIcon: <SearchIcon /> },
        { to: '/help', label: t('header.nav.help'), icon: <HelpOutlineIcon />, selectedIcon: <HelpIcon /> },
        { to: '/account', label: t('header.nav.account'), icon: <ManageAccountsOutlinedIcon />, selectedIcon: <ManageAccountsIcon /> },
        ...(loggedUser.isAdmin
          ? [{ to: '/admin', label: t('header.nav.admin'), icon: <AdminPanelSettingsOutlinedIcon />, selectedIcon: <AdminPanelSettingsIcon /> }]
          : []),
      ],
    },
  ]
}

/** The four destinations used every day (dashboard, squad, lineup, market) with their short
 * labels - the navigation bar's and the rail's entries, next to "More" / the menu button. Squad
 * appears once the user's team is known. */
export function usePrimaryDestinations(): NavItem[] {
  const { t } = useTranslation()
  const { data: userTeam } = useUserTeam(true)
  const squadPath = userTeam ? `/team/${userTeam.team.id}` : undefined
  return [
    { to: '/dashboard', label: t('nav.bottom.dashboard'), icon: <DashboardOutlinedIcon />, selectedIcon: <DashboardIcon /> },
    ...(squadPath ? [{ to: squadPath, label: t('nav.bottom.squad'), icon: <PersonOutlinedIcon />, selectedIcon: <PersonIcon /> }] : []),
    { to: '/dashboard/lineup', label: t('nav.bottom.lineup'), icon: <SportsSoccerOutlinedIcon />, selectedIcon: <SportsSoccerIcon /> },
    { to: '/transfers', label: t('nav.bottom.market'), icon: <SwapHorizOutlinedIcon />, selectedIcon: <SwapHorizIcon /> },
  ]
}
