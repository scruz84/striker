-- Lets a notification carry a frontend i18n key + params alongside its English message text, so
-- the notification bell can render it in whichever language the viewer has chosen instead of
-- always English (see common.NotificationMessage). Both columns are nullable and simply absent
-- for any notification created before this shipped - the frontend falls back to the plain
-- message text in that case, same as it always has.
ALTER TABLE notifications ADD COLUMN IF NOT EXISTS message_key varchar(64);
ALTER TABLE notifications ADD COLUMN IF NOT EXISTS message_params jsonb;
