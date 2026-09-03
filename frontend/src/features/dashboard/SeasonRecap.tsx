import Alert from '@mui/material/Alert'
import AlertTitle from '@mui/material/AlertTitle'
import Stack from '@mui/material/Stack'
import { useMarkNotificationRead, useNotifications } from '../../api/hooks'
import { NOTIFICATION_CUP, NOTIFICATION_PROMOTION, NOTIFICATION_RELEGATION, type Notification } from '../../api/types'

// Promotion/relegation and cup notifications are fired exactly at season rollover (see
// backend/engine/season.go and cups.go), so the unread ones among them are, in effect, this
// team's season recap: what changed for them at the moment the season turned over. There's no
// separate "final standings" to show alongside it - classification is reset in place for the
// new season rather than archived, so last season's table no longer exists once the season has
// rolled over.
const SEASON_RECAP_TYPES = new Set([NOTIFICATION_PROMOTION, NOTIFICATION_RELEGATION, NOTIFICATION_CUP])

function severityFor(notificationType: number): 'success' | 'warning' | 'info' {
  if (notificationType === NOTIFICATION_PROMOTION) return 'success'
  if (notificationType === NOTIFICATION_RELEGATION) return 'warning'
  return 'info'
}

function titleFor(notificationType: number): string {
  if (notificationType === NOTIFICATION_PROMOTION) return 'Promoted!'
  if (notificationType === NOTIFICATION_RELEGATION) return 'Relegated'
  return 'Cup'
}

export function SeasonRecap({ enabled }: { enabled: boolean }) {
  const { data } = useNotifications(enabled)
  const markRead = useMarkNotificationRead()

  const recap = (data?.notifications ?? []).filter(
    (n): n is Notification => !n.read && SEASON_RECAP_TYPES.has(n.notificationType),
  )

  if (recap.length === 0) {
    return null
  }

  return (
    <Stack spacing={1} sx={{ mb: 2 }}>
      {recap.map((notification) => (
        <Alert
          key={notification.notificationId}
          severity={severityFor(notification.notificationType)}
          onClose={() => markRead.mutate(notification.notificationId)}
        >
          <AlertTitle>{titleFor(notification.notificationType)}</AlertTitle>
          {notification.message}
        </Alert>
      ))}
    </Stack>
  )
}
