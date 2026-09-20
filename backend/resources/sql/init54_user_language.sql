-- Lets a user's chosen UI language (see frontend LanguageSwitcher.tsx) be known server-side too,
-- so external notification delivery (email/push - see common.deliverExternalNotifications) can
-- render in that language instead of always English. NULL (unset) means "no preference recorded
-- yet" - falls back to English, same as before this column existed.
ALTER TABLE users ADD COLUMN IF NOT EXISTS language varchar(5);
