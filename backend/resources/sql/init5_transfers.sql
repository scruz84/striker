-- Transfer market listings. A listing is bought instantly at its asking price (no bidding/
-- offer negotiation) - see handlers/transferHandlers.go and common.ExecuteTransfer.
--
-- Runs automatically after init4_finances.sql on a fresh database. On an already-initialized
-- one, apply it manually once - see backend/README.md.

CREATE TABLE IF NOT EXISTS transfer_listings (
    listing_id      bigserial PRIMARY KEY,
    player_id       bigint NOT NULL UNIQUE REFERENCES players(player_id) ON DELETE CASCADE,
    selling_team_id bigint NOT NULL REFERENCES teams(team_id),
    asking_price    bigint NOT NULL,
    listed_at       timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS transfer_listings_idx1 ON transfer_listings(selling_team_id);
