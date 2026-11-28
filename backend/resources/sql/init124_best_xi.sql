-- Team of the week and team of the season (openspec/specs/best-xi). Each league picks a 4-4-2 of
-- its best rated players once every match of a matchday has finished its live replay
-- (engine.maybeCompleteBestXi), and once more for the whole season at its rollover
-- (engine.awardSeasonHonors). week_number 0 is the team of the season.
--
-- best_xi_rounds is the claim: whoever inserts the round row (ON CONFLICT DO NOTHING) writes its
-- players in the same transaction and sends the notifications, so several engine replicas finishing
-- the same matchday at once pick it exactly once. player_id has no foreign key: a retired player
-- keeps his selections (openspec/specs/retired-players).
--
-- Re-runnable: CREATE ... IF NOT EXISTS.

CREATE TABLE IF NOT EXISTS best_xi_rounds (
    league_id   bigint NOT NULL,
    season      int    NOT NULL,
    week_number int    NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (league_id, season, week_number)
);

CREATE TABLE IF NOT EXISTS best_xi_players (
    league_id   bigint   NOT NULL,
    season      int      NOT NULL,
    week_number int      NOT NULL,
    -- 1 goalkeeper, 2-5 defenders, 6-9 midfielders, 10-11 forwards.
    slot        smallint NOT NULL,
    -- 1 goalkeeper, 2 defence, 3 midfield, 4 attack (common.BestXiLine*).
    line        smallint NOT NULL,
    player_id   bigint   NOT NULL,
    team_id     bigint   NOT NULL,
    -- The matchday rating, or the season's average rating for the team of the season.
    rating      real     NOT NULL,
    goals       smallint NOT NULL DEFAULT 0,
    -- 1 for a matchday; the rated league matches counted for the team of the season.
    matches     smallint NOT NULL DEFAULT 1,
    PRIMARY KEY (league_id, season, week_number, slot),
    FOREIGN KEY (league_id, season, week_number) REFERENCES best_xi_rounds (league_id, season, week_number) ON DELETE CASCADE
);

-- CREATE INDEX IF NOT EXISTS locks its table in SHARE mode before it checks the index exists, and
-- this file runs again on every startup while results are being saved: each index is only created
-- when to_regclass, which takes no lock, does not find it (see init120).
DO $$
DECLARE
    idx record;
BEGIN
    FOR idx IN SELECT * FROM (VALUES
        ('best_xi_players_player_idx', 'CREATE INDEX IF NOT EXISTS best_xi_players_player_idx ON best_xi_players (player_id)')
    ) AS v(name, ddl) LOOP
        IF to_regclass(idx.name) IS NULL THEN
            EXECUTE idx.ddl;
        END IF;
    END LOOP;
END
$$;
