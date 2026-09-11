package common

import (
	"database/sql"
	"encoding/json"
	"errors"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"os"
	"strconv"
	"time"
)

const ForUpdate = "FOR UPDATE"

const GetCurrentSeasonStmtSql = "select current_season from season"
const UpdateCurrentSeasonStmtSql = "UPDATE season SET current_season = $1"

const GetAllLeagueIdsStmtSql = "SELECT league_id FROM leagues"

const GetSeasonMatchCountsStmtSql = "SELECT count(*) FILTER (WHERE played = false), count(*) FROM matches WHERE season = $1"

const ResetClassificationForSeasonStmtSql = "UPDATE classification SET season = $1, matches = 0, points = 0, wins = 0, " +
	"losses = 0, draws = 0, goals_scored = 0, goals_received = 0 WHERE league_id = $2"

// Prefers the highest division (lowest tier number) that still has an unclaimed team, only
// spilling into the next division down once the current one is full of human owners - a new
// manager should land in the top flight while there's room, not in a random division. Within
// whichever tier ends up supplying the pick, the specific team is still random.
const GetFreeTeamStmtSql = "SELECT t.team_id FROM teams t JOIN leagues l ON t.league_id = l.league_id " +
	"WHERE t.owner_id IS NULL AND t.country = $1 ORDER BY l.tier ASC, random() LIMIT 1"

const GetLineupsForMatchStmtSql = "select a.team_id, coalesce(b.player_id, a.player_id) player_id, coalesce(b.\"position\", a.\"position\") \"position\" " +
	"from default_lineup_positions a left join (select c.team_id, d.* from lineup c join lineup_positions d " +
	"on (c.lineup_id = d.lineup_id) where team_id in ($1, $2) and match_id = $3) b on (a.team_id = b.team_id) where a.team_id in ($1, $2) "

const GetDivisionDetailsStmtSql = "SELECT a.team_id, b.name, a.matches, a.points, a.wins, a.losses, a.draws, a.goals_scored, a.goals_received " +
	" FROM classification a JOIN teams b ON a.team_id = b.team_id " +
	"WHERE a.league_id = $1 and a.season = $2 ORDER BY a.points DESC, a.goals_scored-a.goals_received DESC, a.goals_scored DESC, a.wins DESC, a.draws DESC"

const GetDivisionNameStmtSql = "SELECT name from leagues WHERE league_id = $1"
const GetDivisionTeamsStmtSql = "SELECT team_id from teams WHERE league_id = $1"

const GetTeamStmtByIdSql = "SELECT a.owner_id, c.name owner_name,  a.country, a.league_id, b.name league_name, a.name team_name, a.ai_level " +
	"FROM teams a JOIN leagues b ON (a.league_id = b.league_id) LEFT JOIN users c " +
	" ON (a.owner_id = c.user_id ) " +
	"WHERE a.team_id = $1"

const GetTeamDetailsStmtByOwnerSql = "SELECT team_id, country, league_id, name, name_changed FROM teams WHERE owner_id = $1"
const GetTeamDetailsStmtSql = "SELECT owner_id, country, league_id, name FROM teams WHERE team_id = $1"
const GetTeamIdsFromLeagueSql = "SELECT team_id FROM teams where league_id = $1"
const GetTeamOwnerStmtSql = "SELECT owner_id FROM teams WHERE team_id = $1"

const GetPlayerFullNameStmtSql = "SELECT first_name, last_name FROM players WHERE player_id = $1"

const GetPlayersFromTeamStmtSql = "SELECT player_id, p.team_id, first_name, last_name, birth_day, goal_keeper, defense, \"lateral\", pass, midfield, " +
	"dribbling, shot, \"header\", quick, creative, technical, wage, age, avatar, contract_years_remaining " +
	"FROM teams t JOIN players p ON t.team_id = p.team_id " +
	"WHERE t.team_id = $1"

const GetSimulableMatchesStmtSql = "select match_id, league_id, season, week_number, match_date, home_team_id, away_team_id, " +
	"home_goals, away_goals from matches where season = $1 and match_date > now() and match_date < now()+interval '15 minutes' and played is false and in_simulation is false order by league_id"

const GetSimulableMatchesStmtForUpdateSql = GetSimulableMatchesStmtSql + " " + ForUpdate

const GetUserIdByEmailStmtSql = "SELECT user_id FROM users WHERE email = $1"

// COALESCEd because a user who hasn't picked a country yet (see RegisterPendingUserStmtSql) has
// no row in `teams` to join and a NULL country - scanning those straight into Go's non-nullable
// string fields would error, where the empty-string/'0' sentinels let GetUserDetails represent
// "no team assigned yet" instead.
const GetUserStmtSql = "SELECT u.name userName, COALESCE(u.country, '') as country, u.provider, u.email, " +
	"COALESCE(t.team_id::text, '0') as teamId, COALESCE(t.name, '') as teamName, COALESCE(t.league_id::text, '0') as leagueId, u.is_admin " +
	"from users u left join teams t on (u.user_id = t.owner_id) where u.user_id = $1"

const GetUserIsAdminStmtSql = "SELECT is_admin FROM users WHERE user_id = $1"

const SetUserAdminStmtSql = "UPDATE users SET is_admin = $1 WHERE user_id = $2"

const GetWeekMatchesStmtSql = "select match_id, w.league_id, w.season, played, in_simulation, week_number, match_date, home_team_id, away_team_id, home_goals, away_goals," +
	" t0.name as home_team_name, t1.name as away_team_name " +
	"FROM (SELECT match_id, league_id, season, played, in_simulation, week_number, match_date, home_team_id, away_team_id, home_goals, away_goals " +
	"FROM matches where league_id = $1 and week_number = $2 and season = $3) w " +
	"join teams t0 on (w.home_team_id = t0.team_id) join teams t1 on (w.away_team_id = t1.team_id)"

const GetWeekMatchesByIdStmtSql = "select match_id, w.league_id, w.season, played, in_simulation, week_number, match_date, home_team_id, away_team_id, home_goals, away_goals," +
	" t0.name as home_team_name, t1.name as away_team_name " +
	"FROM (SELECT match_id, league_id, season, played, in_simulation, week_number, match_date, home_team_id, away_team_id, home_goals, away_goals " +
	"FROM matches where match_id = $1) w " +
	"join teams t0 on (w.home_team_id = t0.team_id) join teams t1 on (w.away_team_id = t1.team_id)"

const GetWeekMatchesEventsByIdStmtSql = "SELECT event_id, match_id, event_type, team_owner, other_team, minute, team_players, team_player_names, other_team_players, other_team_player_names, home_goals, away_goals " +
	"FROM match_events WHERE match_id = $1 order by minute asc"

const GetWeekMatchesInstantsByIdStmtSql = "SELECT match_id, minute, home_goals, away_goals, home_qua, away_qua, midfield_diff, homeattack_right_diff, homeattack_left_diff, homeattack_center_diff, awayattack_right_diff, awayattack_left_diff, awayattack_center_diff" +
	" FROM match_instants WHERE match_id = $1 order by minute asc"

const InsertTeamToClassificationStmtSql = "INSERT INTO classification (league_id, season, team_id) values ($1, $2, $3)"

const InsertMatch = "INSERT INTO matches (league_id, season, week_number, match_date, home_team_id, away_team_id) " +
	"VALUES ($1, $2, $3, $4, $5, $6)"
