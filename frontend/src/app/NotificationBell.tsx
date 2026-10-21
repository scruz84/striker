import { useState } from 'react'
import Badge from '@mui/material/Badge'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import Divider from '@mui/material/Divider'
import Drawer from '@mui/material/Drawer'
import IconButton from '@mui/material/IconButton'
import List from '@mui/material/List'
import ListItem from '@mui/material/ListItem'
import ListItemButton from '@mui/material/ListItemButton'
import ListItemText from '@mui/material/ListItemText'
import ListSubheader from '@mui/material/ListSubheader'
import Menu from '@mui/material/Menu'
import Stack from '@mui/material/Stack'
import Switch from '@mui/material/Switch'
import Tooltip from '@mui/material/Tooltip'
import Typography from '@mui/material/Typography'
import { useTheme } from '@mui/material/styles'
import useMediaQuery from '@mui/material/useMediaQuery'
import ArrowBackIcon from '@mui/icons-material/ArrowBack'
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
  const theme = useTheme()
  const isDesktop = useMediaQuery(theme.breakpoints.up('md'))

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

  const settingsView = (
    <List disablePadding>
      <ListSubheader disableSticky sx={{ lineHeight: '32px' }}>
        {t('notifications.deliverBy')}
      </ListSubheader>
      <ListItem
        secondaryAction={
          <Switch
            edge="end"
            checked={prefs?.emailEnabled ?? false}
            disabled={!prefs || setPrefs.isPending}
            onChange={(event) => prefs && setPrefs.mutate({ ...prefs, emailEnabled: event.target.checked })}
            slotProps={{ input: { 'aria-label': t('notifications.email') } }}
          />
        }
      >
        <ListItemText primary={t('notifications.email')} />
      </ListItem>
      <ListItem
        secondaryAction={
          <Switch
            edge="end"
            checked={prefs?.pushEnabled ?? false}
            disabled={
              !prefs ||
              !pushSupported ||
              setPrefs.isPending ||
              subscribeToPush.isPending ||
              unsubscribeFromPush.isPending
            }
            onChange={(event) => void handlePushToggle(event.target.checked)}
            slotProps={{ input: { 'aria-label': t('notifications.push') } }}
          />
        }
      >
        <ListItemText primary={t('notifications.push')} secondary={pushSupported ? undefined : t('notifications.pushUnsupported')} />
      </ListItem>
      {subscribeToPush.isError && (
        <ListItem>
          <Typography variant="caption" color="error">
            {t('notifications.pushPermissionDenied')}
          </Typography>
        </ListItem>
      )}
    </List>
  )

  const listView = (
    <>
      {notifications.length === 0 && (
        <Box sx={{ px: 2, py: 3, textAlign: 'center' }}>
          <Typography variant="body2" color="text.secondary">
            {t('notifications.empty')}
          </Typography>
        </Box>
      )}
      <List disablePadding>
        {notifications.map((notification) => (
          <ListItem
            key={notification.notificationId}
            disablePadding
            divider
            secondaryAction={
              <IconButton
                edge="end"
                aria-label={t('notifications.delete')}
                onClick={() => deleteNotification.mutate(notification.notificationId)}
              >
                <CloseIcon fontSize="small" />
              </IconButton>
            }
            sx={{ backgroundColor: notification.read ? 'transparent' : 'action.hover' }}
          >
            <ListItemButton onClick={() => handleItemClick(notification)} sx={{ pr: 7, whiteSpace: 'normal' }}>
              <ListItemText
                primary={notification.messageKey ? t(notification.messageKey, notification.messageParams) : notification.message}
                secondary={formatDateTime(notification.createdAt)}
                slotProps={{ primary: { variant: 'body2' }, secondary: { variant: 'caption' } }}
              />
            </ListItemButton>
          </ListItem>
        ))}
      </List>
    </>
  )

  const actions = (
    <Stack direction="row" spacing={0.5} sx={{ alignItems: 'center', flexWrap: 'wrap', justifyContent: 'flex-end' }}>
      {!settingsOpen && unread > 0 && (
        <Button size="small" onClick={() => markAllRead.mutate()}>
          {t('notifications.markAllRead')}
        </Button>
      )}
      {!settingsOpen && notifications.length > 0 && (
        <Button size="small" color="inherit" onClick={() => deleteAllNotifications.mutate()}>
          {t('notifications.clearAll')}
        </Button>
      )}
      <Tooltip title={t('notifications.deliveryPreferences')}>
        <IconButton
          aria-label={t('notifications.deliveryPreferences')}
          color={settingsOpen ? 'primary' : 'default'}
          onClick={() => setSettingsOpen((open) => !open)}
        >
          <SettingsIcon fontSize="small" />
        </IconButton>
      </Tooltip>
    </Stack>
  )

  const close = () => {
    setAnchorEl(null)
    setSettingsOpen(false)
  }

  return (
    <Box sx={{ flexGrow: 0 }}>
      <Tooltip title={t('notifications.title')}>
        <IconButton
          onClick={(event) => setAnchorEl(event.currentTarget)}
          color="inherit"
          aria-label={t('header.notifications')}
        >
          <Badge badgeContent={unread} color="error">
            <NotificationsIcon />
          </Badge>
        </IconButton>
      </Tooltip>
      {isDesktop ? (
        <Menu
          sx={{ mt: '8px' }}
          slotProps={{ paper: { sx: { width: 380, maxWidth: '100%' } } }}
          anchorEl={anchorEl}
          anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
          transformOrigin={{ vertical: 'top', horizontal: 'right' }}
          open={Boolean(anchorEl)}
          onClose={close}
        >
          <Box sx={{ px: 2, py: 1, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <Typography variant="subtitle2">{t('notifications.title')}</Typography>
            {actions}
          </Box>
          <Divider />
          {settingsOpen ? settingsView : listView}
        </Menu>
      ) : (
        // A full-screen sheet on a phone: a 380px popover hugs the screen edge and truncates the
        // delivery settings, so the settings get a screen of their own with right-aligned switches.
        <Drawer
          anchor="right"
          open={Boolean(anchorEl)}
          onClose={close}
          slotProps={{ paper: { sx: { width: '100%', maxWidth: '100%' } } }}
        >
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5, px: 1, minHeight: 56, borderBottom: 1, borderColor: 'divider' }}>
            <IconButton
              edge="start"
              onClick={settingsOpen ? () => setSettingsOpen(false) : close}
              aria-label={settingsOpen ? t('common.back') : t('common.close')}
            >
              {settingsOpen ? <ArrowBackIcon /> : <CloseIcon />}
            </IconButton>
            <Typography variant="subtitle1" sx={{ fontWeight: 600, flexGrow: 1 }} noWrap>
              {settingsOpen ? t('notifications.deliveryPreferences') : t('notifications.title')}
            </Typography>
            {actions}
          </Box>
          <Box sx={{ overflowY: 'auto', pb: 'env(safe-area-inset-bottom)' }}>{settingsOpen ? settingsView : listView}</Box>
        </Drawer>
      )}
    </Box>
  )
}
