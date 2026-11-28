-- Player match stats (openspec/specs/player-stats): one row per player who featured in a played
-- league match, cup tie or friendly, written in the same transaction as the result
-- (common.replacePlayerMatchStatsTx), with what the history pages total: minutes, start or sub,
-- rating, goals (penalties apart) and cards, plus the fixture's kickoff, season and competition so
-- a total needs no join to the three fixture tables. Readers only count rows whose kickoff is at
-- or before common.LiveBroadcastCutoff(), so a stat never shows before its match's replay ends.
--
-- kind/match_id follow match_player_ratings (a common.SimulationKind* value and that kind's own
-- id - three independent sequences, so no foreign key). player_id has no foreign key either: a
-- retired player's rows stay (openspec/specs/retired-players).
--
-- competition is common.StatsCompetition*: 1 league, 2 domestic cup, 3 promotion playoff,
-- 4 regional club cup, 5 World Cup, 6 club friendly, 7 national-team friendly. competition_id is
-- the league, the cup or the World Cup, NULL for a friendly. season is the league's or the
-- domestic cup's own season, and for every other competition the season the player's team's
-- country was playing at kickoff.
--
-- appeared is false only for rows the backfill below made from the goal and card events of a
-- match simulated before stored ratings (init89): they count goals and cards, not appearances.
--
-- Re-runnable: CREATE ... IF NOT EXISTS, and the backfill runs once, marked in schema_backfills
-- in the same implicit transaction as the whole file.

CREATE TABLE IF NOT EXISTS player_match_stats (
    kind           text      NOT NULL,
    match_id       bigint    NOT NULL,
    player_id      bigint    NOT NULL,
    team_id        bigint    NOT NULL,
    kickoff        timestamptz NOT NULL,
    season         int       NOT NULL,
    competition    smallint  NOT NULL,
    competition_id bigint,
    appeared       boolean   NOT NULL DEFAULT true,
    started        boolean   NOT NULL DEFAULT false,
    minutes        integer   NOT NULL DEFAULT 0,
    rated          boolean   NOT NULL DEFAULT false,
    rating         real      NOT NULL DEFAULT 0,
    goals          smallint  NOT NULL DEFAULT 0,
    penalty_goals  smallint  NOT NULL DEFAULT 0,
    yellow_cards   smallint  NOT NULL DEFAULT 0,
    red_cards      smallint  NOT NULL DEFAULT 0,
    -- The tactical position he spent the most minutes in (match_player_ratings.position, a
    -- common position constant); NULL without an appearance.
    position       smallint,
    PRIMARY KEY (kind, match_id, player_id)
);

CREATE INDEX IF NOT EXISTS player_match_stats_player_idx ON player_match_stats (player_id, season);
CREATE INDEX IF NOT EXISTS player_match_stats_competition_idx ON player_match_stats (competition, competition_id, season);
CREATE INDEX IF NOT EXISTS player_match_stats_team_idx ON player_match_stats (team_id, season);

-- One row per one-off data backfill that must not run again on the next startup.
CREATE TABLE IF NOT EXISTS schema_backfills (
    name       text PRIMARY KEY,
    applied_at timestamptz NOT NULL DEFAULT now()
);

