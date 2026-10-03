-- Gives every player an individual, hidden lifetime ceiling for each of the seven core skills
-- (see common.rollSkillCap/PlayerSkillCaps and engine/progression.go's driftSkill) - until now
-- match-by-match skill growth clamped every player to the same universal [1, 99] range, so a
-- squad full of ordinary players could all eventually become 99-rated, which real player
-- development never allows. These columns are never read by any frontend-facing query (see
-- common.GetPlayerSkillCaps, the only reader) and are deliberately absent from the Player struct
-- that gets serialized to API responses - a manager sees a player's current skills, never their
-- potential.
--
-- DEFAULT 0 is a sentinel meaning "not yet backfilled" rather than a real ceiling (every real
-- rolled value is at least 6 - see the gap distribution below) - every INSERT INTO players from
-- here on (common.doCreatePlayers, common.InjectFreeAgents) always supplies real values
-- explicitly, so 0 only ever shows up on a row created before this migration ran.
ALTER TABLE players ADD COLUMN IF NOT EXISTS goal_keeper_max integer NOT NULL DEFAULT 0;
ALTER TABLE players ADD COLUMN IF NOT EXISTS defense_max integer NOT NULL DEFAULT 0;
ALTER TABLE players ADD COLUMN IF NOT EXISTS lateral_max integer NOT NULL DEFAULT 0;
ALTER TABLE players ADD COLUMN IF NOT EXISTS pass_max integer NOT NULL DEFAULT 0;
ALTER TABLE players ADD COLUMN IF NOT EXISTS midfield_max integer NOT NULL DEFAULT 0;
ALTER TABLE players ADD COLUMN IF NOT EXISTS dribbling_max integer NOT NULL DEFAULT 0;
ALTER TABLE players ADD COLUMN IF NOT EXISTS shot_max integer NOT NULL DEFAULT 0;

-- Backfills every pre-existing player (the 0 sentinel above) with the same weighted gap
-- distribution rollSkillCap uses for a freshly created one, based on their *current* skill value
-- rather than a now-unrecoverable creation-time value: 70% of rolls add a modest 5-15 points of
-- headroom, 25% add 15-30, and a rare 5% add a wonderkid-sized 30-45 - each independently rolled
-- per skill, every result clamped to the universal 99 ceiling. Idempotent: once a row's
-- goal_keeper_max is no longer 0 (every real rolled value is >= 6), this WHERE clause excludes it
-- on the next time this migration file is reapplied (see common.applyPendingMigrations), so a
-- player is only ever backfilled once.
CREATE OR REPLACE FUNCTION pg_temp.random_skill_gap() RETURNS integer AS $$
DECLARE
    r double precision := random();
BEGIN
    IF r < 0.70 THEN
        RETURN 5 + floor(random() * 11)::int; -- 5..15
    ELSIF r < 0.95 THEN
        RETURN 15 + floor(random() * 16)::int; -- 15..30
    ELSE
        RETURN 30 + floor(random() * 16)::int; -- 30..45
    END IF;
END;
$$ LANGUAGE plpgsql VOLATILE;

UPDATE players SET
    goal_keeper_max = LEAST(99, goal_keeper + pg_temp.random_skill_gap()),
    defense_max = LEAST(99, defense + pg_temp.random_skill_gap()),
    lateral_max = LEAST(99, "lateral" + pg_temp.random_skill_gap()),
    pass_max = LEAST(99, pass + pg_temp.random_skill_gap()),
    midfield_max = LEAST(99, midfield + pg_temp.random_skill_gap()),
    dribbling_max = LEAST(99, dribbling + pg_temp.random_skill_gap()),
    shot_max = LEAST(99, shot + pg_temp.random_skill_gap())
WHERE goal_keeper_max = 0;
