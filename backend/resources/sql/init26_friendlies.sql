-- Friendly matches: a team can challenge any other team to a non-competitive match that the
-- other side must accept before it's scheduled - no effect on league classification or cup
-- progression, no ticket revenue/sponsorship, no AI-team upkeep hook (see engine/friendlies.go).
-- Deliberately NOT built on the matches/match_events tables, same reasoning as cup_ties (see
-- init7_cups.sql): those are deeply wired to league classification, so a different kind of
-- match gets its own tables rather than risking the core league match pipeline.
--
-- max_friendlies_per_week bounds how many friendly matches (accepted, not just requested) a
-- single team may be part of per real calendar week (see common.StartOfWeek) - admin-configurable
-- since how much this should be allowed to happen is a game-balance call, not a fixed rule.
ALTER TABLE season ADD COLUMN IF NOT EXISTS max_friendlies_per_week smallint NOT NULL DEFAULT 1;

-- A pending invite - deleted the moment it's accepted (a friendly_matches row is created instead)
-- or rejected/withdrawn. The unique constraint stops the same team spamming repeat challenges at
-- the same opponent while one is still pending.
CREATE TABLE IF NOT EXISTS friendly_challenges (
    challenge_id        bigserial PRIMARY KEY,
    challenger_team_id  bigint NOT NULL REFERENCES teams(team_id),
    opponent_team_id    bigint NOT NULL REFERENCES teams(team_id),
    created_at          timestamptz NOT NULL DEFAULT now(),
    UNIQUE (challenger_team_id, opponent_team_id)
);

-- The actual fixture, created only once both sides have agreed to play - challenger_team_id
-- from the challenge becomes home_team_id here.
CREATE TABLE IF NOT EXISTS friendly_matches (
    match_id       bigserial PRIMARY KEY,
    home_team_id   bigint NOT NULL REFERENCES teams(team_id),
    away_team_id   bigint NOT NULL REFERENCES teams(team_id),
    match_date     timestamptz NOT NULL,
    played         boolean NOT NULL DEFAULT false,
    in_simulation  boolean NOT NULL DEFAULT false,
    home_goals     int NOT NULL DEFAULT 0,
    away_goals     int NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS friendly_events (
    event_id            bigserial PRIMARY KEY,
    match_id            bigint NOT NULL REFERENCES friendly_matches(match_id),
    event_type          int,
    team_owner          bigint REFERENCES teams(team_id),
    other_team          bigint REFERENCES teams(team_id),
    minute              int,
    team_players        varchar(200),
    team_player_names   varchar(1000),
    home_goals          int DEFAULT 0,
    away_goals          int DEFAULT 0
);

CREATE INDEX IF NOT EXISTS friendly_challenges_opponent_idx ON friendly_challenges(opponent_team_id);
CREATE INDEX IF NOT EXISTS friendly_matches_home_idx ON friendly_matches(home_team_id, match_date);
CREATE INDEX IF NOT EXISTS friendly_matches_away_idx ON friendly_matches(away_team_id, match_date);
CREATE INDEX IF NOT EXISTS friendly_events_idx ON friendly_events(match_id);
