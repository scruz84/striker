-- Player contracts + free agency. A player's contract counts down by one every season (see
-- engine.ProgressPlayers / common.AdvancePlayerContract); when it hits zero they're released to
-- free agency (team_id set to NULL) rather than staying with their club forever. Any player with
-- no team - whether released this way or orphaned earlier by common.AssignTeam reassigning a CPU
-- squad - is signable by any team via common.SignFreeAgent. Defaulting existing rows to 0 is
-- exactly right for team_id IS NULL rows already sitting in the table: they're already free
-- agents, out of contract.
--
-- Runs automatically after init20_team_rename.sql on a fresh database. On an already-initialized
-- one, apply it manually once - see backend/README.md.

ALTER TABLE players ADD COLUMN IF NOT EXISTS contract_years_remaining int NOT NULL DEFAULT 0;

-- The 0 default is correct for a player who's already unowned (team_id IS NULL) - they're
-- already out of contract. An existing player who DOES have a team predates this feature
-- entirely and would otherwise show as contract-expired too, releasing their whole squad to
-- free agency the very next season rollover - so backfill those with a random initial length,
-- the same [2, 4] range doCreatePlayers uses for a freshly created player (see
-- common.RandomContractYears). Newly inserted rows going forward always set this column
-- explicitly and never rely on the default.
UPDATE players SET contract_years_remaining = 2 + floor(random() * 3)::int WHERE team_id IS NOT NULL;
