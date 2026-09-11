-- Per-user notification log (match results, promotion/relegation, transfers, cup results).
-- See common/notifications.go for the generation points and engine wiring.
--
-- Runs automatically after init7_cups.sql on a fresh database. On an already-initialized one,
-- apply it manually once - see backend/README.md.

CREATE TABLE IF NOT EXISTS notifications (
    notification_id   bigserial PRIMARY KEY,
    user_id            bigint NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    notification_type  smallint NOT NULL,
    message            varchar(1000) NOT NULL,
    read               boolean NOT NULL DEFAULT false,
    created_at         timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS notifications_idx1 ON notifications(user_id, created_at DESC);
