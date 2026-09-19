package common

// AdminLeagueInfo is one row of the commissioner's per-league schedule table - see
// ListLeaguesForAdmin and handlers.AdminListLeaguesHandler.
type AdminLeagueInfo struct {
	LeagueId    int64         `json:"leagueId"`
	Country     string        `json:"country"`
	Name        string        `json:"name"`
	Tier        int           `json:"tier"`
	Schedule    MatchSchedule `json:"schedule"`
	HasOverride bool          `json:"hasOverride"`
}

const listLeaguesForAdminStmtSql = "SELECT l.league_id, l.country, l.name, l.tier, " +
	"ls.weekdays_mask, ls.hour, ls.minute " +
	"FROM leagues l LEFT JOIN league_schedule ls ON ls.league_id = l.league_id " +
	"ORDER BY l.country, l.tier, l.name"

// ListLeaguesForAdmin lists every league with its effective match schedule - the per-league
// override if one is set, otherwise the app-wide default (see GetMatchSchedule) - for the
// commissioner's per-league schedule table.
func ListLeaguesForAdmin(dbCtx *DatabaseContext) ([]AdminLeagueInfo, error) {
	rows, err := dbCtx.Db.Query(listLeaguesForAdminStmtSql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	defaultSchedule := GetMatchSchedule(dbCtx)

	leagues := make([]AdminLeagueInfo, 0)
	for rows.Next() {
		var info AdminLeagueInfo
		var mask, hour, minute *int
		if err := rows.Scan(&info.LeagueId, &info.Country, &info.Name, &info.Tier, &mask, &hour, &minute); err != nil {
			return nil, err
		}
		if mask != nil {
			info.Schedule = MatchSchedule{WeekdaysMask: *mask, Hour: *hour, Minute: *minute}
			info.HasOverride = true
		} else {
			info.Schedule = defaultSchedule
		}
		leagues = append(leagues, info)
	}
	return leagues, rows.Err()
}

const upsertLeagueScheduleStmtSql = "INSERT INTO league_schedule (league_id, weekdays_mask, hour, minute) " +
	"VALUES ($1, $2, $3, $4) " +
	"ON CONFLICT (league_id) DO UPDATE SET weekdays_mask = EXCLUDED.weekdays_mask, hour = EXCLUDED.hour, minute = EXCLUDED.minute"

// SetLeagueSchedule sets or replaces a per-league match schedule override. Callers must run
// ValidateMatchSchedule first, and should also re-date that league's already-generated unplayed
// matches onto the new schedule (see handlers.AdminSetLeagueScheduleHandler/
// engine.RescheduleUnplayedMatches, issue #67) - this alone only affects fixtures generated
// after the call.
func SetLeagueSchedule(dbCtx *DatabaseContext, leagueId int64, schedule MatchSchedule) error {
	_, err := dbCtx.Db.Exec(upsertLeagueScheduleStmtSql, leagueId, schedule.WeekdaysMask, schedule.Hour, schedule.Minute)
	return err
}

// ClearLeagueSchedule removes a league's override, reverting it to the app-wide default.
func ClearLeagueSchedule(dbCtx *DatabaseContext, leagueId int64) error {
	_, err := dbCtx.Db.Exec("DELETE FROM league_schedule WHERE league_id = $1", leagueId)
	return err
}

// GetLeagueScheduleOverrides returns every league's override, keyed by league id - used by
// engine.GenerateMatches so each league's fixtures are dated on its own schedule when it has
// one, falling back to the app-wide default (see GetMatchSchedule) for every other league.
func GetLeagueScheduleOverrides(dbCtx *DatabaseContext) (map[int64]MatchSchedule, error) {
	rows, err := dbCtx.Db.Query("SELECT league_id, weekdays_mask, hour, minute FROM league_schedule")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	overrides := make(map[int64]MatchSchedule)
	for rows.Next() {
		var leagueId int64
		var schedule MatchSchedule
		if err := rows.Scan(&leagueId, &schedule.WeekdaysMask, &schedule.Hour, &schedule.Minute); err != nil {
			return nil, err
		}
		overrides[leagueId] = schedule
	}
	return overrides, rows.Err()
}
