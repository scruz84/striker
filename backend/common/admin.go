package common

import (
	"database/sql"
	"os"
	"strings"
)

// IsConfiguredAdminEmail reports whether email is listed in the ADMIN_EMAILS environment
// variable (comma-separated, case-insensitive) - the only way a user becomes an admin today,
// there's no admin-managing-admins UI. See DoExternalLogin, which checks this on every login
// and persists the promotion via SetUserAdmin.
func IsConfiguredAdminEmail(email string) bool {
	configured := os.Getenv("ADMIN_EMAILS")
	if configured == "" {
		return false
	}
	email = strings.ToLower(strings.TrimSpace(email))
	for _, candidate := range strings.Split(configured, ",") {
		if strings.ToLower(strings.TrimSpace(candidate)) == email {
			return true
		}
	}
	return false
}

// AdminUserSummary is one row of the commissioner's user list - see GetAllUsersWithTeams.
type AdminUserSummary struct {
	UserId   int64  `json:"userId"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Country  string `json:"country"`
	TeamId   *int64 `json:"teamId"`
	TeamName string `json:"teamName"`
	IsAdmin  bool   `json:"isAdmin"`
}

const GetAllUsersWithTeamsStmtSql = "SELECT u.user_id, u.name, u.email, u.country, t.team_id, t.name, u.is_admin " +
	"FROM users u LEFT JOIN teams t ON t.owner_id = u.user_id ORDER BY u.user_id"

// GetAllUsersWithTeams lists every registered user for the commissioner's user list. Small
// game, no pagination - see the frontend admin page for how this is presented.
func GetAllUsersWithTeams(dbCtx *DatabaseContext) ([]AdminUserSummary, error) {
	rows, err := dbCtx.Db.Query(GetAllUsersWithTeamsStmtSql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]AdminUserSummary, 0)
	for rows.Next() {
		var u AdminUserSummary
		var teamId sql.NullInt64
		var teamName sql.NullString
		if err := rows.Scan(&u.UserId, &u.Name, &u.Email, &u.Country, &teamId, &teamName, &u.IsAdmin); err != nil {
			return nil, err
		}
		if teamId.Valid {
			u.TeamId = &teamId.Int64
		}
		u.TeamName = teamName.String
		users = append(users, u)
	}
	return users, nil
}

// CountrySeason is one country's own current season and match progress - see AdminOverview.
// current_season moved onto `countries` (one counter per country, see
// init75_season_per_country.sql), so there's no single season number left to show for the
// whole game at once.
type CountrySeason struct {
	Country         string `json:"country"`
	Season          int    `json:"season"`
	MatchesTotal    int    `json:"matchesTotal"`
	MatchesUnplayed int    `json:"matchesUnplayed"`
}

// AdminOverview is a snapshot of the game's overall state for the commissioner dashboard - see
// handlers.AdminOverviewHandler for how it's assembled.
type AdminOverview struct {
	Seasons                       []CountrySeason  `json:"seasons"`
	UserCount                     int              `json:"userCount"`
	TeamCount                     int              `json:"teamCount"`
	LeagueCount                   int              `json:"leagueCount"`
	Simulation                    SimulationHealth `json:"simulation"`
	ActiveCups                    int              `json:"activeCups"`
	TransferWindowOpen            bool             `json:"transferWindowOpen"`
	MatchScheduleWeekdaysMask     int              `json:"matchScheduleWeekdaysMask"`
	MatchScheduleHour             int              `json:"matchScheduleHour"`
	MatchScheduleMinute           int              `json:"matchScheduleMinute"`
	MaxFriendliesPerWeek          int              `json:"maxFriendliesPerWeek"`
	FriendlyScheduleWeekdays      int              `json:"friendlyScheduleWeekdays"`
	PromotionSlotsPerGroup        int              `json:"promotionSlotsPerGroup"`
	RelegationSlotsPerGroup       int              `json:"relegationSlotsPerGroup"`
	FreeAgentInjectionsPerWeek    int              `json:"freeAgentInjectionsPerWeek"`
	OfferNegotiationWindowMinutes int              `json:"offerNegotiationWindowMinutes"`
}

func CountUsers(dbCtx *DatabaseContext) (int, error) {
	var n int
	err := dbCtx.Db.QueryRow("SELECT count(*) FROM users").Scan(&n)
	return n, err
}

func CountTeams(dbCtx *DatabaseContext) (int, error) {
	var n int
	err := dbCtx.Db.QueryRow("SELECT count(*) FROM teams").Scan(&n)
	return n, err
}

func CountLeagues(dbCtx *DatabaseContext) (int, error) {
	var n int
	err := dbCtx.Db.QueryRow("SELECT count(*) FROM leagues").Scan(&n)
	return n, err
}