const InsertNewPlayerStmtSql = "INSERT INTO players (team_id, first_name, last_name, birth_day, " +
	"goal_keeper, defense, \"lateral\", pass, midfield, dribbling, shot, header, quick," +
	" creative, technical, avatar, wage, age, contract_years_remaining) " +
	"values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)"

const InsertDefaultLineupPositionsStmtSql = "INSERT INTO default_lineup_positions (team_id, player_id, position) values ($1, $2, $3)"
const GetDefaultLineupStmtSql = "SELECT player_id, position FROM default_lineup_positions WHERE team_id = $1"
const DeleteDefaultLineupPositionsStmtSql = "DELETE FROM default_lineup_positions WHERE team_id = $1"

const InsertMatchEventStmtSql = "INSERT INTO match_events (match_id, event_type, team_owner, other_team, minute, team_players, team_player_names, other_team_players, other_team_player_names, home_goals, away_goals) " +
	"VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)"

const InsertMatchInstantStmtSql = "" +
	"INSERT INTO match_instants(match_id, minute, home_goals, away_goals, home_qua, away_qua, midfield_diff, homeattack_right_diff, homeattack_left_diff, homeattack_center_diff, awayattack_right_diff, awayattack_left_diff, awayattack_center_diff) " +
	"VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)"

const RegisterUserStmtSql = "INSERT INTO users (name, provider, email, " +
	"country) VALUES ($1, $2, $3, $4) returning user_id"

// RegisterPendingUserStmtSql registers a brand-new sign-in without a country - used the first
// time someone logs in (Google or dev), before they've picked which country to play in (see
// handlers.SelectCountryHandler). Leaves `country` NULL (the column allows it) rather than
// guessing one, and deliberately doesn't touch `teams` at all - AssignTeam only runs once a
// country is chosen.
const RegisterPendingUserStmtSql = "INSERT INTO users (name, provider, email) VALUES ($1, $2, $3) returning user_id"
const UnAssignPlayersStmtSql = "UPDATE players SET team_id = NULL WHERE team_id = $1"

// Resets name_changed to false alongside the fresh name - a newly assigned owner gets their own
// one-time rename, regardless of whether a previous owner had already used theirs.
const UpdateTeamDetailsStmtSql = "UPDATE teams SET name = $1, owner_id = $2, name_changed = false WHERE team_id = $3"

const UpdateTeamClassificationStmtSql = "UPDATE classification set matches = matches+$1, points = points+$2, wins = wins+$3, " +
	"losses = losses+$4, draws = draws+$5, goals_scored = goals_scored+$6, goals_received = goals_received+$7 " +
	"where league_id = $8 and team_id = $9"

const UpdateMatchStmtSql = "UPDATE matches SET home_goals = $1, away_goals = $2, played = $3, in_simulation = $4 WHERE match_id = $5"
const UpdateMatchSimulateStmtSql = "UPDATE matches SET in_simulation = $1 WHERE "

const GetLeaguesWithTierStmtSql = "SELECT league_id, country, tier FROM leagues ORDER BY country, tier, group_number"

const GetRankedClassificationStmtSql = "SELECT team_id, points, goals_scored - goals_received, goals_scored " +
	"FROM classification WHERE league_id = $1 ORDER BY points DESC, (goals_scored - goals_received) DESC, goals_scored DESC"

const UpdateTeamLeagueStmtSql = "UPDATE teams SET league_id = $1 WHERE team_id = $2"

const UpdateClassificationLeagueStmtSql = "UPDATE classification SET league_id = $1 WHERE team_id = $2"

const UpsertPlayerUnavailabilityStmtSql = "INSERT INTO player_unavailability (player_id, reason, matches_remaining) " +
	"VALUES ($1, $2, $3) " +
	"ON CONFLICT (player_id) DO UPDATE SET " +
	"reason = CASE WHEN EXCLUDED.matches_remaining > player_unavailability.matches_remaining THEN EXCLUDED.reason ELSE player_unavailability.reason END, " +
	"matches_remaining = GREATEST(player_unavailability.matches_remaining, EXCLUDED.matches_remaining)"

const GetUnavailablePlayerIdsStmtSql = "SELECT pu.player_id FROM player_unavailability pu " +
	"JOIN players p ON p.player_id = pu.player_id WHERE p.team_id = $1 AND pu.matches_remaining > 0"

const DecrementUnavailabilityForTeamStmtSql = "UPDATE player_unavailability pu SET matches_remaining = matches_remaining - 1 " +
	"FROM players p WHERE p.player_id = pu.player_id AND p.team_id = $1"

const DeleteExpiredUnavailabilityStmtSql = "DELETE FROM player_unavailability WHERE matches_remaining <= 0"

const GetTeamBudgetStmtSql = "SELECT budget FROM teams WHERE team_id = $1"
const UpdateTeamBudgetStmtSql = "UPDATE teams SET budget = budget + $1 WHERE team_id = $2"

// player_id is nullable - only a transaction about one specific player (a transfer, a release)
// sets it; team-level transactions (wages, ticket revenue, prize money, sponsorship, stadium
// upgrades, admin adjustments) pass nil. See GetPlayerTransactions.
const InsertTeamTransactionStmtSql = "INSERT INTO team_transactions (team_id, amount, reason, description, player_id) VALUES ($1, $2, $3, $4, $5)"
const GetTeamTransactionsStmtSql = "SELECT transaction_id, team_id, amount, reason, description, created_at, player_id " +
	"FROM team_transactions WHERE team_id = $1 ORDER BY created_at DESC LIMIT $2"
const GetTeamWageBillStmtSql = "SELECT COALESCE(SUM(wage), 0) FROM players WHERE team_id = $1"

// nameTableSuffixByCountry maps a seeded country code to its names_<suffix>/lastnames_<suffix>
// table pair (see resources/sql/init.sql and init19_multi_country_names.sql). Table names can't
// be parameterized like ordinary values, so GetRandomName looks the suffix up here - an
// allowlist of the game's own seeded countries - rather than ever building SQL from the country
// string directly.
var nameTableSuffixByCountry = map[string]string{
	"es": "es",
	"en": "en",
	"fr": "fr",
	"de": "de",
	"it": "it",
	"pt": "pt",
}

// GetRandomName picks a random first/last name from the given country's name pool. Falls back
// to Spanish ("es") names for an unrecognized country code rather than failing - a missing/typo'd
// country should never block player generation.
func GetRandomName(dbCtx *DatabaseContext, country string) (string, string, error) {
	suffix, ok := nameTableSuffixByCountry[country]
	if !ok {
		suffix = "es"
	}
	stmtNames, err := dbCtx.Db.Prepare("select name from names_" + suffix + " order by random() limit 1")
	if err != nil {
		return "", "", err
	}
	defer stmtNames.Close()
	stmtLastNames, err := dbCtx.Db.Prepare("select lastname from lastnames_" + suffix + " order by random() limit 1")
	if err != nil {
		return "", "", err
	}
	defer stmtLastNames.Close()
	nameRow := stmtNames.QueryRow()
	lastNameRow := stmtLastNames.QueryRow()
	var name string
	var lastName string
	err = nameRow.Scan(&name)
	if err != nil {
		return "", "", err
	}
	err = lastNameRow.Scan(&lastName)
	if err != nil {
		return "", "", err
	}
	return name, lastName, nil
}

