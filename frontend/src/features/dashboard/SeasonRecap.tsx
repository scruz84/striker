import Alert from '@mui/material/Alert'
import AlertTitle from '@mui/material/AlertTitle'
import Stack from '@mui/material/Stack'
import { useTranslation } from 'react-i18next'
import { useMarkNotificationRead, useNotifications } from '../../api/hooks'
import { NOTIFICATION_AWARD, NOTIFICATION_CUP, NOTIFICATION_PROMOTION, NOTIFICATION_RELEGATION, type Notification } from '../../api/types'

// Promotion/relegation, cup, and award (Golden Boot / Player of the Season) notifications are
// all fired exactly at season rollover (see backend/engine/season.go, cups.go, and awards.go),
// so the unread ones among them are, in effect, this team's season recap: what changed for them
// at the moment the season turned over. There's no separate "final standings" to show alongside
// it - classification is reset in place for the new season rather than archived, so last
// season's table no longer exists once the season has rolled over.
const SEASON_RECAP_TYPES = new Set([NOTIFICATION_PROMOTION, NOTIFICATION_RELEGATION, NOTIFICATION_CUP, NOTIFICATION_AWARD])

function severityFor(notificationType: number): 'success' | 'warning' | 'info' {
  if (notificationType === NOTIFICATION_PROMOTION) return 'success'
  if (notificationType === NOTIFICATION_RELEGATION) return 'warning'
  return 'info'
}

// Golden Boot and Player of the Season share NOTIFICATION_AWARD (same reasoning as cup tie
// results and the champion sharing NOTIFICATION_CUP) - distinguished by messageKey instead.
function titleKeyFor(notification: Notification): string {
  if (notification.notificationType === NOTIFICATION_PROMOTION) return 'seasonRecap.promoted'
  if (notification.notificationType === NOTIFICATION_RELEGATION) return 'seasonRecap.relegated'
  if (notification.notificationType === NOTIFICATION_AWARD) {
    return notification.messageKey === 'notifications.messages.goldenBoot' ? 'seasonRecap.goldenBoot' : 'seasonRecap.playerOfSeason'
  }
  return 'seasonRecap.cup'
}

export function SeasonRecap({ enabled }: { enabled: boolean }) {
  const { t } = useTranslation()
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
          <AlertTitle>{t(titleKeyFor(notification))}</AlertTitle>
          {notification.messageKey ? t(notification.messageKey, notification.messageParams) : notification.message}
        </Alert>
      ))}
    </Stack>
  )
}
