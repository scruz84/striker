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
	LeagueId      *int64  `json:"leagueId"`
	LeagueName    *string `json:"leagueName"`
	FinalPosition *int    `json:"finalPosition"`
	Promoted      bool    `json:"promoted"`
	Relegated     bool    `json:"relegated"`
	CupResult     *string `json:"cupResult"`
}

const UpsertSeasonHistoryLeagueStmtSql = `
INSERT INTO season_history (team_id, season, league_id, league_name, final_position, promoted, relegated,
	matches, points, wins, draws, losses, goals_scored, goals_received)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
ON CONFLICT (team_id, season) DO UPDATE SET
	league_id = EXCLUDED.league_id, league_name = EXCLUDED.league_name,
	final_position = EXCLUDED.final_position, promoted = EXCLUDED.promoted, relegated = EXCLUDED.relegated,
	matches = EXCLUDED.matches, points = EXCLUDED.points, wins = EXCLUDED.wins, draws = EXCLUDED.draws,
	losses = EXCLUDED.losses, goals_scored = EXCLUDED.goals_scored, goals_received = EXCLUDED.goals_received,
	updated_at = now()`

const UpsertSeasonHistoryCupResultStmtSql = `
INSERT INTO season_history (team_id, season, cup_result)
VALUES ($1, $2, $3)
ON CONFLICT (team_id, season) DO UPDATE SET cup_result = EXCLUDED.cup_result, updated_at = now()`

const GetSeasonHistoryStmtSql = "SELECT team_id, season, league_id, league_name, final_position, promoted, relegated, cup_result " +
	"FROM season_history WHERE team_id = $1 ORDER BY season DESC"

// RecordSeasonHistoryLeagueResult upserts a team's final league position, its final table line
// (line's matches, points, wins, draws, losses and goals) and promotion/relegation outcome for one
// season, leaving any cup_result already recorded (from either order of arrival) untouched. A nil
// line stores no counters, which init101_league_season_history.sql backfills from the played
// matches on the next startup.
func RecordSeasonHistoryLeagueResult(dbCtx *DatabaseContext, teamId int64, season int, leagueId int64, leagueName string, finalPosition int, line *ClassificationPosition, promoted bool, relegated bool) error {
	counters := make([]any, 7)
	if line != nil {
		counters = []any{line.Matches, line.Points, line.Wins, line.Draws, line.Losses, line.GoalsScored, line.GoalsReceived}
	}
	args := append([]any{teamId, season, leagueId, leagueName, finalPosition, promoted, relegated}, counters...)
	_, err := dbCtx.Db.Exec(UpsertSeasonHistoryLeagueStmtSql, args...)
	return err
}

// RecordSeasonHistoryCupResult upserts a team's cup result for one season, leaving any league
// data already recorded untouched.
func RecordSeasonHistoryCupResult(dbCtx *DatabaseContext, teamId int64, season int, cupResult string) error {
	_, err := dbCtx.Db.Exec(UpsertSeasonHistoryCupResultStmtSql, teamId, season, cupResult)
	return err
}

// GetMostRecentTier1ChampionsStmtSql picks, per country, the most recently recorded tier-1
// champion (final_position = 1) regardless of what season number that country happens to be on
// right now - see GetMostRecentTier1Champions. DISTINCT ON (l.country) with this ORDER BY keeps
// exactly one row per country: the newest season, and among same-season ties (a country with
// more than one tier-1 group - see common.ValidateWorldConfig) the lowest league_id, mirroring
// the old live-standings version's "only ever contributes its first group's champion".
const GetMostRecentTier1ChampionsStmtSql = `
SELECT DISTINCT ON (l.country) l.country, sh.team_id
FROM season_history sh
JOIN leagues l ON l.league_id = sh.league_id
WHERE l.tier = $1 AND sh.final_position = 1
ORDER BY l.country, sh.season DESC, l.league_id ASC`

