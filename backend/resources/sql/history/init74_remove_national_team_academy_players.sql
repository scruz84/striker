-- Cleans up a one-time data bug: before GetAllTeamIdsStmtSql excluded national teams
-- (common/database.go), engine.ProgressPlayers' season-end academy top-up saw each national
-- team's always-empty squad and generated it a full academy squad every season rollover - fake
-- players that, having a real team_id and a birth_country matching the national team's own
-- country, were themselves eligible to be called up alongside genuinely eligible club players
-- (see common.GetPlayersByBirthCountryStmtSql). A fresh world that never advanced a season while
-- national teams existed has none of these rows, so this is a no-op there.
DELETE FROM players WHERE team_id IN (SELECT team_id FROM teams WHERE is_national_team = true);
