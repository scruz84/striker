-- World Cup: a group stage (round-robin, see common.BuildRoundRobinRounds) followed by a
-- knockout stage that reuses the existing single-elimination cup engine outright - see
-- common.CreateInternationalCup/engine.startWorldCupKnockout, which creates a regular `cups` row
-- with region = 'world' and world_cups.knockout_cup_id pointing at it. Entrants are national
-- teams (teams.is_national_team), never real clubs - see common.StartWorldCupForNewSeason.
CREATE TABLE IF NOT EXISTS world_cups (
    world_cup_id     bigserial PRIMARY KEY,
    season           int NOT NULL,
    name             varchar(200) NOT NULL,
    -- 'groups' while group matches are still being played, 'knockout' once every group's fixtures
    -- are done and the bracket has been seeded (see engine.AdvanceWorldCups).
    stage            varchar(20) NOT NULL DEFAULT 'groups',
    completed        boolean NOT NULL DEFAULT false,
    champion_team_id bigint REFERENCES teams(team_id),
    -- Set only once the group stage finishes and the knockout bracket is created - see
    -- engine.startWorldCupKnockout. Null while stage = 'groups'.
    knockout_cup_id  bigint REFERENCES cups(cup_id)
);

CREATE TABLE IF NOT EXISTS world_cup_groups (
    group_id     bigserial PRIMARY KEY,
    world_cup_id bigint NOT NULL REFERENCES world_cups(world_cup_id),
    group_name   varchar(10) NOT NULL
);

-- One row per team per group, kept as running totals updated after each of that team's group
-- matches is simulated (see common.UpdateWorldCupStandingsAfterMatch) rather than recomputed from
-- scratch each read - the same running-totals approach league classification already uses (see
-- common.ResetClassificationForSeason).
CREATE TABLE IF NOT EXISTS world_cup_group_teams (
    group_id      bigint NOT NULL REFERENCES world_cup_groups(group_id),
    team_id       bigint NOT NULL REFERENCES teams(team_id),
    played        int NOT NULL DEFAULT 0,
    won           int NOT NULL DEFAULT 0,
    drawn         int NOT NULL DEFAULT 0,
    lost          int NOT NULL DEFAULT 0,
    goals_for     int NOT NULL DEFAULT 0,
    goals_against int NOT NULL DEFAULT 0,
    PRIMARY KEY (group_id, team_id)
);

-- A group's fixture is a real friendly_matches row (so it's simulated by the ordinary
-- FriendlyJob, and /friendlies/:id renders it with no changes needed) - this table only records
-- which group/round it belongs to, so standings can be updated and the group stage's completion
-- detected (see common.UpdateWorldCupStandingsAfterMatch/IsWorldCupGroupStageComplete).
CREATE TABLE IF NOT EXISTS world_cup_group_matches (
    group_id bigint NOT NULL REFERENCES world_cup_groups(group_id),
    round    int NOT NULL,
    match_id bigint NOT NULL REFERENCES friendly_matches(match_id),
    PRIMARY KEY (match_id)
);
