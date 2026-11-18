-- A manager's planned in-match substitutions for one specific league match's lineup override, and
-- the cup-tie equivalent - the same (match/tie_id, team_id, slot) shape
-- friendly_lineup_substitutions already established (init59_friendly_lineup_substitutions.sql),
-- now that a league match's or cup tie's starting XI can itself be overridden per match (see
-- lineup/cup_tie_lineup) rather than only through the team's standing default lineup. A
-- substitution only makes sense against the XI it's paired with (see
-- handlers.validateSubstitutions), so these are scoped to the match/tie rather than the team as a
-- whole, unlike default_lineup_substitutions.
--
-- slot is 1-3, assigned by save order, same convention as default_lineup_substitutions/
-- friendly_lineup_substitutions.
CREATE TABLE IF NOT EXISTS lineup_substitutions (
    match_id      bigint NOT NULL REFERENCES matches(match_id),
    team_id       bigint NOT NULL REFERENCES teams(team_id),
    slot          smallint NOT NULL CHECK (slot BETWEEN 1 AND 3),
    player_out_id bigint NOT NULL REFERENCES players(player_id) ON DELETE CASCADE,
    player_in_id  bigint NOT NULL REFERENCES players(player_id) ON DELETE CASCADE,
    minute        smallint NOT NULL CHECK (minute BETWEEN 1 AND 90),
    PRIMARY KEY (match_id, team_id, slot)
);

CREATE TABLE IF NOT EXISTS cup_tie_lineup_substitutions (
    tie_id        bigint NOT NULL REFERENCES cup_ties(tie_id),
    team_id       bigint NOT NULL REFERENCES teams(team_id),
    slot          smallint NOT NULL CHECK (slot BETWEEN 1 AND 3),
    player_out_id bigint NOT NULL REFERENCES players(player_id) ON DELETE CASCADE,
    player_in_id  bigint NOT NULL REFERENCES players(player_id) ON DELETE CASCADE,
    minute        smallint NOT NULL CHECK (minute BETWEEN 1 AND 90),
    PRIMARY KEY (tie_id, team_id, slot)
);
