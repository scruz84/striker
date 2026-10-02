-- Adds the two new per-player attributes described in GAME_MECHANICS.md's "Player skill
-- progression" section: stamina (how well a player holds up physically within a single match,
-- read every simulated minute by engine.StaminaSimulation) and form (a slower-moving measure of
-- recent performances, read once per match by common.ComputeMatchRatings). Both default to 50 -
-- the same mid-point neutral value doCreatePlayers/InjectFreeAgents now seed a brand-new player
-- with for form (stamina is instead randomized like the seven core skills - see those callers),
-- and the value every already-existing player backfills to here, so no existing squad suddenly
-- starts the next match either exhausted or red-hot.
ALTER TABLE players ADD COLUMN IF NOT EXISTS stamina integer NOT NULL DEFAULT 50;
ALTER TABLE players ADD COLUMN IF NOT EXISTS form integer NOT NULL DEFAULT 50;
