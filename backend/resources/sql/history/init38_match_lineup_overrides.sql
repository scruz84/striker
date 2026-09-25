-- Puts the existing, previously-dormant lineup/lineup_positions tables to use as a per-league-
-- match lineup override, and adds an equivalent pair for cup ties.
--
-- lineup/lineup_positions already existed (keyed by matches(match_id)) and common.
-- GetLineupsForMatch already reads them with a fallback to default_lineup_positions - nothing
-- had ever written to them, since only formation was missing to make them a complete mirror of
-- friendly_lineup/friendly_lineup_positions. Adding it here is enough to light the feature up.
ALTER TABLE lineup ADD COLUMN IF NOT EXISTS formation smallint NOT NULL DEFAULT 0;

-- Cup ties can NOT reuse lineup/lineup_positions the way league matches now do: cup_ties.tie_id
-- is its own independent bigserial sequence, unrelated to matches.match_id, so a tie and a
-- league match can easily share the same numeric id - engine.runMatchSimulation's own doc
-- comment already flagged this as "safe today but would stop being safe" the moment anything
-- started writing match-keyed override rows, which this migration's ALTER above is exactly. Cup
-- ties get their own tables instead, identically shaped, the same reasoning
-- friendly_lineup/friendly_lineup_positions already established for friendly_matches.match_id
-- vs matches.match_id.
CREATE TABLE IF NOT EXISTS cup_tie_lineup (
    lineup_id bigserial PRIMARY KEY,
    tie_id    bigint NOT NULL REFERENCES cup_ties(tie_id),
    team_id   bigint NOT NULL REFERENCES teams(team_id),
    formation smallint NOT NULL DEFAULT 0,
    UNIQUE (tie_id, team_id)
);

CREATE TABLE IF NOT EXISTS cup_tie_lineup_positions (
    lineup_id bigint NOT NULL REFERENCES cup_tie_lineup(lineup_id),
    player_id bigint REFERENCES players(player_id) ON DELETE SET NULL,
    position  smallint NOT NULL
);

CREATE INDEX IF NOT EXISTS cup_tie_lineup_positions_idx1 ON cup_tie_lineup_positions(lineup_id);
