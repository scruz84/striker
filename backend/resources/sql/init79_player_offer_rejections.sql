-- Remembers transfer offers that were turned down (or lapsed unanswered) so a CPU team that made
-- the offer doesn't hammer the same player again straight away: see common.RecordOfferRejection,
-- written by RejectOffer and by ExpireStaleOffers, and read by engine's AI market pass, which
-- waits out common.AiOfferCooldown and then only returns with a better bid. The offer row itself
-- is deleted the moment it resolves, so this is the only trace left of it.
CREATE TABLE IF NOT EXISTS player_offer_rejections (
    rejection_id bigserial PRIMARY KEY,
    player_id    bigint NOT NULL REFERENCES players(player_id) ON DELETE CASCADE,
    from_team_id bigint NOT NULL REFERENCES teams(team_id) ON DELETE CASCADE,
    amount       bigint NOT NULL,
    rejected_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_player_offer_rejections_team ON player_offer_rejections(from_team_id, rejected_at DESC);
CREATE INDEX IF NOT EXISTS idx_player_offer_rejections_rejected_at ON player_offer_rejections(rejected_at);
