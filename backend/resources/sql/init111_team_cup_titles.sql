-- Cup titles in a team's history (openspec/changes/add-cup-titles-to-team-history, spec
-- team-history).
--
-- * season_history.international_cup_id is the regional cup edition (European Cup, Copa America,
--   Asian Cup, African Cup) the team won during that league season. engine.advanceCup records it
--   in the season the champion's country was playing when the final was resolved.
-- * Before this, advanceCup wrote cup_result = 'Champion' under cups.season for every cup, and for
--   a regional cup or the World Cup knockout that is an edition / cycle number, not a league
--   season. Every cup_result no domestic cup backs is cleared, and a row that only existed for it
--   is deleted.
-- * Every completed regional cup is recorded in its champion's history, in the season of the
--   champion's last league match on or before the final (or its first one after, if none before).
--
-- A team that won two editions in one season keeps the newest one.
--
-- Re-runnable: the column is IF NOT EXISTS, the clean-up only touches rows no domestic cup backs,
-- and the backfill only fills international_cup_id where it is still NULL.

ALTER TABLE season_history ADD COLUMN IF NOT EXISTS international_cup_id bigint REFERENCES cups(cup_id) ON DELETE SET NULL;

UPDATE season_history sh SET cup_result = NULL, updated_at = now()
WHERE sh.cup_result IS NOT NULL
  AND NOT EXISTS (
      SELECT 1 FROM cups c
      WHERE c.country IS NOT NULL AND c.kind = 'cup' AND c.season = sh.season AND c.champion_team_id = sh.team_id
  );

WITH finals AS (
    SELECT c.cup_id, c.champion_team_id AS team_id, max(ct.match_date) AS final_date
    FROM cups c
    JOIN cup_ties ct ON ct.cup_id = c.cup_id
    WHERE c.region IS NOT NULL AND c.region <> 'world' AND c.completed AND c.champion_team_id IS NOT NULL
    GROUP BY c.cup_id, c.champion_team_id
), placed AS (
    SELECT f.cup_id, f.team_id, COALESCE(
        (SELECT m.season FROM matches m
         WHERE (m.home_team_id = f.team_id OR m.away_team_id = f.team_id) AND m.match_date <= f.final_date
         ORDER BY m.match_date DESC LIMIT 1),
        (SELECT m.season FROM matches m
         WHERE (m.home_team_id = f.team_id OR m.away_team_id = f.team_id) AND m.match_date > f.final_date
         ORDER BY m.match_date ASC LIMIT 1)
    ) AS season
    FROM finals f
    WHERE f.final_date IS NOT NULL
)
INSERT INTO season_history (team_id, season, international_cup_id)
SELECT DISTINCT ON (team_id, season) team_id, season, cup_id FROM placed WHERE season IS NOT NULL
ORDER BY team_id, season, cup_id DESC
ON CONFLICT (team_id, season) DO UPDATE SET international_cup_id = EXCLUDED.international_cup_id, updated_at = now()
WHERE season_history.international_cup_id IS NULL;

DELETE FROM season_history
WHERE league_id IS NULL AND final_position IS NULL AND matches IS NULL
  AND cup_result IS NULL AND international_cup_id IS NULL
  AND NOT promoted AND NOT relegated;
