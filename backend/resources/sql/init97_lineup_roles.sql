-- Lineup roles (see openspec/specs/match-simulation "Lineup roles" and common.LineupRoles): an
-- optional captain, penalty taker and free-kick taker on every lineup. The three per-fixture
-- lineup tables carry them as columns (so the kickoff snapshot keeps them); the two standing
-- defaults (league/cup and friendly) keep theirs in default_lineup_roles. A deleted player leaves
-- the role empty ("automatic"). Re-runnable.
ALTER TABLE lineup ADD COLUMN IF NOT EXISTS captain_id bigint NULL REFERENCES players(player_id) ON DELETE SET NULL;
ALTER TABLE lineup ADD COLUMN IF NOT EXISTS penalty_taker_id bigint NULL REFERENCES players(player_id) ON DELETE SET NULL;
ALTER TABLE lineup ADD COLUMN IF NOT EXISTS free_kick_taker_id bigint NULL REFERENCES players(player_id) ON DELETE SET NULL;

ALTER TABLE cup_tie_lineup ADD COLUMN IF NOT EXISTS captain_id bigint NULL REFERENCES players(player_id) ON DELETE SET NULL;
ALTER TABLE cup_tie_lineup ADD COLUMN IF NOT EXISTS penalty_taker_id bigint NULL REFERENCES players(player_id) ON DELETE SET NULL;
ALTER TABLE cup_tie_lineup ADD COLUMN IF NOT EXISTS free_kick_taker_id bigint NULL REFERENCES players(player_id) ON DELETE SET NULL;

ALTER TABLE friendly_lineup ADD COLUMN IF NOT EXISTS captain_id bigint NULL REFERENCES players(player_id) ON DELETE SET NULL;
ALTER TABLE friendly_lineup ADD COLUMN IF NOT EXISTS penalty_taker_id bigint NULL REFERENCES players(player_id) ON DELETE SET NULL;
ALTER TABLE friendly_lineup ADD COLUMN IF NOT EXISTS free_kick_taker_id bigint NULL REFERENCES players(player_id) ON DELETE SET NULL;

CREATE TABLE IF NOT EXISTS default_lineup_roles (
    team_id            bigint NOT NULL REFERENCES teams(team_id) ON DELETE CASCADE,
    friendly           boolean NOT NULL,
    captain_id         bigint NULL REFERENCES players(player_id) ON DELETE SET NULL,
    penalty_taker_id   bigint NULL REFERENCES players(player_id) ON DELETE SET NULL,
    free_kick_taker_id bigint NULL REFERENCES players(player_id) ON DELETE SET NULL,
    PRIMARY KEY (team_id, friendly)
);