-- Backfill from what every played fixture already stores: its ratings rows (who featured, for how
-- long, how well) and its goal/card events (team_players holds the one player concerned, as JSON).
WITH done AS (SELECT 1 FROM schema_backfills WHERE name = 'player_match_stats'),
season_starts AS (
    SELECT l.country, m.season, min(m.match_date) AS starts_at
    FROM matches m JOIN leagues l ON l.league_id = m.league_id
    GROUP BY l.country, m.season
),
fixtures AS (
    SELECT 'league'::text AS kind, m.match_id AS id, m.match_date AS kickoff, m.season AS season,
           1::smallint AS competition, m.league_id AS competition_id
    FROM matches m WHERE m.played
    UNION ALL
    SELECT 'cup', ct.tie_id, COALESCE(ct.match_date, now()),
           CASE WHEN c.country IS NOT NULL THEN c.season END,
           (CASE WHEN c.kind = 'playoff' THEN 3 WHEN c.country IS NOT NULL THEN 2 WHEN c.region = 'world' THEN 5 ELSE 4 END)::smallint,
           CASE WHEN c.region = 'world' THEN wc.world_cup_id ELSE c.cup_id END
    FROM cup_ties ct JOIN cups c ON c.cup_id = ct.cup_id
    LEFT JOIN world_cups wc ON wc.knockout_cup_id = c.cup_id
    WHERE ct.played
    UNION ALL
    SELECT 'friendly', fm.match_id, fm.match_date, NULL::int,
           (CASE WHEN g.world_cup_id IS NOT NULL THEN 5 WHEN h.is_national_team THEN 7 ELSE 6 END)::smallint,
           g.world_cup_id
    FROM friendly_matches fm JOIN teams h ON h.team_id = fm.home_team_id
    LEFT JOIN world_cup_group_matches wgm ON wgm.match_id = fm.match_id
    LEFT JOIN world_cup_groups g ON g.group_id = wgm.group_id
    WHERE fm.played
),
events AS (
    SELECT 'league'::text AS kind, match_id AS id, event_type, set_piece, team_owner, team_players FROM match_events
    WHERE event_type IN (0, 2, 3)
    UNION ALL
    SELECT 'cup', tie_id, event_type, set_piece, team_owner, team_players FROM cup_tie_events WHERE event_type IN (0, 2, 3)
    UNION ALL
    SELECT 'friendly', match_id, event_type, set_piece, team_owner, team_players FROM friendly_events WHERE event_type IN (0, 2, 3)
),
event_tally AS (
    SELECT kind, id, (team_players::jsonb ->> 0)::bigint AS player_id, min(team_owner) AS team_id,
           count(*) FILTER (WHERE event_type = 0)::smallint AS goals,
           count(*) FILTER (WHERE event_type = 0 AND set_piece = 1)::smallint AS penalty_goals,
           count(*) FILTER (WHERE event_type = 2)::smallint AS yellow_cards,
           count(*) FILTER (WHERE event_type = 3)::smallint AS red_cards
    FROM events
    WHERE team_players LIKE '[%' AND team_players::jsonb ->> 0 IS NOT NULL
    GROUP BY kind, id, (team_players::jsonb ->> 0)::bigint
),
appearances AS (
    SELECT r.kind, r.match_id AS id, r.player_id, r.team_id, true AS appeared, r.came_on IS NULL AS started,
           r.minutes, r.rated, r.rating, r.position::smallint AS position,
           COALESCE(e.goals, 0)::smallint AS goals, COALESCE(e.penalty_goals, 0)::smallint AS penalty_goals,
           COALESCE(e.yellow_cards, 0)::smallint AS yellow_cards, COALESCE(e.red_cards, 0)::smallint AS red_cards
    FROM match_player_ratings r
    LEFT JOIN event_tally e ON e.kind = r.kind AND e.id = r.match_id AND e.player_id = r.player_id
    UNION ALL
    SELECT e.kind, e.id, e.player_id, e.team_id, false, false, 0, false, 0, NULL::smallint,
           e.goals, e.penalty_goals, e.yellow_cards, e.red_cards
    FROM event_tally e
    WHERE e.team_id IS NOT NULL AND NOT EXISTS (
        SELECT 1 FROM match_player_ratings r WHERE r.kind = e.kind AND r.match_id = e.id AND r.player_id = e.player_id)
)
INSERT INTO player_match_stats (kind, match_id, player_id, team_id, kickoff, season, competition, competition_id,
                                appeared, started, minutes, rated, rating, position, goals, penalty_goals, yellow_cards, red_cards)
SELECT a.kind, a.id, a.player_id, a.team_id, f.kickoff,
       COALESCE(f.season,
                (SELECT max(s.season) FROM season_starts s WHERE s.country = t.country AND s.starts_at <= f.kickoff),
                c.current_season, 1),
       f.competition, f.competition_id,
       a.appeared, a.started, a.minutes, a.rated, a.rating, a.position, a.goals, a.penalty_goals, a.yellow_cards, a.red_cards
FROM appearances a
JOIN fixtures f ON f.kind = a.kind AND f.id = a.id
JOIN teams t ON t.team_id = a.team_id
LEFT JOIN countries c ON c.name = t.country
WHERE NOT EXISTS (SELECT 1 FROM done)
ON CONFLICT (kind, match_id, player_id) DO NOTHING;

INSERT INTO schema_backfills (name) VALUES ('player_match_stats') ON CONFLICT (name) DO NOTHING;
