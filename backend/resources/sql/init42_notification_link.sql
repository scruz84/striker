-- Optional in-app route a notification click-throughs to (e.g. "/players/evolutions") - NULL for
-- every existing notification type, which stay exactly as before (mark-as-read only, see
-- NotificationBell). First user: NOTIFICATION_PROGRESSION, whose message alone can't point a
-- viewer at the squad-wide "latest evolutions" page the way a single match/transfer notification
-- doesn't need to (those already say everything relevant right in the message).
ALTER TABLE notifications ADD COLUMN IF NOT EXISTS link VARCHAR(200);