// GetMostRecentTier1Champions returns, per country, the team id of its most recently completed
// season's tier-1 champion - reads persisted history rather than a country's live
// classification, so it works once countries no longer all roll over together (see
// init75_season_per_country.sql, engine.AdvanceSeasonsIfComplete). Replaces the old
// regionChampions, which had to be read synchronously during a single shared rollover moment
// that no longer exists. A country that has never completed a season yet simply has no entry.
func GetMostRecentTier1Champions(dbCtx *DatabaseContext, tier int) (map[string]int64, error) {
	rows, err := dbCtx.Db.Query(GetMostRecentTier1ChampionsStmtSql, tier)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	champions := make(map[string]int64)
	for rows.Next() {
		var country string
		var teamId int64
		if err := rows.Scan(&country, &teamId); err != nil {
			return nil, err
		}
		champions[country] = teamId
	}
	return champions, rows.Err()
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
		var leagueId, finalPosition sql.NullInt64
		if err := rows.Scan(&e.TeamId, &e.Season, &leagueId, &leagueName, &finalPosition, &e.Promoted, &e.Relegated, &cupResult); err != nil {
			return nil, err
		}
		if leagueId.Valid {
			e.LeagueId = &leagueId.Int64
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

// LeagueSeasons is the league page's season picker: the league country's current season and
// every past season with history rows for the league, newest first.
type LeagueSeasons struct {
	Current int   `json:"current"`
	Past    []int `json:"past"`
}

// LeagueSeasonPosition is one line of a past season's final table. Name is the club's current name.
type LeagueSeasonPosition struct {
	Position      int    `json:"position"`
	TeamId        int64  `json:"teamId"`
	Name          string `json:"name"`
	Matches       int    `json:"matches"`
	Points        int    `json:"points"`
	Wins          int    `json:"wins"`
	Draws         int    `json:"draws"`
	Losses        int    `json:"losses"`
	GoalsScored   int    `json:"goalsScored"`
	GoalsReceived int    `json:"goalsReceived"`
	Promoted      bool   `json:"promoted"`
	Relegated     bool   `json:"relegated"`
}

// LeagueSeasonTable is a league's final table for a past season. Name is the league name recorded
// that season; TotalWeeks is 2 x (teams - 1), the season's matchdays.
type LeagueSeasonTable struct {
	Id         int64                  `json:"id"`
	Name       string                 `json:"name"`
	Season     int                    `json:"season"`
	TotalWeeks int                    `json:"totalWeeks"`
	Positions  []LeagueSeasonPosition `json:"positions"`
}

const GetLeaguePastSeasonsStmtSql = "SELECT DISTINCT season FROM season_history WHERE league_id = $1 AND season < $2 ORDER BY season DESC"

const GetLeagueSeasonTableStmtSql = "SELECT sh.final_position, sh.team_id, t.name, COALESCE(sh.league_name, ''), " +
	"COALESCE(sh.matches, 0), COALESCE(sh.points, 0), COALESCE(sh.wins, 0), COALESCE(sh.draws, 0), COALESCE(sh.losses, 0), " +
	"COALESCE(sh.goals_scored, 0), COALESCE(sh.goals_received, 0), sh.promoted, sh.relegated " +
	"FROM season_history sh JOIN teams t ON t.team_id = sh.team_id " +
	"WHERE sh.league_id = $1 AND sh.season = $2 AND sh.final_position IS NOT NULL ORDER BY sh.final_position, sh.team_id"

// GetLeagueSeasons returns a league's current season and its past seasons, newest first. An unknown
// league is sql.ErrNoRows.
func GetLeagueSeasons(dbCtx *DatabaseContext, leagueId int64) (*LeagueSeasons, error) {
	current, err := GetCurrentSeasonForLeague(dbCtx, leagueId)
	if err != nil {
		return nil, err
	}
	rows, err := dbCtx.Db.Query(GetLeaguePastSeasonsStmtSql, leagueId, current)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	seasons := &LeagueSeasons{Current: current, Past: make([]int, 0)}
	for rows.Next() {
		var season int
		if err := rows.Scan(&season); err != nil {
			return nil, err
		}
		seasons.Past = append(seasons.Past, season)
	}
	return seasons, rows.Err()
}

// GetLeagueSeasonTable returns a league's final table for one season from season_history, ordered
// by final position, or nil when that season has no history rows for the league (the current
// season never has any: its rows are written at rollover).
func GetLeagueSeasonTable(dbCtx *DatabaseContext, leagueId int64, season int) (*LeagueSeasonTable, error) {
	rows, err := dbCtx.Db.Query(GetLeagueSeasonTableStmtSql, leagueId, season)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	table := &LeagueSeasonTable{Id: leagueId, Season: season, Positions: make([]LeagueSeasonPosition, 0)}
	for rows.Next() {
		var p LeagueSeasonPosition
		if err := rows.Scan(&p.Position, &p.TeamId, &p.Name, &table.Name, &p.Matches, &p.Points, &p.Wins, &p.Draws, &p.Losses,
			&p.GoalsScored, &p.GoalsReceived, &p.Promoted, &p.Relegated); err != nil {
			return nil, err
		}
		table.Positions = append(table.Positions, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(table.Positions) == 0 {
		return nil, nil
	}
	table.TotalWeeks = 2 * (len(table.Positions) - 1)
	return table, nil
}
