-- Per-match lineup override for a friendly, so a manager can set a different XI for each
-- upcoming friendly instead of always playing their team's standing default lineup - see
-- common.GetLineupsForFriendly, common.SetFriendlyLineup, handlers.SetFriendlyLineupHandler.
--
-- Deliberately NOT built on `lineup`/`lineup_positions` (init28_lineup_substitutions.sql):
-- those are keyed by `match_id` with a foreign key to `matches(match_id)` (competitive league
-- matches only) - friendly_matches has its own id sequence, so writing a friendly's override
-- there would either violate that foreign key or, worse, silently collide with an unrelated
-- competitive match that happens to share the same id (see engine/simulation.go's
-- runMatchSimulation comment on why that collision has been safe so far: nothing has ever
-- written to `lineup` at all). A friendly gets its own, identically-shaped pair of tables
-- instead, so this can never happen.
-- formation is stored per friendly (rather than always reusing the team's standing formation -
-- see teams.formation) so a manager can pick a different shape for one friendly without
-- changing anything about their competitive lineup.
CREATE TABLE IF NOT EXISTS friendly_lineup (
    lineup_id bigserial PRIMARY KEY,
    match_id  bigint NOT NULL REFERENCES friendly_matches(match_id),
    team_id   bigint NOT NULL REFERENCES teams(team_id),
    formation smallint NOT NULL DEFAULT 0,
    UNIQUE (match_id, team_id)
);

CREATE TABLE IF NOT EXISTS friendly_lineup_positions (
    lineup_id bigint NOT NULL REFERENCES friendly_lineup(lineup_id),
    player_id bigint REFERENCES players(player_id) ON DELETE SET NULL,
    position  smallint NOT NULL
);

CREATE INDEX IF NOT EXISTS friendly_lineup_positions_idx1 ON friendly_lineup_positions(lineup_id);
