import { useState } from 'react'
import AppBar from '@mui/material/AppBar'
import Avatar from '@mui/material/Avatar'
import Box from '@mui/material/Box'
import IconButton from '@mui/material/IconButton'
import Link from '@mui/material/Link'
import Menu from '@mui/material/Menu'
import MenuIcon from '@mui/icons-material/Menu'
import MenuItem from '@mui/material/MenuItem'
import Toolbar from '@mui/material/Toolbar'
import Tooltip from '@mui/material/Tooltip'
import Typography from '@mui/material/Typography'
import { useTheme } from '@mui/material/styles'
import useMediaQuery from '@mui/material/useMediaQuery'
import { Link as RouterLink, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useLogout, useMe } from '../api/hooks'
import { LanguageSwitcher } from './LanguageSwitcher'
import { NotificationBell } from './NotificationBell'
import { ThemeModeToggle } from './ThemeModeToggle'

export function Header({ onMenuClick }: { onMenuClick: () => void }) {
  const { t } = useTranslation()
  const { data: loggedUser } = useMe()
  const theme = useTheme()
  const isDesktop = useMediaQuery(theme.breakpoints.up('md'))

  return (
    <Box sx={{ flexGrow: 1 }}>
      <AppBar position="static">
        <Toolbar>
          {loggedUser && !isDesktop && (
            <IconButton
              color="inherit"
              edge="start"
              onClick={onMenuClick}
              sx={{ mr: 1 }}
              aria-label={t('header.openMenu')}
            >
              <MenuIcon />
            </IconButton>
          )}
          <Typography variant="h6" component="div" sx={{ flexGrow: 1 }}>
            <Link component={RouterLink} to="/" underline="none" color="inherit">
              Striker
            </Link>
          </Typography>
          <ThemeModeToggle />
          <LanguageSwitcher />
          <NotificationBell />
          <AvatarOrLogin />
        </Toolbar>
      </AppBar>
    </Box>
  )
}

function AvatarOrLogin() {
  const { t } = useTranslation()
  const { data: loggedUser } = useMe()
  const logout = useLogout()
  const navigate = useNavigate()
  const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null)

  if (!loggedUser) {
    return (
      <Link component={RouterLink} to="/login" underline="hover" color="inherit">
        {t('header.login')}
      </Link>
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
        <IconButton onClick={(event) => setAnchorEl(event.currentTarget)} sx={{ p: 0 }}>
          <Avatar src={loggedUser.pictureUrl} slotProps={{ img: { referrerPolicy: 'no-referrer' } }} />
        </IconButton>
      </Tooltip>
      <Menu
        sx={{ mt: '45px' }}
        anchorEl={anchorEl}
        anchorOrigin={{ vertical: 'top', horizontal: 'right' }}
        transformOrigin={{ vertical: 'top', horizontal: 'right' }}
        open={Boolean(anchorEl)}
        onClose={() => setAnchorEl(null)}
      >
        <MenuItem onClick={handleLogout}>{t('header.nav.logout')}</MenuItem>
      </Menu>
    </Box>
  )
}
