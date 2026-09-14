-- Squad numbers (issue #42): fixed for now (not user-editable), assigned by the application as
-- the lowest number not already in use on a player's team (see common.nextFreeShirtNumber).
ALTER TABLE players ADD COLUMN IF NOT EXISTS shirt_number INTEGER;

-- Guards against two players ever ending up with the same number on the same team - belt and
-- braces alongside the application-level lowest-free-number assignment, which isn't itself
-- immune to a race between two concurrent squad changes for the same team.
CREATE UNIQUE INDEX IF NOT EXISTS players_team_shirt_number_idx ON players (team_id, shirt_number)
    WHERE team_id IS NOT NULL AND shirt_number IS NOT NULL;

-- Backfills every pre-existing senior-squad player (this column didn't exist for them yet) with
-- sequential numbers per team, lowest player_id first - as good a default ordering as any for
-- players who already existed before numbers were a concept. Academy prospects are left
-- unnumbered until actually promoted, same as a freshly created one (see
-- common.PromoteFromAcademy).
UPDATE players p
SET shirt_number = sub.rn
FROM (
    SELECT player_id, ROW_NUMBER() OVER (PARTITION BY team_id ORDER BY player_id) AS rn
    FROM players
    WHERE team_id IS NOT NULL AND in_academy = false
) sub
WHERE p.player_id = sub.player_id AND p.shirt_number IS NULL;
