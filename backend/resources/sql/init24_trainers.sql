-- A team's head coach - hired from an open market of generated candidates (see
-- common.GenerateTrainerCandidates), at most one per team. A hired trainer's skill accelerates
-- player attribute growth and raises the youth academy's prospect ceiling (see
-- engine.ProgressPlayers) and costs a wage deducted alongside player wages after each match (see
-- engine.deductTrainerWage). team_id NULL means the trainer is on the open market, unemployed -
-- the same convention players.team_id uses for free agents.
CREATE TABLE IF NOT EXISTS trainers (
    trainer_id bigserial PRIMARY KEY,
    first_name varchar(100) NOT NULL,
    last_name varchar(100) NOT NULL,
    team_id bigint REFERENCES teams(team_id) ON DELETE SET NULL,
    skill int NOT NULL,
    wage int NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS trainers_team_unique_idx ON trainers(team_id) WHERE team_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS trainers_available_idx ON trainers(skill) WHERE team_id IS NULL;
