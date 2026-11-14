package common

// UserCountry is one user's id and country - just enough to reassign them a fresh team after a
// leagues recreate (see handlers.RecreateLeaguesKeepingUsers, AssignTeam).
type UserCountry struct {
	UserId  int64
	Country string
}

// ListUsersWithCountry returns every registered user's id and country - see UserCountry.
func ListUsersWithCountry(dbCtx *DatabaseContext) ([]UserCountry, error) {
	rows, err := dbCtx.Db.Query("SELECT user_id, country FROM users")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]UserCountry, 0)
	for rows.Next() {
		var u UserCountry
		if err := rows.Scan(&u.UserId, &u.Country); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// ResetCurrentSeason sets every country's season back to 1 in place (UPDATE, not delete+
// recreate) so the season table's own global config - notably the admin-configured
// match_schedule_* default (see GetMatchSchedule) - survives a leagues recreate untouched.
func ResetCurrentSeason(dbCtx *DatabaseContext) error {
	_, err := dbCtx.Db.Exec("UPDATE countries SET current_season = 1")
	return err
}

// GetLeagueIdsForCountry returns every league id belonging to country - used right after
// creating a new country's leagues (see handlers.AdminAddCountryHandler) to generate fixtures
// for exactly those leagues.
func GetLeagueIdsForCountry(dbCtx *DatabaseContext, country string) ([]int64, error) {
	rows, err := dbCtx.Db.Query("SELECT league_id FROM leagues WHERE country = $1 AND archived_season IS NULL", country)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	leagueIds := make([]int64, 0)
	for rows.Next() {
		var leagueId int64
		if err := rows.Scan(&leagueId); err != nil {
			return nil, err
		}
		leagueIds = append(leagueIds, leagueId)
	}
	return leagueIds, rows.Err()
}

// GetLeagueIdsForCountryTier returns every league id belonging to country at tier - used to
// check whether a division already exists before creating it again (see
// handlers.AdminAddDivisionHandler, issue #68).
func GetLeagueIdsForCountryTier(dbCtx *DatabaseContext, country string, tier int) ([]int64, error) {
	rows, err := dbCtx.Db.Query("SELECT league_id FROM leagues WHERE country = $1 AND tier = $2 AND archived_season IS NULL", country, tier)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	leagueIds := make([]int64, 0)
	for rows.Next() {
		var leagueId int64
		if err := rows.Scan(&leagueId); err != nil {
			return nil, err
		}
		leagueIds = append(leagueIds, leagueId)
	}
	return leagueIds, rows.Err()
}
