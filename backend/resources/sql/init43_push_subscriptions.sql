-- A browser's Web Push subscription (see PushManager.subscribe on the frontend) for one of a
-- user's devices/browsers - a user can have several (phone, laptop, ...), each getting its own
-- row, so a notification goes out to every device they've enabled push on. endpoint is unique
-- per subscription (the browser's own push service URL), so re-subscribing the same
-- device/browser just replaces its keys rather than accumulating duplicates. Deleted once its
-- endpoint starts reporting expired/unsubscribed (a 404/410 from the push service - see
-- common.SendWebPush) or the user disables push in their notification preferences.
CREATE TABLE IF NOT EXISTS push_subscriptions (
    subscription_id bigserial PRIMARY KEY,
    user_id         bigint NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    endpoint        varchar(500) NOT NULL UNIQUE,
    p256dh          varchar(255) NOT NULL,
    auth            varchar(255) NOT NULL,
    created_at      timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_push_subscriptions_user_id ON push_subscriptions(user_id);