// SquadSize is how many players a freshly created team starts with.
const SquadSize = 18

// SeniorSkillMin/Max and SeniorAgeMin/Max are the ranges used for a new team's starting
// squad. AcademySkillMin/Max and AcademyAgeMin/Max (see CreateAcademyPlayers) are deliberately
// lower/younger - raw prospects, not the finished article.
const SeniorSkillMin, SeniorSkillMax = 30, 60
const SeniorAgeMin, SeniorAgeMax = 18, 32

func CreatePlayers(dbCtx *DatabaseContext, teamId int64, country string) {
	insertStmt, errInsert := dbCtx.Db.Prepare(InsertNewPlayerStmtSql)
	if errInsert != nil {
		log.Printf("error inserting new players %s", errInsert)
		return
	}
	defer insertStmt.Close()
	doCreatePlayers(dbCtx, teamId, insertStmt, SquadSize, SeniorSkillMin, SeniorSkillMax, SeniorAgeMin, SeniorAgeMax, country)
}

func GetTeamDetails(dbCtx *DatabaseContext, stmt *sql.Stmt, teamId int64) (*Team, error) {
	var userId = int64(-1)
	var country = ""
	var name = ""
	var leagueId = int64(-1)
	err := stmt.QueryRow(teamId).Scan(&userId, &country, &leagueId, &name)
	if err != nil {
		return nil, err
	}
	divName, err := GetDivisionName(dbCtx, leagueId)
	if err != nil {
		return nil, err
	}
	return &Team{Id: teamId,
		TeamName:     name,
		OwnerId:      userId,
		Country:      country,
		DivisionId:   leagueId,
		DivisionName: divName}, nil
}

// doCreatePlayers inserts `count` random players for a team, with skills uniformly drawn from
// [minSkill, maxSkill] and age from [minAge, maxAge]. Used both for a new team's starting
// squad and for youth academy backfills (see CreateAcademyPlayers), which use lower/younger
// ranges.
func doCreatePlayers(dbCtx *DatabaseContext, teamId int64, insertStmt *sql.Stmt, count int, minSkill int, maxSkill int, minAge int, maxAge int, country string) {
	for i := 0; i < count; i++ {
		firstName, lastName, _ := GetRandomName(dbCtx, country)
		avatar := RandomAvatarAsString()
		goalKeeper := minSkill + rand.Intn(maxSkill-minSkill+1)
		defense := minSkill + rand.Intn(maxSkill-minSkill+1)
		lateral := minSkill + rand.Intn(maxSkill-minSkill+1)
		pass := minSkill + rand.Intn(maxSkill-minSkill+1)
		midfield := minSkill + rand.Intn(maxSkill-minSkill+1)
		dribbling := minSkill + rand.Intn(maxSkill-minSkill+1)
		shot := minSkill + rand.Intn(maxSkill-minSkill+1)
		age := minAge + rand.Intn(maxAge-minAge+1)
		wage := PlayerWageFromSkills(goalKeeper, defense, lateral, pass, midfield, dribbling, shot)
		contractYears := RandomContractYears()
		insertStmt.Exec(&teamId, &firstName, &lastName, time.Now().UnixMilli(),
			goalKeeper,
			defense,
			lateral,
			pass,
			midfield,
			dribbling,
			shot,
			false,
			false,
			false,
			false,
			&avatar,
			wage,
			age,
			contractYears)
	}
}

// AssignTeam assigns a team to a user on a country
func AssignTeam(dbCtx *DatabaseContext, userId int64, country string) (*Team, error) {
	tx, err := dbCtx.Db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	//Get a free team
	var teamId = int64(-1)
	err = tx.QueryRow(GetFreeTeamStmtSql, &country).Scan(&teamId)
	if err != nil {
		return nil, err
	}

	//Un-assign current players
	if _, err = tx.Exec(UnAssignPlayersStmtSql, &teamId); err != nil {
		return nil, err
	}

	//Create new players
	insertStmt, err2 := tx.Prepare(InsertNewPlayerStmtSql)
	if err2 != nil {
		return nil, err2
	}
	defer insertStmt.Close()
	doCreatePlayers(dbCtx, teamId, insertStmt, SquadSize, SeniorSkillMin, SeniorSkillMax, SeniorAgeMin, SeniorAgeMax, country)

	//Update team details
	ustmt, err := tx.Prepare(UpdateTeamDetailsStmtSql)
	if err != nil {
		return nil, err
	}
	defer ustmt.Close()
	teamName := RandomTeamName(country)
	_, err = ustmt.Exec(teamName, userId, teamId)
	if err != nil {
		return nil, err
	}

	//Get team details
	teamStmt, err := tx.Prepare(GetTeamDetailsStmtSql)
	if err != nil {
		return nil, err
	}
	defer teamStmt.Close()
	team, err := GetTeamDetails(dbCtx, teamStmt, teamId)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}

	// The squad just created above is brand new, but any default lineup this team already had
	// (from whoever/whatever fielded it before - a CPU-managed team, or a previous owner) still
	// points at the players just orphaned by UnAssignPlayersStmtSql, not this one. Seed a
	// sensible lineup now rather than leaving the new owner's first matches fielding
	// stale/missing players (silently resolved to zero-skill ghosts by the simulation) until
	// they visit the lineup editor themselves. Best-effort: the team and its players already
	// exist regardless of whether this succeeds.
	if players, err := GetPlayersFromTeamId(dbCtx, teamId); err != nil {
		log.Errorf("error fetching new squad for team %d to seed its lineup: %s", teamId, err)
	} else if err := SetDefaultLineup(dbCtx, teamId, FlattenLineup(SelectBestLineup(*players))); err != nil {
		log.Errorf("error seeding default lineup for team %d: %s", teamId, err)
	}

	return team, nil
}

