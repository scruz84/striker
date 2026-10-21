import { useState, type ReactNode } from 'react'
import AppBar from '@mui/material/AppBar'
import Avatar from '@mui/material/Avatar'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import Divider from '@mui/material/Divider'
import IconButton from '@mui/material/IconButton'
import Link from '@mui/material/Link'
import ListItemIcon from '@mui/material/ListItemIcon'
import ListItemText from '@mui/material/ListItemText'
import Menu from '@mui/material/Menu'
import MenuItem from '@mui/material/MenuItem'
import Slide from '@mui/material/Slide'
import Toolbar from '@mui/material/Toolbar'
import Tooltip from '@mui/material/Tooltip'
import useScrollTrigger from '@mui/material/useScrollTrigger'
import { useTheme } from '@mui/material/styles'
import useMediaQuery from '@mui/material/useMediaQuery'
import CheckIcon from '@mui/icons-material/Check'
import DarkModeIcon from '@mui/icons-material/DarkMode'
import HelpOutlinedIcon from '@mui/icons-material/HelpOutlined'
import LightModeIcon from '@mui/icons-material/LightMode'
import LogoutIcon from '@mui/icons-material/Logout'
import MoreVertIcon from '@mui/icons-material/MoreVert'
import SearchIcon from '@mui/icons-material/Search'
import TranslateIcon from '@mui/icons-material/Translate'
import { Link as RouterLink, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useLogout, useMe } from '../api/hooks'
import { SUPPORTED_LANGUAGES, type SupportedLanguage } from '../i18n'
import { LanguageSwitcher } from './LanguageSwitcher'
import { Logo } from './Logo'
import { NotificationBell } from './NotificationBell'
import { ThemeModeToggle } from './ThemeModeToggle'
import { useThemeMode } from './themeMode'
import { useChangeLanguage } from './useChangeLanguage'

/** The top bar. On a phone it is reduced to logo + search + bell + avatar (theme and language live
 * in the avatar menu, navigation in the bottom bar) and slides away while scrolling down, coming
 * back on the first scroll up - which matters most in landscape, where it would otherwise eat a
 * third of the screen. */
export function Header() {
  const { t } = useTranslation()
  const { data: loggedUser } = useMe()
  const theme = useTheme()
  const isDesktop = useMediaQuery(theme.breakpoints.up('md'))
  const scrolledDown = useScrollTrigger({ threshold: 64 })

  return (
    <Slide appear={false} direction="down" in={isDesktop || !scrolledDown}>
      <AppBar position="sticky">
        <Toolbar sx={{ gap: { xs: 0, md: 0.5 }, minHeight: { xs: 56, md: 64 } }}>
          <Box sx={{ flexGrow: 1, minWidth: 0 }}>
            <Link component={RouterLink} to="/" underline="none" color="inherit" sx={{ display: 'inline-flex' }}>
              <Logo />
            </Link>
          </Box>
          {isDesktop ? (
            <>
              {!loggedUser && (
                <Link component={RouterLink} to="/help" underline="hover" color="inherit" sx={{ mr: 2 }}>
                  {t('header.nav.help')}
                </Link>
              )}
              <ThemeModeToggle />
              <LanguageSwitcher />
              <NotificationBell />
              <AvatarOrLogin />
            </>
          ) : loggedUser ? (
            <>
              <Tooltip title={t('header.searchIcon')}>
                <IconButton color="inherit" component={RouterLink} to="/search" aria-label={t('header.searchIcon')}>
                  <SearchIcon />
                </IconButton>
              </Tooltip>
              <NotificationBell />
              <AvatarOrLogin />
            </>
          ) : (
            <>
              <GuestMenu />
              <AvatarOrLogin />
            </>
          )}
        </Toolbar>
      </AppBar>
    </Slide>
  )
}

/** Theme + language entries, shared by the signed-in avatar menu and the guest "more" menu on
 * a phone (where they no longer have header icons of their own). */
