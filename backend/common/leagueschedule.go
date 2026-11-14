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
	// LocalTime is true when Schedule's hour and minute are the country's local time - a league with
	// no override follows its country's league kickoff for its tier (see EffectiveLeagueSchedule).
	// An override's hour and minute are on the server clock.
	LocalTime bool `json:"localTime"`
}

const listLeaguesForAdminStmtSql = "SELECT l.league_id, l.country, l.name, l.tier, " +
	"ls.weekdays_mask, ls.hour, ls.minute " +
	"FROM leagues l LEFT JOIN league_schedule ls ON ls.league_id = l.league_id " +
	"WHERE l.archived_season IS NULL ORDER BY l.country, l.tier, l.name"

// ListLeaguesForAdmin lists every active league with its effective match schedule - the
// per-league override if one is set, otherwise its country's league kickoff for its tier on the
// app-wide default's weekdays (see EffectiveLeagueSchedule) - for the commissioner's per-league
// schedule table.
func ListLeaguesForAdmin(dbCtx *DatabaseContext) ([]AdminLeagueInfo, error) {
	rows, err := dbCtx.Db.Query(listLeaguesForAdminStmtSql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	defaultSchedule := GetMatchSchedule(dbCtx)
	kickoffs := make(map[string]int)

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
			kickoff, ok := kickoffs[info.Country]
			if !ok {
				kickoff = GetCountryLeagueKickoff(dbCtx, info.Country)
				kickoffs[info.Country] = kickoff
			}
			info.Schedule = EffectiveLeagueSchedule(nil, defaultSchedule, info.Country, info.Tier, kickoff)
			info.LocalTime = true
		}
		leagues = append(leagues, info)
	}
	return leagues, rows.Err()
}

// EffectiveLeagueSchedule is the schedule a league's fixtures are dated on (see
// openspec/specs/match-scheduling "League kickoff schedule"): its admin-set override when it has
// one, otherwise the app-wide default's weekdays at its country's league kickoff (kickoffMinute,
// tier 1's local minute of day) plus LeagueKickoffTierStagger per tier below the first, read on the
// country's own clock. Pure.
func EffectiveLeagueSchedule(override *MatchSchedule, defaultSchedule MatchSchedule, country string, tier int, kickoffMinute int) MatchSchedule {
	if override != nil {
		return *override
	}
	minute := TierKickoffMinute(kickoffMinute, tier)
	if minute >= minutesPerDay {
		minute = minutesPerDay - 1
	}
	return MatchSchedule{
		WeekdaysMask: defaultSchedule.WeekdaysMask,
		Hour:         minute / 60,
		Minute:       minute % 60,
		Location:     CountryLocation(country),
	}
}

const getLeaguesCountryTierStmtSql = "SELECT league_id, country, tier FROM leagues WHERE league_id = ANY($1)"

// GetEffectiveLeagueSchedules returns the effective schedule of each of leagueIds (see
// EffectiveLeagueSchedule). A league whose country and tier cannot be read falls back to the
// app-wide default.
func GetEffectiveLeagueSchedules(dbCtx *DatabaseContext, leagueIds []int64) (map[int64]MatchSchedule, error) {
	defaultSchedule := GetMatchSchedule(dbCtx)
	overrides, err := GetLeagueScheduleOverrides(dbCtx)
	if err != nil {
		return nil, err
	}
	rows, err := dbCtx.Db.Query(getLeaguesCountryTierStmtSql, leagueIds)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	kickoffs := make(map[string]int)
	schedules := make(map[int64]MatchSchedule, len(leagueIds))
	for rows.Next() {
		var leagueId int64
		var country string
		var tier int
		if err := rows.Scan(&leagueId, &country, &tier); err != nil {
			return nil, err
		}
		var override *MatchSchedule
		if o, ok := overrides[leagueId]; ok {
			override = &o
		}
		kickoff, ok := kickoffs[country]
		if !ok {
			kickoff = GetCountryLeagueKickoff(dbCtx, country)
			kickoffs[country] = kickoff
		}
		schedules[leagueId] = EffectiveLeagueSchedule(override, defaultSchedule, country, tier, kickoff)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, id := range leagueIds {
		if _, ok := schedules[id]; !ok {
			schedules[id] = defaultSchedule
		}
	}
	return schedules, nil
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