func GetLineupsForMatch(dbCtx *DatabaseContext, matchId int64, teamIds *[]int64) (*map[int64]*Lineup, error) {
	lineups := make(map[int64]*Lineup, 0)

	lineups[(*teamIds)[0]] = &Lineup{
		MatchId:   matchId,
		TeamId:    (*teamIds)[0],
		Positions: make([]LineupPosition, 0),
	}
	lineups[(*teamIds)[1]] = &Lineup{
		MatchId:   matchId,
		TeamId:    (*teamIds)[0],
		Positions: make([]LineupPosition, 0),
	}

	rows, err := dbCtx.Db.Query(GetLineupsForMatchStmtSql, (*teamIds)[0], (*teamIds)[1], matchId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var teamId int64
	var playerId int64
	var position int
	for rows.Next() {
		err := rows.Scan(&teamId, &playerId, &position)
		if err != nil {
			return nil, err
		}
		lineup := lineups[teamId]
		lineup.Positions = append(lineup.Positions, LineupPosition{
			PlayerId: playerId,
			Position: position,
		})
	}

	return &lineups, nil
}

func GetSimulableMatches(dbCtx *DatabaseContext) (*[]WeekMatch, error) {
	currentSeason := GetCurrentSeason(dbCtx)
	matches := make([]WeekMatch, 0)
	var matchId int64
	var divisionId int64
	var season int
	var weekNumber int
	var matchDate time.Time
	var home int64
	var away int64
	var homeGoals int
	var awayGoals int

	tx, err := dbCtx.Db.Begin()
	if err != nil {
		log.Errorf("error starting transaction for simulable matches: %s", err)
		return nil, err
	}
	rows, err := tx.Query(GetSimulableMatchesStmtForUpdateSql, currentSeason)
	if err != nil {
		log.Errorf("error getting matches for simulation: %s", err)
		_ = tx.Rollback()
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		err := rows.Scan(&matchId, &divisionId, &season, &weekNumber, &matchDate, &home, &away, &homeGoals, &awayGoals)
		if err != nil {
			log.Errorf("error getting match for simulation: %s", err)
		} else {
			matches = append(matches, WeekMatch{
				MatchId:    matchId,
				DivisionId: divisionId,
				Season:     season,
				WeekNumber: weekNumber,
				MatchDate:  matchDate,
				Home:       home,
				Away:       away,
				HomeGoals:  homeGoals,
				AwayGoals:  awayGoals,
			})
		}
	}
	if len(matches) > 0 {
		var updateMatches = UpdateMatchSimulateStmtSql
		var inClause = "match_id in ("
		var first = true
		for _, m := range matches {
			if !first {
				inClause += ", "
			}
			inClause += strconv.FormatInt(m.MatchId, 10)
			first = false
		}
		inClause += ")"
		updateMatches += inClause
		tx.Exec(updateMatches, true)
	}
	tx.Commit()
	return &matches, nil
}

func GetUserDetails(userId int64, dbCtx *DatabaseContext) (*LoggedUser, error) {
	var name string
	var country string
	var provider string
	var email string
	var teamId string
	var teamName string
	var leagueId string
	var isAdmin bool
	err := dbCtx.Db.QueryRow(GetUserStmtSql, userId).Scan(&name, &country, &provider, &email, &teamId, &teamName, &leagueId, &isAdmin)
	if err != nil {
		return nil, err
	}
	return &LoggedUser{
		Id:                      strconv.FormatInt(userId, 10),
		Name:                    name,
		Provider:                provider,
		Email:                   email,
		Country:                 country,
		TeamId:                  teamId,
		TeamName:                teamName,
		TeamLeague:              leagueId,
		IsAdmin:                 isAdmin,
		PendingCountrySelection: teamId == "0",
	}, nil
}

// SetUserCountry records the country a user picked during onboarding (see
// handlers.SelectCountryHandler) - called once, right before AssignTeam hands them a team in
// that country.
func SetUserCountry(dbCtx *DatabaseContext, userId int64, country string) error {
	_, err := dbCtx.Db.Exec("UPDATE users SET country = $1 WHERE user_id = $2", country, userId)
	return err
}

// IsUserAdmin is a lightweight admin check (no join) for the RequireAdmin middleware - it's on
// the hot path of every admin request, so it skips GetUserDetails' team join entirely.
func IsUserAdmin(dbCtx *DatabaseContext, userId int64) (bool, error) {
	var isAdmin bool
	err := dbCtx.Db.QueryRow(GetUserIsAdminStmtSql, userId).Scan(&isAdmin)
	return isAdmin, err
}

// SetUserAdmin grants or revokes admin access - see IsConfiguredAdminEmail for the only way
// it's currently called (auto-promotion at login time; there's no admin-managing-admins UI).
func SetUserAdmin(dbCtx *DatabaseContext, userId int64, isAdmin bool) error {
	_, err := dbCtx.Db.Exec(SetUserAdminStmtSql, isAdmin, userId)
	return err
}

func GetUserIdByEmail(email string, dbCtx *DatabaseContext) (int64, error) {
	var userId = int64(-1)
	err := dbCtx.Db.QueryRow(GetUserIdByEmailStmtSql, &email).Scan(&userId)
	return userId, err
}

func GetTeamByOwner(userId int64, dbCtx *DatabaseContext) (*Team, error) {
	var teamId int64
	var country string
	var leagueId int64
	var name string
	var nameChanged bool
	err := dbCtx.Db.QueryRow(GetTeamDetailsStmtByOwnerSql, userId).Scan(&teamId, &country, &leagueId, &name, &nameChanged)
	if err != nil {
		return nil, err
	}
	divisionName, err := GetDivisionName(dbCtx, leagueId)
	if err != nil {
		return nil, err
	}
	return &Team{
		Id:           teamId,
		OwnerId:      userId,
		Country:      country,
		TeamName:     name,
		DivisionId:   leagueId,
		DivisionName: divisionName,
		NameChanged:  nameChanged,
	}, nil
}

func GetTeamById(teamId int64, dbCtx *DatabaseContext) (*Team, error) {
	var userId sql.NullInt64
	var userName sql.NullString
	var country string
	var leagueId int64
	var leagueName string
	var name string
	var aiLevel int
	err := dbCtx.Db.QueryRow(GetTeamStmtByIdSql, teamId).Scan(&userId, &userName, &country, &leagueId, &leagueName, &name, &aiLevel)
	if err != nil {
		return nil, err
	}
	return &Team{
		Id:           teamId,
		OwnerId:      userId.Int64,
		OwnerName:    userName.String,
		Country:      country,
		TeamName:     name,
		DivisionId:   leagueId,
		DivisionName: leagueName,
		AiLevel:      aiLevel,
	}, nil
}

func GetDivisionDetails(dbCtx *DatabaseContext, leagueId int64) (*DivisionDetails, error) {
	leagueName, err := GetDivisionName(dbCtx, leagueId)
	if err != nil {
		return nil, err
	}
	currentSeason := GetCurrentSeason(dbCtx)

	rows, err := dbCtx.Db.Query(GetDivisionDetailsStmtSql, leagueId, currentSeason)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Classification
	positions := make([]ClassificationPosition, TeamsPerLeague)
	teamPosition := 1
	for rows.Next() {
		var teamId int64
		var teamName string
		var matches int
		var points int
		var wins int
		var losses int
		var draws int
		var goalsScored int
		var goalsReceived int

		rows.Scan(&teamId, &teamName, &matches, &points, &wins, &losses, &draws, &goalsScored, &goalsReceived)
		positions[teamPosition-1] = ClassificationPosition{
			Position:      teamPosition,
			TeamId:        teamId,
			TeamName:      teamName,
			Points:        points,
			Matches:       matches,
			Wins:          wins,
			Draws:         draws,
			Losses:        losses,
			GoalsScored:   goalsScored,
			GoalsReceived: goalsReceived,
		}
		teamPosition++
	}

	var currentWeek int
	currentWeek = positions[0].Matches
	//Last week matches
	var previousWeekMatches *[]WeekMatch
	var nextWeekMatches *[]WeekMatch
	if currentWeek > 1 {
		previousWeekMatches, err = WeekMatches(dbCtx, leagueId, currentSeason, currentWeek)
	}
	//Next week matches
	nextWeekMatches, err = WeekMatches(dbCtx, leagueId, currentSeason, currentWeek+1)

	if nextWeekMatches == nil {
		aux := make([]WeekMatch, 0)
		nextWeekMatches = &aux
	}
	if previousWeekMatches == nil {
		aux := make([]WeekMatch, 0)
		previousWeekMatches = &aux
	}

	return &DivisionDetails{
		Id:     leagueId,
		Season: currentSeason,
		Name:   leagueName,
		Classification: Classification{
			Positions: positions,
		},
		PreviousWeekMatches: *previousWeekMatches,
		NextWeekMatches:     *nextWeekMatches,
	}, nil
}

func WeekMatches(dbCtx *DatabaseContext, leagueId int64, season int, weekNumber int) (*[]WeekMatch, error) {
	rows, err := dbCtx.Db.Query(GetWeekMatchesStmtSql, leagueId, weekNumber, season)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	matches := make([]WeekMatch, 0)
	for rows.Next() {
		matches = append(matches, *readWeekMatch(rows))
	}
	return &matches, nil
}

func readWeekMatch(rows *sql.Rows) *WeekMatch {
	var matchId int64
	var leagueId int64
	var season int
	var played bool
	var inSimulation bool
	var weekNumber int
	var matchDate time.Time
	var homeTeamId int64
	var awayTeamId int64
	var homeGoals int
	var awayGoals int
	var homeTeamName string
	var awayTeamName string
	rows.Scan(&matchId, &leagueId, &season, &played, &inSimulation, &weekNumber, &matchDate, &homeTeamId, &awayTeamId, &homeGoals, &awayGoals,
		&homeTeamName, &awayTeamName)
	return &WeekMatch{
		MatchId:      matchId,
		DivisionId:   leagueId,
		Season:       season,
		WeekNumber:   weekNumber,
		MatchDate:    matchDate,
		Played:       played,
		InSimulation: inSimulation,
		Home:         homeTeamId,
		HomeName:     homeTeamName,
		Away:         awayTeamId,
		AwayName:     awayTeamName,
		HomeGoals:    homeGoals,
		AwayGoals:    awayGoals,
	}
}

func GetDivisionName(dbCtx *DatabaseContext, leagueId int64) (string, error) {
	divs, err := dbCtx.Db.Query(GetDivisionNameStmtSql, leagueId)
	if err != nil {
		return "", err
	}
	defer divs.Close()
	var divisionName string
	if divs.Next() {
		divs.Scan(&divisionName)
	}
	if divisionName == "" {
		return "", errors.New("league " + strconv.FormatInt(leagueId, 10) + " not found")
	}
	return divisionName, nil
}

func GetFullTeamDetailsByUser(userId int64, isOwner bool, dbCtx *DatabaseContext) (*TeamDetails, error) {
	team, errTeam := GetTeamByOwner(userId, dbCtx)
	if errTeam != nil {
		return nil, errTeam
	}

	players, err2 := GetPlayersFromTeamId(dbCtx, team.Id)
	if err2 != nil {
		return nil, err2
	}

	return &TeamDetails{
		Team:    team,
		Players: *players,
	}, nil
}

func WeekMatchById(dbCtx *DatabaseContext, matchId int64) (*WeekMatch, error) {
	rows, err := dbCtx.Db.Query(GetWeekMatchesByIdStmtSql, matchId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, nil
	}
	match := readWeekMatch(rows)
	match.Events = GetMatchEvents(dbCtx, matchId)
	match.MatchInstants = GetMatchInstants(dbCtx, matchId)
	lineups, err := GetLineupsForMatch(dbCtx, matchId, &[]int64{match.Home, match.Away})
	if err != nil {
		log.Errorf("error getting line ups for match: %v", err)
	} else {
		match.HomeLineup = (*lineups)[match.Home]
		match.AwayLineup = (*lineups)[match.Away]
	}
	homePlayers, err := GetPlayersFromTeamId(dbCtx, match.Home)
	if err != nil {
		log.Errorf("error getting home team players: %v", err)
	} else {
		pl := playerListToMap(homePlayers)
		match.HomePlayers = &pl
	}
	awayPlayers, err := GetPlayersFromTeamId(dbCtx, match.Away)
	if err != nil {
		log.Errorf("error getting away team players: %v", err)
	} else {
		pl := playerListToMap(awayPlayers)
		match.AwayPlayers = &pl
	}

	return match, nil
}

func playerListToMap(players *[]*Player) map[int64]Player {
	m := make(map[int64]Player)
	for _, player := range *players {
		m[player.Id] = *player
	}
	return m
}

func GetMatchEvents(dbCtx *DatabaseContext, matchId int64) *[]Event {
	rows, err := dbCtx.Db.Query(GetWeekMatchesEventsByIdStmtSql, matchId)
	if err != nil {
		return nil
	}
	defer rows.Close()
	events := make([]Event, 0)
	for rows.Next() {
		var eventId int64
		var matchId int64
		var eventType int
		var teamOwner int64
		var otherTeam int64
		var minute int
		var teamPlayers string
		var teamPlayerNames string
		var otherTeamPlayers string
		var otherTeamPlayerNames string
		var homeGoals int
		var awayGoals int
		rows.Scan(&eventId, &matchId, &eventType, &teamOwner, &otherTeam, &minute, &teamPlayers, &teamPlayerNames,
			&otherTeamPlayers, &otherTeamPlayerNames, &homeGoals, &awayGoals)

		event := Event{
			EventType:             eventType,
			Minute:                minute,
			TeamOwner:             teamOwner,
			TeamOwnerPlayers:      nil,
			TeamOwnerPlayersNames: nil,
			OtherTeam:             otherTeam,
			OtherTeamPlayers:      nil,
			OtherTeamPlayersNames: nil,
			HomeGoals:             homeGoals,
			AwayGoals:             awayGoals,
		}
		if teamPlayers != "" {
			players := make([]int64, 0)
			_ = json.Unmarshal([]byte(teamPlayers), &players)
			event.TeamOwnerPlayers = players
		}
		if teamPlayerNames != "" {
			players := make([]string, 0)
			_ = json.Unmarshal([]byte(teamPlayerNames), &players)
			event.TeamOwnerPlayersNames = players
		}
		if otherTeamPlayers != "" {
			players := make([]int64, 0)
			_ = json.Unmarshal([]byte(otherTeamPlayers), &players)
			event.OtherTeamPlayers = players
		}
		if otherTeamPlayerNames != "" {
			players := make([]string, 0)
			_ = json.Unmarshal([]byte(otherTeamPlayerNames), &players)
			event.OtherTeamPlayersNames = players
		}
		events = append(events, event)
	}
	return &events
}

func GetMatchInstants(dbCtx *DatabaseContext, matchId int64) *[]MatchInstant {
	rows, err := dbCtx.Db.Query(GetWeekMatchesInstantsByIdStmtSql, matchId)
	if err != nil {
		return nil
	}
	defer rows.Close()
	instants := make([]MatchInstant, 0)
	for rows.Next() {
		var matchId int64
		var minute int
		var homeGoals int
		var awayGoals int
		var homeQua string
		var awayQua string
		var midFieldDiff float32
		var homeAttackRightDiff float32
		var homeAttackLeftDiff float32
		var homeAttackCenterDiff float32
		var awayAttackRightDiff float32
		var awayAttackLeftDiff float32
		var awayAttackCenterDiff float32
		rows.Scan(&matchId, &minute, &homeGoals, &awayGoals, &homeQua, &awayQua, &midFieldDiff,
			&homeAttackRightDiff, &homeAttackLeftDiff, &homeAttackCenterDiff,
			&awayAttackRightDiff, &awayAttackLeftDiff, &awayAttackCenterDiff)

		homeTeamQua := TeamQualification{}
		awayTeamQua := TeamQualification{}
		if homeQua != "" {
			_ = json.Unmarshal([]byte(homeQua), &homeTeamQua)
		}
		if awayQua != "" {
			_ = json.Unmarshal([]byte(awayQua), &awayTeamQua)
		}
		instants = append(instants, MatchInstant{
			MatchId:              matchId,
			Minute:               minute,
			HomeGoals:            homeGoals,
			AwayGoals:            awayGoals,
			HomeQua:              homeTeamQua,
			AwayQua:              awayTeamQua,
			MidfieldDiff:         midFieldDiff,
			HomeAttackRightDiff:  homeAttackRightDiff,
			HomeAttackLeftDiff:   homeAttackLeftDiff,
			HomeAttackCenterDiff: homeAttackCenterDiff,
			AwayAttackRightDiff:  awayAttackRightDiff,
			AwayAttackLeftDiff:   awayAttackLeftDiff,
			AwayAttackCenterDiff: awayAttackCenterDiff,
		})
	}
	return &instants
}

func GetTeamDetailsById(teamId int64, isOwner bool, dbCtx *DatabaseContext) (*TeamDetails, error) {
	team, errTeam := GetTeamById(teamId, dbCtx)
	if errTeam != nil {
		return nil, errTeam
	}

	players, err := GetPlayersFromTeamId(dbCtx, teamId)
	if err != nil {
		return nil, err
	}

	return &TeamDetails{
		Team:    team,
		Players: *players,
	}, nil
}

func GetTeamOwner(dbCtx *DatabaseContext, teamId int64) (int64, error) {
	rows, err := dbCtx.Db.Query(GetTeamOwnerStmtSql, teamId)
	if err != nil {
		return -1, err
	}
	defer rows.Close()
	var userId = int64(-1)
	if rows.Next() {
		rows.Scan(&userId)
	}
	return userId, nil
}

func GetCurrentSeason(dbCtx *DatabaseContext) int {
	var season int
	dbCtx.Db.QueryRow(GetCurrentSeasonStmtSql).Scan(&season)
	return season
}

func SetCurrentSeason(dbCtx *DatabaseContext, season int) error {
	_, err := dbCtx.Db.Exec(UpdateCurrentSeasonStmtSql, season)
	return err
}

const GetTransferWindowOpenStmtSql = "SELECT transfer_window_open FROM season"
const SetTransferWindowOpenStmtSql = "UPDATE season SET transfer_window_open = $1"

// IsTransferWindowOpen reports whether the transfer/loan markets are currently open - a single
// global, admin-toggled flag (see handlers.SetTransferWindowHandler) rather than anything
// derived from the season schedule, since fixtures are generated per-league on independent
// clocks and there's no single "week of the season" to hang an automatic window on. Fails open
// (returns true) on a lookup error, so a DB hiccup can't silently freeze the whole market.
func IsTransferWindowOpen(dbCtx *DatabaseContext) bool {
	var open bool
	if err := dbCtx.Db.QueryRow(GetTransferWindowOpenStmtSql).Scan(&open); err != nil {
		return true
	}
	return open
}

func SetTransferWindowOpen(dbCtx *DatabaseContext, open bool) error {
	_, err := dbCtx.Db.Exec(SetTransferWindowOpenStmtSql, open)
	return err
}

func GetAllLeagueIds(dbCtx *DatabaseContext) ([]int64, error) {
	rows, err := dbCtx.Db.Query(GetAllLeagueIdsStmtSql)
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
	return leagueIds, nil
}

// GetSeasonMatchCounts returns how many matches exist for a season and how many of those
// are still unplayed, so callers can tell "season complete" (total > 0, unplayed == 0) apart
// from "season never started" (total == 0).
func GetSeasonMatchCounts(dbCtx *DatabaseContext, season int) (total int, unplayed int, err error) {
	err = dbCtx.Db.QueryRow(GetSeasonMatchCountsStmtSql, season).Scan(&unplayed, &total)
	return total, unplayed, err
}

// ResetClassificationForSeason rolls a league's existing classification rows into a new
// season. Note: the classification table's primary key is (league_id, team_id) - it has no
// season column in its key, so only the *current* season's standings can ever be stored per
// team; there is no historical per-season archive. Resetting in place (rather than inserting
// a new row per season) is what that constraint allows without a schema migration.
func ResetClassificationForSeason(dbCtx *DatabaseContext, leagueId int64, season int) error {
	_, err := dbCtx.Db.Exec(ResetClassificationForSeasonStmtSql, season, leagueId)
	return err
}

// GetLeaguesWithTier lists every league's country and tier (1 = top division), used to build
// each country's promotion/relegation pyramid.
func GetLeaguesWithTier(dbCtx *DatabaseContext) ([]LeagueTierInfo, error) {
	rows, err := dbCtx.Db.Query(GetLeaguesWithTierStmtSql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	leagues := make([]LeagueTierInfo, 0)
	for rows.Next() {
		var info LeagueTierInfo
		if err := rows.Scan(&info.LeagueId, &info.Country, &info.Tier); err != nil {
			return nil, err
		}
		leagues = append(leagues, info)
	}
	return leagues, nil
}

const GetTeamTierStmtSql = "SELECT l.tier FROM teams t JOIN leagues l ON l.league_id = t.league_id WHERE t.team_id = $1"

// GetTeamTier returns the tier (1 = top division) of the league a team currently plays in -
// used to scale tier-dependent per-match income like sponsorship (see engine.creditSponsorship).
func GetTeamTier(dbCtx *DatabaseContext, teamId int64) (int, error) {
	var tier int
	err := dbCtx.Db.QueryRow(GetTeamTierStmtSql, teamId).Scan(&tier)
	return tier, err
}

// GetRankedClassification returns a league's teams ordered best to worst (points, then goal
// difference, then goals scored - same tiebreak GetDivisionDetailsStmtSql uses).
func GetRankedClassification(dbCtx *DatabaseContext, leagueId int64) ([]TeamStanding, error) {
	rows, err := dbCtx.Db.Query(GetRankedClassificationStmtSql, leagueId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	standings := make([]TeamStanding, 0)
	for rows.Next() {
		var s TeamStanding
		if err := rows.Scan(&s.TeamId, &s.Points, &s.GoalDiff, &s.GoalsScored); err != nil {
			return nil, err
		}
		standings = append(standings, s)
	}
	return standings, nil
}

// UpdateTeamLeague reassigns a team to a different league (promotion/relegation).
func UpdateTeamLeague(dbCtx *DatabaseContext, teamId int64, newLeagueId int64) error {
	_, err := dbCtx.Db.Exec(UpdateTeamLeagueStmtSql, newLeagueId, teamId)
	return err
}

// UpdateClassificationLeague repoints a team's existing classification row at its new league
// after a promotion/relegation move, so the season reset that follows (which operates
// per-league) picks it up correctly.
func UpdateClassificationLeague(dbCtx *DatabaseContext, teamId int64, newLeagueId int64) error {
	_, err := dbCtx.Db.Exec(UpdateClassificationLeagueStmtSql, newLeagueId, teamId)
	return err
}

// AddPlayerUnavailability records a player as unavailable (suspended or injured) for their
// next `matches` matches. If the player is already unavailable, this only ever extends the
// duration (never shortens an existing suspension/injury) and keeps whichever reason belongs
// to the longer outstanding duration.
func AddPlayerUnavailability(dbCtx *DatabaseContext, playerId int64, reason int, matches int) error {
	_, err := dbCtx.Db.Exec(UpsertPlayerUnavailabilityStmtSql, playerId, reason, matches)
	return err
}

// GetUnavailablePlayerIds returns the set of a team's players currently serving a suspension
// or injury.
func GetUnavailablePlayerIds(dbCtx *DatabaseContext, teamId int64) (map[int64]bool, error) {
	rows, err := dbCtx.Db.Query(GetUnavailablePlayerIdsStmtSql, teamId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	unavailable := make(map[int64]bool)
	for rows.Next() {
		var playerId int64
		if err := rows.Scan(&playerId); err != nil {
			return nil, err
		}
		unavailable[playerId] = true
	}
	return unavailable, nil
}

// DecrementUnavailabilityForTeam counts a played match against every currently-unavailable
// player on a team, and clears anyone whose suspension/injury has now run out.
func DecrementUnavailabilityForTeam(dbCtx *DatabaseContext, teamId int64) error {
	if _, err := dbCtx.Db.Exec(DecrementUnavailabilityForTeamStmtSql, teamId); err != nil {
		return err
	}
	_, err := dbCtx.Db.Exec(DeleteExpiredUnavailabilityStmtSql)
	return err
}

func GetTeamBudget(dbCtx *DatabaseContext, teamId int64) (int64, error) {
	var budget int64
	err := dbCtx.Db.QueryRow(GetTeamBudgetStmtSql, teamId).Scan(&budget)
	return budget, err
}

// AdjustTeamBudget changes a team's budget by `amount` (negative for an expense) and records
// it in the ledger, atomically.
func AdjustTeamBudget(dbCtx *DatabaseContext, teamId int64, amount int64, reason int, description string) error {
	tx, err := dbCtx.Db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(UpdateTeamBudgetStmtSql, amount, teamId); err != nil {
		return err
	}
	if _, err := tx.Exec(InsertTeamTransactionStmtSql, teamId, amount, reason, description, nil); err != nil {
		return err
	}
	return tx.Commit()
}

func GetTeamTransactions(dbCtx *DatabaseContext, teamId int64, limit int) ([]TeamTransaction, error) {
	rows, err := dbCtx.Db.Query(GetTeamTransactionsStmtSql, teamId, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	transactions := make([]TeamTransaction, 0)
	for rows.Next() {
		var t TeamTransaction
		var description sql.NullString
		var playerId sql.NullInt64
		if err := rows.Scan(&t.Id, &t.TeamId, &t.Amount, &t.Reason, &description, &t.CreatedAt, &playerId); err != nil {
			return nil, err
		}
		t.Description = description.String
		if playerId.Valid {
			t.PlayerId = &playerId.Int64
		}
		transactions = append(transactions, t)
	}
	return transactions, nil
}

// GetTeamWageBill returns the combined wage of every player currently on a team's roster.
func GetTeamWageBill(dbCtx *DatabaseContext, teamId int64) (int64, error) {
	var total int64
	err := dbCtx.Db.QueryRow(GetTeamWageBillStmtSql, teamId).Scan(&total)
	return total, err
}

func GetPlayerFullName(dbCtx *DatabaseContext, playerId int64) (string, error) {
	var firstName string
	var lastName string
	row, err := dbCtx.Db.Query(GetPlayerFullNameStmtSql, playerId)
	if err != nil {
		return "", err
	}
	defer row.Close()
	if row.Next() {
		row.Scan(&firstName, &lastName)
	}
	return firstName + " " + lastName, nil
}

func GetPlayersFromTeamId(dbCtx *DatabaseContext, teamId int64) (*[]*Player, error) {
	rows, err := dbCtx.Db.Query(GetPlayersFromTeamStmtSql, teamId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	players := make([]*Player, 0)
	var playerId int64
	var firstName string
	var lastName string
	var birthDay int64
	var goalKeeper int
	var defense int
	var lateral int
	var pass int
	var midfield int
	var dribbling int
	var shot int
	var header bool
	var quick bool
	var creative bool
	var technical bool
	var wage int
	var age int
	var avatar string
	var contractYearsRemaining int
	for rows.Next() {
		err := rows.Scan(&playerId, &teamId, &firstName, &lastName, &birthDay, &goalKeeper, &defense, &lateral, &pass,
			&midfield, &dribbling, &shot, &header, &quick, &creative, &technical, &wage, &age, &avatar, &contractYearsRemaining)
		if err != nil {
			log.Errorf("error scanning player row: %s", err)
		} else {
			players = append(players, &Player{
				Id:                     playerId,
				TeamId:                 teamId,
				FirstName:              firstName,
				LastName:               lastName,
				BirthDay:               time.UnixMilli(birthDay),
				GoalKeeper:             goalKeeper,
				Defense:                defense,
				Lateral:                lateral,
				Pass:                   pass,
				Midfield:               midfield,
				Dribbling:              dribbling,
				Shot:                   shot,
				Header:                 header,
				Quick:                  quick,
				Creative:               creative,
				Technical:              technical,
				Wage:                   wage,
				Age:                    age,
				Avatar:                 GetAvatarFromString(avatar),
				ContractYearsRemaining: contractYearsRemaining,
			})
		}
	}
	return &players, nil
}

const GetAllTeamIdsStmtSql = "SELECT team_id FROM teams"
const CountTeamPlayersStmtSql = "SELECT count(*) FROM players WHERE team_id = $1"
const UpdatePlayerProgressionStmtSql = "UPDATE players SET age = $1, goal_keeper = $2, defense = $3, \"lateral\" = $4, " +
	"pass = $5, midfield = $6, dribbling = $7, shot = $8, wage = $9 WHERE player_id = $10"
const DeletePlayerStmtSql = "DELETE FROM players WHERE player_id = $1"

const AcademySkillMin, AcademySkillMax = 15, 40
const AcademyAgeMin, AcademyAgeMax = 17, 19

func GetAllTeamIds(dbCtx *DatabaseContext) ([]int64, error) {
	rows, err := dbCtx.Db.Query(GetAllTeamIdsStmtSql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	teamIds := make([]int64, 0)
	for rows.Next() {
		var teamId int64
		if err := rows.Scan(&teamId); err != nil {
			return nil, err
		}
		teamIds = append(teamIds, teamId)
	}
	return teamIds, nil
}

func CountTeamPlayers(dbCtx *DatabaseContext, teamId int64) (int, error) {
	var count int
	err := dbCtx.Db.QueryRow(CountTeamPlayersStmtSql, teamId).Scan(&count)
	return count, err
}

// UpdatePlayerProgression persists a player's new age and skills after a season's aging/
// attribute drift (see engine/progression.go), recomputing wage from the new skills.
func UpdatePlayerProgression(dbCtx *DatabaseContext, playerId int64, age, goalKeeper, defense, lateral, pass, midfield, dribbling, shot int) error {
	wage := PlayerWageFromSkills(goalKeeper, defense, lateral, pass, midfield, dribbling, shot)
	_, err := dbCtx.Db.Exec(UpdatePlayerProgressionStmtSql, age, goalKeeper, defense, lateral, pass, midfield, dribbling, shot, wage, playerId)
	return err
}

// RetirePlayer removes a player from the game. player_unavailability and transfer_listings
// both cascade-delete on the player_id FK, so those clean up automatically. But
// default_lineup_positions.player_id only goes to NULL on delete (see init.sql) - leaving that
// row behind would break GetDefaultLineup, which scans player_id into a non-nullable int64 -
// so that one is deleted explicitly first, the same way the transfer market already does for a
// sold player (see DeletePlayerLineupPositionsStmtSql).
func RetirePlayer(dbCtx *DatabaseContext, playerId int64) error {
	if _, err := dbCtx.Db.Exec(DeletePlayerLineupPositionsStmtSql, playerId); err != nil {
		return err
	}
	_, err := dbCtx.Db.Exec(DeletePlayerStmtSql, playerId)
	return err
}

// CreateAcademyPlayers generates `count` young prospects (lower skills, younger age than a
// starting senior squad - see AcademySkillMin/Max, AcademyAgeMin/Max) directly onto a team's
// roster, to backfill a squad thinned out by retirements. trainerSkill (0 if the team has no
// head coach) raises the top of the prospect skill range - see academySkillMax - so investing
// in coaching pays off in the pipeline too, not just in match-ready squad development.
func CreateAcademyPlayers(dbCtx *DatabaseContext, teamId int64, count int, trainerSkill int) error {
	if count <= 0 {
		return nil
	}
	country, err := GetTeamCountry(dbCtx, teamId)
	if err != nil {
		return err
	}
	insertStmt, err := dbCtx.Db.Prepare(InsertNewPlayerStmtSql)
	if err != nil {
		return err
	}
	defer insertStmt.Close()
	doCreatePlayers(dbCtx, teamId, insertStmt, count, AcademySkillMin, academySkillMax(trainerSkill), AcademyAgeMin, AcademyAgeMax, country)
	return nil
}

// academySkillMax raises the top of the academy prospect range by up to a quarter of a hired
// trainer's skill (0-99 -> +0-24), leaving the floor untouched - a team with no trainer (skill
// 0) gets exactly the historical AcademySkillMax.
func academySkillMax(trainerSkill int) int {
	return AcademySkillMax + trainerSkill/4
}

const GetTeamCountryStmtSql = "SELECT country FROM teams WHERE team_id = $1"

// GetTeamCountry looks up which country a team belongs to - needed wherever player/team name
// generation (see GetRandomName) has only a team id in scope, not the country itself.
func GetTeamCountry(dbCtx *DatabaseContext, teamId int64) (string, error) {
	var country string
	err := dbCtx.Db.QueryRow(GetTeamCountryStmtSql, teamId).Scan(&country)
	return country, err
}

func InsertMatches(dbCtx *DatabaseContext, weekMatches [][]*WeekMatch) error {
	stmt, err := dbCtx.Db.Prepare(InsertMatch)
	if err != nil {
		log.Errorf("error loading insert match statement: %s", err)
		return err
	}
	defer stmt.Close()
	for weekIdx, _ := range weekMatches {
		for _, match := range weekMatches[weekIdx] {
			_, err := stmt.Exec(match.DivisionId, match.Season, match.WeekNumber, match.MatchDate, match.Home, match.Away)
			if err != nil {
				log.Errorf("error inserting match: %s", err)
			}
		}
	}
	return nil
}

// GetDefaultLineup returns a team's standing lineup (used for any match without a
// match-specific override, see GetLineupsForMatch).
func GetDefaultLineup(dbCtx *DatabaseContext, teamId int64) (*Lineup, error) {
	rows, err := dbCtx.Db.Query(GetDefaultLineupStmtSql, teamId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	positions := make([]LineupPosition, 0)
	for rows.Next() {
		var playerId int64
		var position int
		if err := rows.Scan(&playerId, &position); err != nil {
			return nil, err
		}
		positions = append(positions, LineupPosition{PlayerId: playerId, Position: position})
	}

	formation, err := GetTeamFormation(dbCtx, teamId)
	if err != nil {
		return nil, err
	}

	substitutions, err := GetDefaultLineupSubstitutions(dbCtx, teamId)
	if err != nil {
		return nil, err
	}

	return &Lineup{TeamId: teamId, Formation: formation, Positions: positions, Substitutions: substitutions}, nil
}

// SetDefaultLineup atomically replaces a team's default lineup. Callers must have already
// validated that every position is filled exactly once and every player belongs to the team.
func SetDefaultLineup(dbCtx *DatabaseContext, teamId int64, positions []LineupPosition) error {
	tx, err := dbCtx.Db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Guards against two concurrent calls for the same team racing the delete-then-insert below
	// into a duplicate-key violation on default_lineup_positions_pkey (e.g. a league match and a
	// cup tie for the same AI-managed team landing in the same simulation pass - see
	// engine.RunAITeamManagement). Deliberately non-blocking (pg_try_, not pg_advisory_xact_lock)
	// so a losing caller never ties up a pooled connection waiting - if another call for this
	// team is already in flight, this one simply no-ops, since both would compute the identical
	// deterministic lineup from the same squad anyway.
	var acquiredLock bool
	if err := tx.QueryRow("SELECT pg_try_advisory_xact_lock($1)", teamId).Scan(&acquiredLock); err != nil {
		return err
	}
	if !acquiredLock {
		return nil
	}

	if _, err := tx.Exec(DeleteDefaultLineupPositionsStmtSql, teamId); err != nil {
		return err
	}

	insertStmt, err := tx.Prepare(InsertDefaultLineupPositionsStmtSql)
	if err != nil {
		return err
	}
	defer insertStmt.Close()

	for _, p := range positions {
		if _, err := insertStmt.Exec(teamId, p.PlayerId, p.Position); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func InsertLineup(dbCtx *DatabaseContext, teamId int64, lineup map[int][]int64, asDefault bool) error {
	insertStmt, errInsert := dbCtx.Db.Prepare(InsertDefaultLineupPositionsStmtSql)
	if errInsert != nil {
		log.Errorf("error inserting new default lineup %s", errInsert)
		return errInsert
	}
	defer insertStmt.Close()
	for position, players := range lineup {
		for _, player := range players {
			_, err := insertStmt.Exec(teamId, player, position)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func InsertTeamsToClassification(dbCtx *DatabaseContext, leagueId int64, season int, teams []int64) {
	stmt, err := dbCtx.Db.Prepare(InsertTeamToClassificationStmtSql)
	if err != nil {
		log.Errorf("error preparing insert team to classification: %s", err)
		return
	}
	defer stmt.Close()
	for _, t := range teams {
		if _, err := stmt.Exec(leagueId, season, t); err != nil {
			log.Errorf("error inserting team %d to classification: %s", t, err)
		}
	}
}

func GetDivisionTeams(dbCtx *DatabaseContext, leagueId int64) []int64 {
	teams := make([]int64, 0)
	rows, err := dbCtx.Db.Query(GetDivisionTeamsStmtSql, leagueId)
	if err != nil {
		log.Errorf("error getting division teams for league %d: %s", leagueId, err)
		return teams
	}
	defer rows.Close()
	for rows.Next() {
		var t int64
		rows.Scan(&t)
		teams = append(teams, t)
	}
	return teams
}

func CreateDatabaseContext() (*DatabaseContext, error) {
	port, _ := strconv.Atoi(os.Getenv("DB_PORT"))
	databaseDetails := DatabaseDetails{
		Host:     os.Getenv("DB_HOST"),
		Port:     port,
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		DbName:   os.Getenv("DB_DATABASE"),
		SslMode:  "disable",
	}

	dbCtx, err := OpenDatabase(databaseDetails)
	return dbCtx, err
}