function PreferenceItems({ onDone }: { onDone: () => void }) {
  const { t } = useTranslation()
  const { mode, toggleMode } = useThemeMode()
  const { currentLanguage, changeLanguage } = useChangeLanguage()

  const items: ReactNode[] = [
    <MenuItem
      key="theme"
      onClick={() => {
        toggleMode()
        onDone()
      }}
    >
      <ListItemIcon>{mode === 'dark' ? <LightModeIcon /> : <DarkModeIcon />}</ListItemIcon>
      <ListItemText>{mode === 'dark' ? t('header.lightMode') : t('header.darkMode')}</ListItemText>
    </MenuItem>,
    ...(Object.keys(SUPPORTED_LANGUAGES) as SupportedLanguage[]).map((language) => (
      <MenuItem
        key={language}
        selected={language === currentLanguage}
        onClick={() => {
          changeLanguage(language)
          onDone()
        }}
      >
        <ListItemIcon>{language === currentLanguage ? <CheckIcon /> : <TranslateIcon />}</ListItemIcon>
        <ListItemText>{SUPPORTED_LANGUAGES[language]}</ListItemText>
      </MenuItem>
    )),
  ]
  return <>{items}</>
}

function GuestMenu() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null)

  return (
    <>
      <IconButton color="inherit" onClick={(event) => setAnchorEl(event.currentTarget)} aria-label={t('header.more')}>
        <MoreVertIcon />
      </IconButton>
      <Menu anchorEl={anchorEl} open={Boolean(anchorEl)} onClose={() => setAnchorEl(null)}>
        <MenuItem
          onClick={() => {
            setAnchorEl(null)
            navigate('/help')
          }}
        >
          <ListItemIcon>
            <HelpOutlinedIcon />
          </ListItemIcon>
          <ListItemText>{t('header.nav.help')}</ListItemText>
        </MenuItem>
        <Divider />
        <PreferenceItems onDone={() => setAnchorEl(null)} />
      </Menu>
    </>
  )
}

function AvatarOrLogin() {
  const { t } = useTranslation()
  const { data: loggedUser } = useMe()
  const logout = useLogout()
  const navigate = useNavigate()
  const theme = useTheme()
  const isDesktop = useMediaQuery(theme.breakpoints.up('md'))
  const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null)

  if (!loggedUser) {
    return (
      <Button
        component={RouterLink}
        to="/login"
        color="inherit"
        size="small"
        sx={{ whiteSpace: 'nowrap', flexShrink: 0, px: { xs: 1.5, md: 2 } }}
      >
        {t('header.login')}
      </Button>
    )
  }

  const handleLogout = () => {
    setAnchorEl(null)
    logout.mutate(undefined, {
      onSuccess: () => navigate('/'),
    })
  }

  return (
    <Box sx={{ flexGrow: 0 }}>
      <Tooltip title={t('header.account')}>
        <IconButton
          onClick={(event) => setAnchorEl(event.currentTarget)}
          sx={{ p: 0.5 }}
          aria-label={t('header.accountMenu')}
        >
          <Avatar
            src={loggedUser.pictureUrl}
            sx={{ width: 36, height: 36 }}
            slotProps={{ img: { referrerPolicy: 'no-referrer' } }}
          />
        </IconButton>
      </Tooltip>
      <Menu
        anchorEl={anchorEl}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
        transformOrigin={{ vertical: 'top', horizontal: 'right' }}
        open={Boolean(anchorEl)}
        onClose={() => setAnchorEl(null)}
      >
        {!isDesktop && <PreferenceItems onDone={() => setAnchorEl(null)} />}
        {!isDesktop && <Divider />}
        <MenuItem onClick={handleLogout}>
          <ListItemIcon>
            <LogoutIcon />
          </ListItemIcon>
          <ListItemText>{t('header.nav.logout')}</ListItemText>
        </MenuItem>
      </Menu>
    </Box>
  )
}
