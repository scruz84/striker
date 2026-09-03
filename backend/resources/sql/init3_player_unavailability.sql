-- Tracks players currently unavailable through suspension (red card / second yellow) or
-- injury, and for how many more of their team's matches. See engine.applyUnavailability and
-- handlers.validateLineup for how this is populated, decremented, and enforced.
--
-- Runs automatically after init.sql and init2_league_tier.sql on a fresh database. On an
-- already-initialized one, apply it manually once - see backend/README.md.

CREATE TABLE IF NOT EXISTS player_unavailability (
    player_id        bigint PRIMARY KEY REFERENCES players(player_id) ON DELETE CASCADE,
    reason            smallint NOT NULL,
    matches_remaining int NOT NULL
);
