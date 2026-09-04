-- A team's trophy cabinet: one row per (team, season), independent of the current-season
-- classification table (which is reset in place at each rollover - see
-- ResetClassificationForSeason - so it can never hold last season's standings). Written from
-- two independent places that can each fire first depending on timing, so every column but the
-- key is nullable and both writers upsert rather than assuming the row already/doesn't exist:
-- engine.recordSeasonHistory (league final position + promotion/relegation, at season rollover)
-- and engine.recordCupChampion (cup_result, whenever a cup completes - usually mid-season,
-- before that season's rollover, but not guaranteed).
CREATE TABLE IF NOT EXISTS season_history (
    team_id        bigint NOT NULL REFERENCES teams(team_id) ON DELETE CASCADE,
    season         int NOT NULL,
    league_id      bigint,
    league_name    varchar(200),
    final_position int,
    promoted       boolean NOT NULL DEFAULT false,
    relegated      boolean NOT NULL DEFAULT false,
    cup_result     varchar(50),
    updated_at     timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (team_id, season)
);

CREATE INDEX IF NOT EXISTS season_history_idx1 ON season_history(team_id, season DESC);
