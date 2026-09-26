-- A manager's planned in-match substitutions for one specific friendly's lineup override - the
-- friendly equivalent of default_lineup_substitutions (init28_lineup_substitutions.sql), kept
-- entirely separate for the same reason friendly_lineup is separate from lineup/lineup_positions
-- (see init34_friendly_lineup.sql): friendly_matches has its own id sequence, so writing here
-- against `matches(match_id)` could collide with an unrelated competitive match sharing the same
-- id. Scoped to (match_id, team_id) rather than team_id alone (unlike default_lineup_substitutions)
-- since a friendly's starting XI can itself differ from the team's standing default lineup (see
-- common.GetLineupsForFriendly) - a substitution only makes sense against the XI it's paired with
-- (see handlers.validateSubstitutions, common.SetFriendlyLineup).
--
-- slot is 1-3, assigned by save order, same convention as default_lineup_substitutions.
CREATE TABLE IF NOT EXISTS friendly_lineup_substitutions (
    match_id      bigint NOT NULL REFERENCES friendly_matches(match_id),
    team_id       bigint NOT NULL REFERENCES teams(team_id),
    slot          smallint NOT NULL CHECK (slot BETWEEN 1 AND 3),
    player_out_id bigint NOT NULL REFERENCES players(player_id) ON DELETE CASCADE,
    player_in_id  bigint NOT NULL REFERENCES players(player_id) ON DELETE CASCADE,
    minute        smallint NOT NULL CHECK (minute BETWEEN 1 AND 90),
    PRIMARY KEY (match_id, team_id, slot)
);
