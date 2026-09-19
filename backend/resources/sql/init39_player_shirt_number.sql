-- Squad numbers (issue #42): fixed for now (not user-editable), assigned by the application as
-- the lowest number not already in use on a player's team (see common.nextFreeShirtNumber).
ALTER TABLE players ADD COLUMN IF NOT EXISTS shirt_number INTEGER;

-- Guards against two players ever ending up with the same number on the same team - belt and
-- braces alongside the application-level lowest-free-number assignment, which isn't itself
-- immune to a race between two concurrent squad changes for the same team.
CREATE UNIQUE INDEX IF NOT EXISTS players_team_shirt_number_idx ON players (team_id, shirt_number)
    WHERE team_id IS NOT NULL AND shirt_number IS NOT NULL;

-- Backfills every senior-squad player still missing a shirt number (a pre-existing player from
-- before this column existed, or one whose earlier assignment failed) with the lowest number not
-- already taken on their own team, lowest player_id first - the same "lowest free number" rule
-- common.nextFreeShirtNumbers applies at runtime, computed here in SQL. This must find genuinely
-- free numbers rather than just a positional rank (this file's original approach): every numbered
-- migration is re-applied on every startup (see common.applyPendingMigrations), and a team's
-- shirt numbers are rarely a contiguous 1..N block once a player has ever left (freeing up a low
-- number) - a plain ROW_NUMBER() OVER (PARTITION BY team_id ORDER BY player_id) ignores which
-- numbers are already held and can reassign one a teammate still has, tripping the unique index
-- above (issue #73). Academy prospects are left unnumbered until actually promoted, same as a
-- freshly created one (see common.PromoteFromAcademy).
WITH needs_number AS (
    SELECT player_id, team_id,
           ROW_NUMBER() OVER (PARTITION BY team_id ORDER BY player_id) AS need_rank
    FROM players
    WHERE team_id IS NOT NULL AND in_academy = false AND shirt_number IS NULL
),
free_numbers AS (
    SELECT teams_needing.team_id, n,
           ROW_NUMBER() OVER (PARTITION BY teams_needing.team_id ORDER BY n) AS free_rank
    FROM (SELECT DISTINCT team_id FROM needs_number) teams_needing
    CROSS JOIN generate_series(1, 1000) AS n
    WHERE NOT EXISTS (
        SELECT 1 FROM players p2 WHERE p2.team_id = teams_needing.team_id AND p2.shirt_number = n
    )
)
UPDATE players p
SET shirt_number = free_numbers.n
FROM needs_number
JOIN free_numbers ON free_numbers.team_id = needs_number.team_id AND free_numbers.free_rank = needs_number.need_rank
WHERE p.player_id = needs_number.player_id;
