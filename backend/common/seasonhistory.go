package common

import "database/sql"

// SeasonHistoryEntry is one team's record for one season - the "trophy cabinet." Any field but
// TeamId/Season can be absent: league fields are unset if the team's league data wasn't
// recorded (shouldn't happen in practice, but the schema allows it - see init15_season_history.sql),
// and CupResult is unset for a season the team's country's cup hadn't completed, or that had no
// cup entry for them at all. CupResult is only ever a domestic cup ('Champion'); a regional cup win
// is InternationalCup (see openspec/specs/team-history).
type SeasonHistoryEntry struct {
	TeamId     int64   `json:"teamId"`
	Season     int     `json:"season"`
	LeagueId   *int64  `json:"leagueId"`
	LeagueName *string `json:"leagueName"`
	// LeagueTier is the tier of LeagueId (1 = top division), unset with it.
	LeagueTier    *int `json:"leagueTier"`
	FinalPosition *int `json:"finalPosition"`
	Promoted      bool `json:"promoted"`
	Relegated     bool `json:"relegated"`
	// ViaPlayoff is true when the promotion or relegation came through the promotion playoff (see
	// openspec/specs/league-pyramid "Season history marks playoff moves").
	ViaPlayoff bool    `json:"viaPlayoff"`
	CupResult  *string `json:"cupResult"`
	// InternationalCup is the regional cup edition the team won during this league season.
	InternationalCup *SeasonHistoryInternationalCup `json:"internationalCup"`
}

// SeasonHistoryInternationalCup is a regional cup edition won: its cup id, region (the client
// names the cup from it) and edition number within the region (cups.season).
type SeasonHistoryInternationalCup struct {
	CupId   int64  `json:"cupId"`
	Region  string `json:"region"`
	Edition int    `json:"edition"`
}

const UpsertSeasonHistoryLeagueStmtSql = `
INSERT INTO season_history (team_id, season, league_id, league_name, final_position, promoted, relegated, via_playoff,
	matches, points, wins, draws, losses, goals_scored, goals_received)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
ON CONFLICT (team_id, season) DO UPDATE SET
	league_id = EXCLUDED.league_id, league_name = EXCLUDED.league_name,
	final_position = EXCLUDED.final_position, promoted = EXCLUDED.promoted, relegated = EXCLUDED.relegated,
	via_playoff = EXCLUDED.via_playoff,
	matches = EXCLUDED.matches, points = EXCLUDED.points, wins = EXCLUDED.wins, draws = EXCLUDED.draws,
	losses = EXCLUDED.losses, goals_scored = EXCLUDED.goals_scored, goals_received = EXCLUDED.goals_received,
	updated_at = now()`

const UpsertSeasonHistoryCupResultStmtSql = `
INSERT INTO season_history (team_id, season, cup_result)
VALUES ($1, $2, $3)
ON CONFLICT (team_id, season) DO UPDATE SET cup_result = EXCLUDED.cup_result, updated_at = now()`

// UpsertSeasonHistoryInternationalCupStmtSql records a regional cup title, leaving the league
// columns and cup_result untouched.
const UpsertSeasonHistoryInternationalCupStmtSql = `
INSERT INTO season_history (team_id, season, international_cup_id)
VALUES ($1, $2, $3)
ON CONFLICT (team_id, season) DO UPDATE SET international_cup_id = EXCLUDED.international_cup_id, updated_at = now()`

const GetSeasonHistoryStmtSql = "SELECT sh.team_id, sh.season, sh.league_id, sh.league_name, l.tier, sh.final_position, " +
	"sh.promoted, sh.relegated, sh.via_playoff, sh.cup_result, ic.cup_id, ic.region, ic.season " +
	"FROM season_history sh LEFT JOIN leagues l ON l.league_id = sh.league_id " +
	"LEFT JOIN cups ic ON ic.cup_id = sh.international_cup_id " +
	"WHERE sh.team_id = $1 ORDER BY sh.season DESC"

// RecordSeasonHistoryLeagueResult upserts a team's final league position, its final table line
// (line's matches, points, wins, draws, losses and goals) and promotion/relegation outcome for one
// season, leaving any cup_result already recorded (from either order of arrival) untouched. A nil
// line stores no counters, which init101_league_season_history.sql backfills from the played
// matches on the next startup. viaPlayoff marks a promotion or relegation decided by the playoff.
func RecordSeasonHistoryLeagueResult(dbCtx *DatabaseContext, teamId int64, season int, leagueId int64, leagueName string, finalPosition int, line *ClassificationPosition, promoted bool, relegated bool, viaPlayoff bool) error {
	counters := make([]any, 7)
	if line != nil {
		counters = []any{line.Matches, line.Points, line.Wins, line.Draws, line.Losses, line.GoalsScored, line.GoalsReceived}
	}
	args := append([]any{teamId, season, leagueId, leagueName, finalPosition, promoted, relegated, viaPlayoff}, counters...)
	_, err := dbCtx.Db.Exec(UpsertSeasonHistoryLeagueStmtSql, args...)
	return err
}

// RecordSeasonHistoryCupResult upserts a team's domestic cup result for one season (the cup's own
// season), leaving any league data already recorded untouched.
func RecordSeasonHistoryCupResult(dbCtx *DatabaseContext, teamId int64, season int, cupResult string) error {
	_, err := dbCtx.Db.Exec(UpsertSeasonHistoryCupResultStmtSql, teamId, season, cupResult)
	return err
}

// RecordSeasonHistoryInternationalCup records that teamId won regional cup cupId during its
// league season season, leaving every other column of that season untouched.
func RecordSeasonHistoryInternationalCup(dbCtx *DatabaseContext, teamId int64, season int, cupId int64) error {
	_, err := dbCtx.Db.Exec(UpsertSeasonHistoryInternationalCupStmtSql, teamId, season, cupId)
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
		var leagueName, cupResult, cupRegion sql.NullString
		var leagueId, leagueTier, finalPosition, cupId, cupEdition sql.NullInt64
		if err := rows.Scan(&e.TeamId, &e.Season, &leagueId, &leagueName, &leagueTier, &finalPosition, &e.Promoted, &e.Relegated,
			&e.ViaPlayoff, &cupResult, &cupId, &cupRegion, &cupEdition); err != nil {
			return nil, err
		}
		if leagueTier.Valid {
			tier := int(leagueTier.Int64)
			e.LeagueTier = &tier
		}
		if cupId.Valid {
			e.InternationalCup = &SeasonHistoryInternationalCup{CupId: cupId.Int64, Region: cupRegion.String, Edition: int(cupEdition.Int64)}
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
	return entries, rows.Err()
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
	ViaPlayoff    bool   `json:"viaPlayoff"`
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
	"COALESCE(sh.goals_scored, 0), COALESCE(sh.goals_received, 0), sh.promoted, sh.relegated, sh.via_playoff " +
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
			&p.GoalsScored, &p.GoalsReceived, &p.Promoted, &p.Relegated, &p.ViaPlayoff); err != nil {
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
