package common

import "database/sql"

// SeasonHistoryEntry is one team's record for one season - the "trophy cabinet." Any field but
// TeamId/Season can be absent: league fields are unset if the team's league data wasn't
// recorded (shouldn't happen in practice, but the schema allows it - see init15_season_history.sql),
// and CupResult is unset for a season the team's country's cup hadn't completed, or that had no
// cup entry for them at all.
type SeasonHistoryEntry struct {
	TeamId        int64   `json:"teamId"`
	Season        int     `json:"season"`
	LeagueName    *string `json:"leagueName"`
	FinalPosition *int    `json:"finalPosition"`
	Promoted      bool    `json:"promoted"`
	Relegated     bool    `json:"relegated"`
	CupResult     *string `json:"cupResult"`
}

const UpsertSeasonHistoryLeagueStmtSql = `
INSERT INTO season_history (team_id, season, league_id, league_name, final_position, promoted, relegated)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (team_id, season) DO UPDATE SET
	league_id = EXCLUDED.league_id, league_name = EXCLUDED.league_name,
	final_position = EXCLUDED.final_position, promoted = EXCLUDED.promoted, relegated = EXCLUDED.relegated,
	updated_at = now()`

const UpsertSeasonHistoryCupResultStmtSql = `
INSERT INTO season_history (team_id, season, cup_result)
VALUES ($1, $2, $3)
ON CONFLICT (team_id, season) DO UPDATE SET cup_result = EXCLUDED.cup_result, updated_at = now()`

const GetSeasonHistoryStmtSql = "SELECT team_id, season, league_name, final_position, promoted, relegated, cup_result " +
	"FROM season_history WHERE team_id = $1 ORDER BY season DESC"

// RecordSeasonHistoryLeagueResult upserts a team's final league position and
// promotion/relegation outcome for one season, leaving any cup_result already recorded (from
// either order of arrival) untouched.
func RecordSeasonHistoryLeagueResult(dbCtx *DatabaseContext, teamId int64, season int, leagueId int64, leagueName string, finalPosition int, promoted bool, relegated bool) error {
	_, err := dbCtx.Db.Exec(UpsertSeasonHistoryLeagueStmtSql, teamId, season, leagueId, leagueName, finalPosition, promoted, relegated)
	return err
}

// RecordSeasonHistoryCupResult upserts a team's cup result for one season, leaving any league
// data already recorded untouched.
func RecordSeasonHistoryCupResult(dbCtx *DatabaseContext, teamId int64, season int, cupResult string) error {
	_, err := dbCtx.Db.Exec(UpsertSeasonHistoryCupResultStmtSql, teamId, season, cupResult)
	return err
}

// GetSeasonHistory returns every season a team has a recorded history entry for, newest first.
func GetSeasonHistory(dbCtx *DatabaseContext, teamId int64) ([]SeasonHistoryEntry, error) {
	rows, err := dbCtx.Db.Query(GetSeasonHistoryStmtSql, teamId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := make([]SeasonHistoryEntry, 0)
	for rows.Next() {
		var e SeasonHistoryEntry
		var leagueName, cupResult sql.NullString
		var finalPosition sql.NullInt64
		if err := rows.Scan(&e.TeamId, &e.Season, &leagueName, &finalPosition, &e.Promoted, &e.Relegated, &cupResult); err != nil {
			return nil, err
		}
		if leagueName.Valid {
			e.LeagueName = &leagueName.String
		}
		if finalPosition.Valid {
			pos := int(finalPosition.Int64)
			e.FinalPosition = &pos
		}
		if cupResult.Valid {
			e.CupResult = &cupResult.String
		}
		entries = append(entries, e)
	}
	return entries, nil
}
