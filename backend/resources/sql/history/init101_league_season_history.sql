-- League season history (see openspec/specs/league-page and engine/season.go):
--
-- * classification.seed is a league's starting order for the current season, 1..N: teams relegated
--   into the league first, then the teams that stayed, then the teams promoted into it, each group
--   in the order it finished last season (engine.ComputeSeasonSeeds). It is the last tie-break of
--   every standings ordering, so a new season's table reads in that order on matchday 0.
-- * season_history keeps each team's full final table line (matches, points, wins, draws, losses,
--   goals), so a past season's whole table can be shown again.
--
-- Re-runnable: the columns and index are IF NOT EXISTS, the counter backfill only fills rows whose
-- matches is still NULL, and the seed backfill only touches leagues where no row has a seed yet.

ALTER TABLE classification ADD COLUMN IF NOT EXISTS seed int;

ALTER TABLE season_history ADD COLUMN IF NOT EXISTS matches int;
ALTER TABLE season_history ADD COLUMN IF NOT EXISTS points int;
ALTER TABLE season_history ADD COLUMN IF NOT EXISTS wins int;
ALTER TABLE season_history ADD COLUMN IF NOT EXISTS draws int;
ALTER TABLE season_history ADD COLUMN IF NOT EXISTS losses int;
ALTER TABLE season_history ADD COLUMN IF NOT EXISTS goals_scored int;
ALTER TABLE season_history ADD COLUMN IF NOT EXISTS goals_received int;

CREATE INDEX IF NOT EXISTS season_history_idx2 ON season_history (league_id, season);

-- Counters of every past season, rebuilt from the team's played league matches in the league and
-- season the row records (the same aggregation common/standingsrepair.go uses).
UPDATE season_history sh SET
    matches = agg.matches, points = agg.points, wins = agg.wins, draws = agg.draws,
    losses = agg.losses, goals_scored = agg.goals_scored, goals_received = agg.goals_received
FROM (
    SELECT h.team_id, h.season, count(m.match_id) AS matches,
        COALESCE(SUM(CASE WHEN g.gf > g.ga THEN 3 WHEN g.gf = g.ga THEN 1 ELSE 0 END), 0) AS points,
        COALESCE(SUM(CASE WHEN g.gf > g.ga THEN 1 ELSE 0 END), 0) AS wins,
        COALESCE(SUM(CASE WHEN g.gf = g.ga THEN 1 ELSE 0 END), 0) AS draws,
        COALESCE(SUM(CASE WHEN g.gf < g.ga THEN 1 ELSE 0 END), 0) AS losses,
        COALESCE(SUM(g.gf), 0) AS goals_scored, COALESCE(SUM(g.ga), 0) AS goals_received
    FROM season_history h
    LEFT JOIN matches m ON m.league_id = h.league_id AND m.season = h.season AND m.played = true
        AND (m.home_team_id = h.team_id OR m.away_team_id = h.team_id)
    LEFT JOIN LATERAL (
        SELECT CASE WHEN m.home_team_id = h.team_id THEN m.home_goals ELSE m.away_goals END AS gf,
               CASE WHEN m.home_team_id = h.team_id THEN m.away_goals ELSE m.home_goals END AS ga
    ) g ON true
    WHERE h.matches IS NULL AND h.league_id IS NOT NULL
    GROUP BY h.team_id, h.season
) agg
WHERE sh.team_id = agg.team_id AND sh.season = agg.season;

-- Starting order of the season already under way, rebuilt from each team's previous-season
-- history: relegated into the league = 0, stayed = 1, promoted into it = 2, no history = 3; each
-- group by final position, then points, goal difference and goals scored.
WITH unseeded AS (
    SELECT league_id FROM classification GROUP BY league_id HAVING bool_and(seed IS NULL)
), ranked AS (
    SELECT c.league_id, c.team_id,
        ROW_NUMBER() OVER (
            PARTITION BY c.league_id
            ORDER BY CASE WHEN sh.team_id IS NULL THEN 3 WHEN sh.relegated THEN 0 WHEN sh.promoted THEN 2 ELSE 1 END,
                sh.final_position ASC NULLS LAST, sh.points DESC NULLS LAST,
                (sh.goals_scored - sh.goals_received) DESC NULLS LAST, sh.goals_scored DESC NULLS LAST,
                c.team_id
        ) AS seed
    FROM classification c
    JOIN unseeded u ON u.league_id = c.league_id
    LEFT JOIN season_history sh ON sh.team_id = c.team_id AND sh.season = c.season - 1 AND sh.league_id IS NOT NULL
)
UPDATE classification c SET seed = r.seed
FROM ranked r
WHERE c.league_id = r.league_id AND c.team_id = r.team_id;
