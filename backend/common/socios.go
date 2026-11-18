package common

// DefaultSocios is the member count every new team starts with - the crowd a default-size ground
// drew at the default ticket price before socios existed (see resources/sql/history/init80_socios.sql).
const DefaultSocios = 500

// RecentFormMatches is how many of a team's latest league matches count toward its "racha"
// (form) in the attendance model - see GetRecentLeagueForm.
const RecentFormMatches = 5

// RecentSelloutMatches is how many of a team's latest home league matches the CPU's stadium-
// expansion rule looks at when deciding whether demand really outgrew the ground - see
// GetRecentSelloutShare.
const RecentSelloutMatches = 10

const GetSociosStmtSql = "SELECT socios FROM teams WHERE team_id = $1"
const SetSociosStmtSql = "UPDATE teams SET socios = $1 WHERE team_id = $2"

// GetRecentLeagueFormStmtSql sums the league points (3/1/0) of teamId's latest played league
// matches, excluding one match id (the match being simulated right now is already flagged played
// by the time its attendance is drawn, and must not count toward its own pre-kickoff form).
const GetRecentLeagueFormStmtSql = `
SELECT coalesce(sum(m.pts), 0), count(*) FROM (
    SELECT CASE
        WHEN home_goals = away_goals THEN 1
        WHEN (home_team_id = $1 AND home_goals > away_goals) OR (away_team_id = $1 AND away_goals > home_goals) THEN 3
        ELSE 0 END AS pts
    FROM matches
    WHERE played = true AND (home_team_id = $1 OR away_team_id = $1) AND match_id <> $2
    ORDER BY match_date DESC
    LIMIT $3
) m`

// GetRecentHomeAttendanceStmtSql counts how many of teamId's latest played home league matches
// drew at least $2 (the stadium's capacity), out of the latest $3 with a recorded attendance.
const GetRecentHomeAttendanceStmtSql = `
SELECT count(*) FILTER (WHERE attendance >= $2), count(*) FROM (
    SELECT attendance FROM matches
    WHERE home_team_id = $1 AND played = true AND attendance IS NOT NULL
    ORDER BY match_date DESC
    LIMIT $3
) m`

// GetSocios returns teamId's current member count - see engine/socios.go for how it moves at
// each season rollover and engine.CalculateAttendance for what it feeds into.
func GetSocios(dbCtx *DatabaseContext, teamId int64) (int, error) {
	var socios int
	err := dbCtx.Db.QueryRow(GetSociosStmtSql, teamId).Scan(&socios)
	return socios, err
}

// SetSocios persists a new member count - callers (engine.driftSocios) are responsible for
// rounding it to a whole number first.
func SetSocios(dbCtx *DatabaseContext, teamId int64, socios int) error {
	_, err := dbCtx.Db.Exec(SetSociosStmtSql, socios, teamId)
	return err
}

// GetRecentLeagueForm returns the league points and match count of teamId's latest
// RecentFormMatches league matches, excluding excludeMatchId - 0 matches (a brand new team, or
// the very first round) simply means "no form to speak of" to the caller.
func GetRecentLeagueForm(dbCtx *DatabaseContext, teamId int64, excludeMatchId int64) (points int, matches int, err error) {
	err = dbCtx.Db.QueryRow(GetRecentLeagueFormStmtSql, teamId, excludeMatchId, RecentFormMatches).Scan(&points, &matches)
	return
}

// GetRecentHomeSellouts returns how many of teamId's latest RecentSelloutMatches home league
// matches drew at least capacity people, and how many such matches had a recorded attendance at
// all. Derived from matches.attendance, so it needs no column of its own - and a freshly expanded
// stadium naturally resets to "not sold out" since the old, smaller crowds no longer reach the new
// capacity.
func GetRecentHomeSellouts(dbCtx *DatabaseContext, teamId int64, capacity int) (soldOut int, total int, err error) {
	err = dbCtx.Db.QueryRow(GetRecentHomeAttendanceStmtSql, teamId, capacity, RecentSelloutMatches).Scan(&soldOut, &total)
	return
}
