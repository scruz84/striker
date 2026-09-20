-- Player page-view counter, incremented every time PlayerDetailsHandler serves that player's
-- profile - lets the transfer market stats panel surface the most-viewed listed player (issue
-- #72).
ALTER TABLE players ADD COLUMN IF NOT EXISTS view_count bigint NOT NULL DEFAULT 0;

-- Append-only log of transfer offers actually made (see common.MakeTransferOffer) - unlike
-- transfer_offers itself, which is deleted the moment an offer resolves (accepted/rejected/
-- withdrawn), this never shrinks, so "how many offers were made in the last hour/day" (issue
-- #72) stays answerable regardless of how quickly offers get resolved.
CREATE TABLE IF NOT EXISTS transfer_offer_log (
    event_id   bigserial PRIMARY KEY,
    created_at timestamptz NOT NULL DEFAULT now()
);
