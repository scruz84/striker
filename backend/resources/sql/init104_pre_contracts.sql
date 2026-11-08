-- Pre-contracts (openspec/specs/pre-contracts): a club's offer of a contract for next season to
-- another club's player in the last season of his contract. A row is pending until signed_at is
-- set; a player signs at most one (the partial unique index), and joins from_team_id at his
-- country's next season rollover. reminded_at marks, on a player's first pending offer, that his
-- club was reminded the decision is near. Re-runnable.
CREATE TABLE IF NOT EXISTS pre_contract_offers (
    offer_id     bigserial PRIMARY KEY,
    player_id    bigint NOT NULL REFERENCES players (player_id) ON DELETE CASCADE,
    from_team_id bigint NOT NULL REFERENCES teams (team_id) ON DELETE CASCADE,
    wage         int NOT NULL,
    years        int NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    signed_at    timestamptz,
    reminded_at  timestamptz,
    UNIQUE (player_id, from_team_id)
);

CREATE UNIQUE INDEX IF NOT EXISTS pre_contract_offers_one_signed_per_player
    ON pre_contract_offers (player_id) WHERE signed_at IS NOT NULL;

CREATE INDEX IF NOT EXISTS pre_contract_offers_from_team_idx ON pre_contract_offers (from_team_id);
