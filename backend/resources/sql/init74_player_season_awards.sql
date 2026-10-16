-- Permanent record of each league's season-end honors (see engine.awardSeasonHonors): award 1 is
-- the Golden Boot (top scorer, value = goals), award 2 is Player of the Season (value = the
-- weighted goals/cards score, see common.GetPlayerOfTheSeason). Until now these were only ever
-- announced in a notification, so a player's profile could not say how many times they had won
-- one. One winner per (league, season, award).
CREATE TABLE IF NOT EXISTS player_season_awards (
    league_id bigint   NOT NULL,
    season    int      NOT NULL,
    award     smallint NOT NULL,
    player_id bigint   NOT NULL REFERENCES players(player_id) ON DELETE CASCADE,
    value     int      NOT NULL DEFAULT 0,
    PRIMARY KEY (league_id, season, award)
);
CREATE INDEX IF NOT EXISTS idx_player_season_awards_player_id ON player_season_awards(player_id);

-- Backfills the seasons already finished before this migration existed, from match_events (same
-- rules as common.GetTopScorersFinal / common.GetPlayerOfTheSeason: goal = 3, yellow = -1, red =
-- -3 for the player of the season, ties broken by lowest player id, only players still on a
-- team's roster). Only runs while the table is empty - once any award has been recorded (by this
-- backfill or by a real season rollover) the file is a no-op when reapplied at startup - and only
-- covers seasons before the current one, since the current season's awards are still to be won.
INSERT INTO player_season_awards (league_id, season, award, player_id, value)
SELECT league_id, season, award, player_id, value FROM (
    SELECT s.league_id, s.season, s.award, s.player_id, s.value,
           row_number() OVER (PARTITION BY s.league_id, s.season, s.award ORDER BY s.value DESC, s.player_id ASC) AS rn
    FROM (
        SELECT m.league_id, m.season,
               substring(me.team_players FROM '^\[(\d+)\]$')::bigint AS player_id,
               1 AS award,
               count(*)::int AS value
        FROM match_events me
        JOIN matches m ON m.match_id = me.match_id
        WHERE me.event_type = 0 AND me.team_players ~ '^\[\d+\]$'
          AND m.season < (SELECT max(current_season) FROM season)
        GROUP BY m.league_id, m.season, substring(me.team_players FROM '^\[(\d+)\]$')::bigint
        UNION ALL
        SELECT m.league_id, m.season,
               substring(me.team_players FROM '^\[(\d+)\]$')::bigint AS player_id,
               2 AS award,
               sum(CASE me.event_type WHEN 0 THEN 3 WHEN 2 THEN -1 WHEN 3 THEN -3 ELSE 0 END)::int AS value
        FROM match_events me
        JOIN matches m ON m.match_id = me.match_id
        WHERE me.event_type IN (0, 2, 3) AND me.team_players ~ '^\[\d+\]$'
          AND m.season < (SELECT max(current_season) FROM season)
        GROUP BY m.league_id, m.season, substring(me.team_players FROM '^\[(\d+)\]$')::bigint
    ) s
    JOIN players p ON p.player_id = s.player_id AND p.team_id IS NOT NULL
    WHERE s.value > 0
) ranked
WHERE rn = 1
  AND NOT EXISTS (SELECT 1 FROM player_season_awards)
ON CONFLICT DO NOTHING;
