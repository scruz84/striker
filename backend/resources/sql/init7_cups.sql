-- A single-elimination cup competition per country per season, entrants drawn from the top
-- division (tier 1). Deliberately NOT built on the matches/match_events tables - those are
-- deeply wired to league classification (see engine.UpdateClassification) and this schema
-- change can't be verified against a live database in this environment, so a cup tie gets its
-- own tables instead of risking the core league match pipeline. See engine/cups.go.
--
-- Runs automatically after init6_player_age.sql on a fresh database. On an already-initialized
-- one, apply it manually once - see backend/README.md.

CREATE TABLE IF NOT EXISTS cups (
    cup_id           bigserial PRIMARY KEY,
    country          varchar(4) NOT NULL REFERENCES countries(name),
    season           int NOT NULL,
    name             varchar(200) NOT NULL,
    completed        boolean NOT NULL DEFAULT false,
    champion_team_id bigint REFERENCES teams(team_id)
);

CREATE TABLE IF NOT EXISTS cup_ties (
    tie_id          bigserial PRIMARY KEY,
    cup_id          bigint NOT NULL REFERENCES cups(cup_id),
    round           int NOT NULL,
    slot_number     int NOT NULL,
    home_team_id    bigint REFERENCES teams(team_id),
    away_team_id    bigint REFERENCES teams(team_id),
    match_date      timestamptz,
    played          boolean NOT NULL DEFAULT false,
    in_simulation   boolean NOT NULL DEFAULT false,
    home_goals      int NOT NULL DEFAULT 0,
    away_goals      int NOT NULL DEFAULT 0,
    went_to_penalties boolean NOT NULL DEFAULT false,
    winner_team_id  bigint REFERENCES teams(team_id),
    UNIQUE (cup_id, round, slot_number)
);

CREATE TABLE IF NOT EXISTS cup_tie_events (
    event_id            bigserial PRIMARY KEY,
    tie_id              bigint NOT NULL REFERENCES cup_ties(tie_id),
    event_type          int,
    team_owner          bigint REFERENCES teams(team_id),
    other_team          bigint REFERENCES teams(team_id),
    minute              int,
    team_players        varchar(200),
    team_player_names   varchar(1000),
    home_goals          int DEFAULT 0,
    away_goals          int DEFAULT 0
);

CREATE INDEX IF NOT EXISTS cup_ties_idx1 ON cup_ties(cup_id, round);
CREATE INDEX IF NOT EXISTS cup_tie_events_idx1 ON cup_tie_events(tie_id);
