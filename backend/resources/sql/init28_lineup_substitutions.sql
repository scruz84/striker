-- A manager's planned in-match substitutions for their default lineup - up to 3, each swapping
-- out a current starter for a bench player at a chosen minute (see common.SetDefaultLineupSubstitutions,
-- engine.SubstitutionSimulation). Deliberately separate from default_lineup_positions since a
-- substitution references two players and a minute, not a single position slot.
--
-- slot is 1-3, assigned by save order (not user-chosen) - just enough to give each of a team's
-- up-to-3 planned substitutions its own row under the (team_id, slot) primary key.
CREATE TABLE IF NOT EXISTS default_lineup_substitutions (
    team_id       bigint NOT NULL REFERENCES teams(team_id),
    slot          smallint NOT NULL CHECK (slot BETWEEN 1 AND 3),
    player_out_id bigint NOT NULL REFERENCES players(player_id) ON DELETE CASCADE,
    player_in_id  bigint NOT NULL REFERENCES players(player_id) ON DELETE CASCADE,
    minute        smallint NOT NULL CHECK (minute BETWEEN 1 AND 90),
    PRIMARY KEY (team_id, slot)
);
