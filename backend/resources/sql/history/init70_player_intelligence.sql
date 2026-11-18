-- Gives every player a hidden "intelligence" tier (see common.IntelligenceNaive/Casual/Aware/
-- Shrewd/Mastermind and common.IntelligenceReactionFactor) that decides how sharply they react, in
-- contract negotiations, to their own recent skill trend (see common.PlayerSkillTrend, derived from
-- player_progression_log): a player who's been improving wants more money to re-sign, one who's
-- been declining is more willing to trade a cheaper wage for a longer, more secure deal (see
-- common.ExpectedWageWithIntelligence/MaxAcceptableRenewalYearsWithIntelligence). Never read by any
-- frontend-facing query and deliberately absent from the Player struct that gets serialized to API
-- responses, same treatment as goal_keeper_max etc. (see init64_player_skill_max.sql) - a manager
-- never sees how savvy a player's own agent is, only the resulting demands.
--
-- DEFAULT 0 is a sentinel meaning "not yet assigned" (every real tier is 1-5) - every INSERT INTO
-- players from here on (common.doCreatePlayers, common.InjectFreeAgents) always supplies a real
-- tier explicitly via common.RandomIntelligence, so 0 only ever shows up on a row created before
-- this migration ran.
ALTER TABLE players ADD COLUMN IF NOT EXISTS intelligence smallint NOT NULL DEFAULT 0;

-- Backfills every pre-existing player (the 0 sentinel above) with the same weighted distribution
-- RandomIntelligence uses for a freshly created one: 1=Naive..5=Mastermind, lower tiers more common
-- than the rare, fully market-savvy few. Idempotent: once a row's intelligence is no longer 0, this
-- WHERE clause excludes it the next time this migration file is reapplied (see
-- common.applyPendingMigrations), so a player is only ever backfilled once.
CREATE OR REPLACE FUNCTION pg_temp.random_intelligence_tier() RETURNS smallint AS $$
DECLARE
    r double precision := random() * 100;
BEGIN
    IF r < 30 THEN
        RETURN 1; -- Naive
    ELSIF r < 55 THEN
        RETURN 2; -- Casual
    ELSIF r < 75 THEN
        RETURN 3; -- Aware
    ELSIF r < 90 THEN
        RETURN 4; -- Shrewd
    ELSE
        RETURN 5; -- Mastermind
    END IF;
END;
$$ LANGUAGE plpgsql VOLATILE;

UPDATE players SET intelligence = pg_temp.random_intelligence_tier() WHERE intelligence = 0;
