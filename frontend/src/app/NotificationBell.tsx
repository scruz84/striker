import { useState } from 'react'
import Badge from '@mui/material/Badge'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import Divider from '@mui/material/Divider'
import FormControlLabel from '@mui/material/FormControlLabel'
import IconButton from '@mui/material/IconButton'
import ListItemText from '@mui/material/ListItemText'
import Menu from '@mui/material/Menu'
import MenuItem from '@mui/material/MenuItem'
import Switch from '@mui/material/Switch'
import Tooltip from '@mui/material/Tooltip'
import Typography from '@mui/material/Typography'
import NotificationsIcon from '@mui/icons-material/Notifications'
import SettingsIcon from '@mui/icons-material/Settings'
import {
  useMarkAllNotificationsRead,
  useMarkNotificationRead,
  useMe,
  useNotificationPreferences,
  useNotifications,
  useSetNotificationPreferences,
} from '../api/hooks'
import type { Notification } from '../api/types'
import { formatDateTime } from '../utils/formatDate'

export function NotificationBell() {
  const { data: loggedUser } = useMe()
  const { data } = useNotifications(!!loggedUser)
  const markRead = useMarkNotificationRead()
  const markAllRead = useMarkAllNotificationsRead()
  const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null)
  const [settingsOpen, setSettingsOpen] = useState(false)
  const { data: prefs } = useNotificationPreferences(!!loggedUser && settingsOpen)
  const setPrefs = useSetNotificationPreferences()

  if (!loggedUser) {
    return null
  }

  const unread = data?.unread ?? 0
  const notifications = data?.notifications ?? []

  const handleItemClick = (notification: Notification) => {
    if (!notification.read) {
      markRead.mutate(notification.notificationId)
    }
  }

  return (
    <Box sx={{ flexGrow: 0 }}>
      <Tooltip title="Notifications">
        <IconButton onClick={(event) => setAnchorEl(event.currentTarget)} color="inherit">
          <Badge badgeContent={unread} color="error">
            <NotificationsIcon />
          </Badge>
        </IconButton>
      </Tooltip>
      <Menu
        sx={{ mt: '45px' }}
        slotProps={{ paper: { sx: { width: 380, maxWidth: '100%' } } }}
        anchorEl={anchorEl}
        anchorOrigin={{ vertical: 'top', horizontal: 'right' }}
        transformOrigin={{ vertical: 'top', horizontal: 'right' }}
        open={Boolean(anchorEl)}
        onClose={() => setAnchorEl(null)}
      >
        <Box sx={{ px: 2, py: 1, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Typography variant="subtitle2">Notifications</Typography>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            {unread > 0 && (
              <Button size="small" onClick={() => markAllRead.mutate()}>
                Mark all read
              </Button>
            )}
            <Tooltip title="Delivery preferences">
              <IconButton size="small" onClick={() => setSettingsOpen((open) => !open)}>
                <SettingsIcon fontSize="small" />
              </IconButton>
            </Tooltip>
          </Box>
        </Box>
        {settingsOpen && (
          <>
            <Divider />
            <Box sx={{ px: 2, py: 1 }}>
              <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 0.5 }}>
                Also deliver notifications by:
              </Typography>
              <FormControlLabel
                control={
                  <Switch
                    size="small"
                    checked={prefs?.emailEnabled ?? false}
                    disabled={!prefs || setPrefs.isPending}
                    onChange={(event) =>
                      prefs && setPrefs.mutate({ ...prefs, emailEnabled: event.target.checked })
                    }
                  />
                }
                label={<Typography variant="body2">Email</Typography>}
              />
              <FormControlLabel
                sx={{ display: 'block' }}
                control={
                  <Switch
                    size="small"
                    checked={prefs?.pushEnabled ?? false}
                    disabled={!prefs || setPrefs.isPending}
                    onChange={(event) =>
                      prefs && setPrefs.mutate({ ...prefs, pushEnabled: event.target.checked })
                    }
                  />
                }
                label={<Typography variant="body2">Push</Typography>}
              />
            </Box>
          </>
        )}
        <Divider />
        {notifications.length === 0 && (
          <MenuItem disabled>
            <Typography variant="body2" color="text.secondary">
              No notifications yet.
            </Typography>
          </MenuItem>
        )}
        {notifications.map((notification) => (
          <MenuItem
            key={notification.notificationId}
            onClick={() => handleItemClick(notification)}
            sx={{ whiteSpace: 'normal', backgroundColor: notification.read ? 'transparent' : 'action.hover' }}
          >
            <ListItemText
              primary={notification.message}
              secondary={formatDateTime(notification.createdAt)}
              slotProps={{ primary: { variant: 'body2' }, secondary: { variant: 'caption' } }}
            />
          </MenuItem>
        ))}
      </Menu>
    </Box>
  )
}
