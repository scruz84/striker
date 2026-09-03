import { useState } from 'react'
import Badge from '@mui/material/Badge'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import Divider from '@mui/material/Divider'
import IconButton from '@mui/material/IconButton'
import ListItemText from '@mui/material/ListItemText'
import Menu from '@mui/material/Menu'
import MenuItem from '@mui/material/MenuItem'
import Tooltip from '@mui/material/Tooltip'
import Typography from '@mui/material/Typography'
import NotificationsIcon from '@mui/icons-material/Notifications'
import { useMarkAllNotificationsRead, useMarkNotificationRead, useMe, useNotifications } from '../api/hooks'
import type { Notification } from '../api/types'
import { formatDateTime } from '../utils/formatDate'

export function NotificationBell() {
  const { data: loggedUser } = useMe()
  const { data } = useNotifications(!!loggedUser)
  const markRead = useMarkNotificationRead()
  const markAllRead = useMarkAllNotificationsRead()
  const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null)

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
          {unread > 0 && (
            <Button size="small" onClick={() => markAllRead.mutate()}>
              Mark all read
            </Button>
          )}
        </Box>
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
