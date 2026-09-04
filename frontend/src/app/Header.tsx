import { useState } from 'react'
import AppBar from '@mui/material/AppBar'
import Avatar from '@mui/material/Avatar'
import Box from '@mui/material/Box'
import IconButton from '@mui/material/IconButton'
import Link from '@mui/material/Link'
import Menu from '@mui/material/Menu'
import MenuItem from '@mui/material/MenuItem'
import Toolbar from '@mui/material/Toolbar'
import Tooltip from '@mui/material/Tooltip'
import Typography from '@mui/material/Typography'
import { Link as RouterLink, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useLogout, useMe } from '../api/hooks'
import { LanguageSwitcher } from './LanguageSwitcher'
import { NotificationBell } from './NotificationBell'

export function Header() {
  return (
    <Box sx={{ flexGrow: 1 }}>
      <AppBar position="static">
        <Toolbar>
          <Typography variant="h6" component="div" sx={{ flexGrow: 1 }}>
            <Link component={RouterLink} to="/" underline="none" color="inherit">
              Striker
            </Link>
          </Typography>
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
        <MenuItem component={RouterLink} to="/dashboard" onClick={() => setAnchorEl(null)}>
          {t('header.nav.dashboard')}
        </MenuItem>
        <MenuItem component={RouterLink} to="/dashboard/lineup" onClick={() => setAnchorEl(null)}>
          {t('header.nav.lineup')}
        </MenuItem>
        <MenuItem component={RouterLink} to="/dashboard/finances" onClick={() => setAnchorEl(null)}>
          {t('header.nav.finances')}
        </MenuItem>
        <MenuItem component={RouterLink} to="/transfers" onClick={() => setAnchorEl(null)}>
          {t('header.nav.transferMarket')}
        </MenuItem>
        <MenuItem component={RouterLink} to="/cup" onClick={() => setAnchorEl(null)}>
          {t('header.nav.cup')}
        </MenuItem>
        <MenuItem component={RouterLink} to="/groups" onClick={() => setAnchorEl(null)}>
          {t('header.nav.friendGroups')}
        </MenuItem>
        {loggedUser.isAdmin && (
          <MenuItem component={RouterLink} to="/admin" onClick={() => setAnchorEl(null)}>
            {t('header.nav.admin')}
          </MenuItem>
        )}
        <MenuItem onClick={handleLogout}>{t('header.nav.logout')}</MenuItem>
      </Menu>
    </Box>
  )
}
