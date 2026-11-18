-- Conditional match orders (see openspec/specs/match-simulation "Conditional match orders" and
-- common.LineupOrder): up to 3 per lineup, each armed from from_minute and fired once at the first
-- minute whose starting score meets condition (ORDER_CONDITION_*), with one action
-- (ORDER_ACTION_*): set the mentality, or substitute player_out_id for player_in_id. One table per
-- lineup kind, like the scheduled substitutions; a deleted player takes his orders with him, as
-- his scheduled substitutions do. Re-runnable.
CREATE TABLE IF NOT EXISTS default_lineup_orders (
    team_id       bigint NOT NULL REFERENCES teams(team_id),
    slot          smallint NOT NULL CHECK (slot BETWEEN 1 AND 3),
    from_minute   smallint NOT NULL CHECK (from_minute BETWEEN 1 AND 89),
    condition     smallint NOT NULL CHECK (condition BETWEEN 0 AND 4),
    action        smallint NOT NULL CHECK (action BETWEEN 0 AND 1),
    mentality     smallint NULL CHECK (mentality BETWEEN -1 AND 1),
    player_out_id bigint NULL REFERENCES players(player_id) ON DELETE CASCADE,
    player_in_id  bigint NULL REFERENCES players(player_id) ON DELETE CASCADE,
    PRIMARY KEY (team_id, slot)
);

CREATE TABLE IF NOT EXISTS lineup_orders (
    match_id      bigint NOT NULL REFERENCES matches(match_id),
    team_id       bigint NOT NULL REFERENCES teams(team_id),
    slot          smallint NOT NULL CHECK (slot BETWEEN 1 AND 3),
    from_minute   smallint NOT NULL CHECK (from_minute BETWEEN 1 AND 89),
    condition     smallint NOT NULL CHECK (condition BETWEEN 0 AND 4),
    action        smallint NOT NULL CHECK (action BETWEEN 0 AND 1),
    mentality     smallint NULL CHECK (mentality BETWEEN -1 AND 1),
    player_out_id bigint NULL REFERENCES players(player_id) ON DELETE CASCADE,
    player_in_id  bigint NULL REFERENCES players(player_id) ON DELETE CASCADE,
    PRIMARY KEY (match_id, team_id, slot)
);

CREATE TABLE IF NOT EXISTS cup_tie_lineup_orders (
    tie_id        bigint NOT NULL REFERENCES cup_ties(tie_id),
    team_id       bigint NOT NULL REFERENCES teams(team_id),
    slot          smallint NOT NULL CHECK (slot BETWEEN 1 AND 3),
    from_minute   smallint NOT NULL CHECK (from_minute BETWEEN 1 AND 89),
    condition     smallint NOT NULL CHECK (condition BETWEEN 0 AND 4),
    action        smallint NOT NULL CHECK (action BETWEEN 0 AND 1),
    mentality     smallint NULL CHECK (mentality BETWEEN -1 AND 1),
    player_out_id bigint NULL REFERENCES players(player_id) ON DELETE CASCADE,
    player_in_id  bigint NULL REFERENCES players(player_id) ON DELETE CASCADE,
    PRIMARY KEY (tie_id, team_id, slot)
);

CREATE TABLE IF NOT EXISTS friendly_lineup_orders (
    match_id      bigint NOT NULL REFERENCES friendly_matches(match_id),
    team_id       bigint NOT NULL REFERENCES teams(team_id),
    slot          smallint NOT NULL CHECK (slot BETWEEN 1 AND 3),
    from_minute   smallint NOT NULL CHECK (from_minute BETWEEN 1 AND 89),
    condition     smallint NOT NULL CHECK (condition BETWEEN 0 AND 4),
    action        smallint NOT NULL CHECK (action BETWEEN 0 AND 1),
    mentality     smallint NULL CHECK (mentality BETWEEN -1 AND 1),
    player_out_id bigint NULL REFERENCES players(player_id) ON DELETE CASCADE,
    player_in_id  bigint NULL REFERENCES players(player_id) ON DELETE CASCADE,
    PRIMARY KEY (match_id, team_id, slot)
);

-- The new mentality of a MENTALITY_CHANGE event (common.Event.Mentality); NULL for every other event.
ALTER TABLE match_events ADD COLUMN IF NOT EXISTS mentality smallint NULL;
ALTER TABLE cup_tie_events ADD COLUMN IF NOT EXISTS mentality smallint NULL;
ALTER TABLE friendly_events ADD COLUMN IF NOT EXISTS mentality smallint NULL;
