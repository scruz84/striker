-- Direct transfer offers ("scouting"): unlike transfer_listings (a seller advertises a fixed
-- price), here a buyer proposes an amount for any player, listed or not, and the current owner
-- accepts or rejects. Every row is a pending offer - accepted/rejected/withdrawn ones are
-- deleted rather than archived, same as transfer_listings/loan_listings. See common/offers.go.

CREATE TABLE IF NOT EXISTS transfer_offers (
    offer_id      bigserial PRIMARY KEY,
    player_id     bigint NOT NULL REFERENCES players(player_id) ON DELETE CASCADE,
    from_team_id  bigint NOT NULL REFERENCES teams(team_id),
    to_team_id    bigint NOT NULL REFERENCES teams(team_id),
    amount        bigint NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    UNIQUE (player_id, from_team_id)
);

CREATE INDEX IF NOT EXISTS transfer_offers_idx1 ON transfer_offers(to_team_id);
CREATE INDEX IF NOT EXISTS transfer_offers_idx2 ON transfer_offers(from_team_id);
