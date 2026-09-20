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
import Stack from '@mui/material/Stack'
import Switch from '@mui/material/Switch'
import Tooltip from '@mui/material/Tooltip'
import Typography from '@mui/material/Typography'
import CloseIcon from '@mui/icons-material/Close'
import NotificationsIcon from '@mui/icons-material/Notifications'
import SettingsIcon from '@mui/icons-material/Settings'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  useDeleteAllNotifications,
  useDeleteNotification,
  useMarkAllNotificationsRead,
  useMarkNotificationRead,
  useMe,
  useNotificationPreferences,
  useNotifications,
  useSetNotificationPreferences,
  useSubscribeToPush,
  useUnsubscribeFromPush,
} from '../api/hooks'
import type { Notification } from '../api/types'
import { isPushSupported } from '../pushNotifications'
import { formatDateTime } from '../utils/formatDate'

export function NotificationBell() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { data: loggedUser } = useMe()
  const { data } = useNotifications(!!loggedUser)
  const markRead = useMarkNotificationRead()
  const markAllRead = useMarkAllNotificationsRead()
  const deleteNotification = useDeleteNotification()
  const deleteAllNotifications = useDeleteAllNotifications()
  const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null)
  const [settingsOpen, setSettingsOpen] = useState(false)
  const { data: prefs } = useNotificationPreferences(!!loggedUser && settingsOpen)
  const setPrefs = useSetNotificationPreferences()
  const subscribeToPush = useSubscribeToPush()
  const unsubscribeFromPush = useUnsubscribeFromPush()
  const pushSupported = isPushSupported()

  if (!loggedUser) {
    return null
  }

  const unread = data?.unread ?? 0
  const notifications = data?.notifications ?? []

  const handlePushToggle = async (enabled: boolean) => {
    if (!prefs) {
      return
    }
    if (enabled) {
      try {
        await subscribeToPush.mutateAsync()
      } catch {
        // Permission denied, or the browser/service worker failed to subscribe - leave the
        // preference off rather than claiming push is on when nothing was actually registered.
        return
      }
    } else {
      await unsubscribeFromPush.mutateAsync()
    }
    setPrefs.mutate({ ...prefs, pushEnabled: enabled })
  }

  const handleItemClick = (notification: Notification) => {
    if (!notification.read) {
      markRead.mutate(notification.notificationId)
    }
    if (notification.link) {
      setAnchorEl(null)
      navigate(notification.link)
    }
  }

  return (
    <Box sx={{ flexGrow: 0 }}>
      <Tooltip title={t('notifications.title')}>
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
          <Typography variant="subtitle2">{t('notifications.title')}</Typography>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            {unread > 0 && (
              <Button size="small" onClick={() => markAllRead.mutate()}>
                {t('notifications.markAllRead')}
              </Button>
            )}
            {notifications.length > 0 && (
              <Button size="small" color="inherit" onClick={() => deleteAllNotifications.mutate()}>
                {t('notifications.clearAll')}
              </Button>
            )}
            <Tooltip title={t('notifications.deliveryPreferences')}>
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
                {t('notifications.deliverBy')}
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
                label={<Typography variant="body2">{t('notifications.email')}</Typography>}
              />
              <Tooltip title={pushSupported ? '' : t('notifications.pushUnsupported')}>
                <FormControlLabel
                  sx={{ display: 'block' }}
                  control={
                    <Switch
                      size="small"
                      checked={prefs?.pushEnabled ?? false}
                      disabled={
                        !prefs ||
                        !pushSupported ||
                        setPrefs.isPending ||
                        subscribeToPush.isPending ||
                        unsubscribeFromPush.isPending
                      }
                      onChange={(event) => void handlePushToggle(event.target.checked)}
                    />
                  }
                  label={<Typography variant="body2">{t('notifications.push')}</Typography>}
                />
              </Tooltip>
              {subscribeToPush.isError && (
                <Typography variant="caption" color="error" sx={{ display: 'block' }}>
                  {t('notifications.pushPermissionDenied')}
                </Typography>
              )}
            </Box>
          </>
        )}
        <Divider />
        {notifications.length === 0 && (
          <MenuItem disabled>
            <Typography variant="body2" color="text.secondary">
              {t('notifications.empty')}
            </Typography>
          </MenuItem>
        )}
        {notifications.map((notification) => (
          <MenuItem
            key={notification.notificationId}
            onClick={() => handleItemClick(notification)}
            sx={{ whiteSpace: 'normal', backgroundColor: notification.read ? 'transparent' : 'action.hover' }}
          >
            <Stack direction="row" spacing={0.5} sx={{ width: '100%', alignItems: 'flex-start' }}>
              <ListItemText
                primary={notification.messageKey ? t(notification.messageKey, notification.messageParams) : notification.message}
                secondary={formatDateTime(notification.createdAt)}
                slotProps={{ primary: { variant: 'body2' }, secondary: { variant: 'caption' } }}
              />
              <Tooltip title={t('notifications.delete')}>
                <IconButton
                  size="small"
                  edge="end"
                  onClick={(event) => {
                    event.stopPropagation()
                    deleteNotification.mutate(notification.notificationId)
                  }}
                >
                  <CloseIcon fontSize="small" />
                </IconButton>
              </Tooltip>
            </Stack>
          </MenuItem>
        ))}
      </Menu>
    </Box>
  )
}
