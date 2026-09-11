CREATE TABLE IF NOT EXISTS notification_preferences (
    user_id       bigint NOT NULL PRIMARY KEY REFERENCES users(user_id) ON DELETE CASCADE,
    email_enabled boolean NOT NULL DEFAULT false,
    push_enabled  boolean NOT NULL DEFAULT false,
    updated_at    timestamptz NOT NULL DEFAULT now()
);
