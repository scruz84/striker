package fmcommon

import (
	"database/sql"
	"encoding/json"
	"errors"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"strconv"
	"time"
)

const GetCurrentSeasonStmtSql = "select current_season from season"

const GetFreeTeamStmtSql = "SELECT team_id FROM teams WHERE owner_id IS NULL and country = $1 limit 1"

const GetLineupsForMatchStmtSql = "select a.team_id, coalesce(b.player_id, a.player_id) player_id, coalesce(b.\"position\", a.\"position\") \"position\" " +
	"from default_lineup_positions a left join (select c.team_id, d.* from lineup c join lineup_positions d " +
	"on (c.lineup_id = d.lineup_id) where team_id in ($1, $2) and match_id = $3) b on (a.team_id = b.team_id) where a.team_id in ($1, $2) "

const GetDivisionDetailsStmtSql = "SELECT a.team_id, b.name, a.matches, a.points, a.wins, a.losses, a.draws, a.goals_scored, a.goals_received " +
	" FROM classification a JOIN teams b ON a.team_id = b.team_id " +
	"WHERE a.league_id = $1 and a.season = $2 ORDER BY a.points DESC, a.goals_scored-a.goals_received DESC, a.goals_scored DESC, a.wins DESC, a.draws DESC"

const GetDivisionNameStmtSql = "SELECT name from leagues WHERE league_id = $1"
const GetDivisionTeamsStmtSql = "SELECT team_id from teams WHERE league_id = $1"

const GetTeamStmtByIdSql = "SELECT a.owner_id, c.name owner_name,  a.country, a.league_id, b.name league_name, a.name team_name " +
	"FROM teams a JOIN leagues b ON (a.league_id = b.league_id) LEFT JOIN users c " +
	" ON (a.owner_id = c.user_id ) " +
	"WHERE a.team_id = $1"

const GetTeamDetailsStmtByOwnerSql = "SELECT team_id, country, league_id, name FROM teams WHERE owner_id = $1"
const GetTeamDetailsStmtSql = "SELECT owner_id, country, league_id, name FROM teams WHERE team_id = $1"
const GetTeamIdsFromLeagueSql = "SELECT team_id FROM teams where league_id = $1"
const GetTeamOwnerStmtSql = "SELECT owner_id FROM teams WHERE team_id = $1"

const GetPlayerFullNameStmtSql = "SELECT first_name, last_name FROM players WHERE player_id = $1"

const GetPlayersFromTeamStmtSql = "SELECT player_id, p.team_id, first_name, last_name, birth_day, goal_keeper, defense, \"lateral\", pass, midfield, " +
	"dribbling, shot, \"header\", quick, creative, technical " +
	"FROM teams t JOIN players p ON t.team_id = p.team_id " +
	"WHERE t.team_id = $1"

const GetSimulableMatchesStmtSql = "select match_id, league_id, season, week_number, match_date, home_team_id, away_team_id, " +
	"home_goals, away_goals from matches where season = $1 and match_date > now() and match_date < now()+interval '15 minutes' and played is false order by league_id"

const GetUserIdByEmailStmtSql = "SELECT user_id FROM users WHERE email = $1"

const GetUserStmtSql = "SELECT u.name userName, u.country, u.provider, u.email, t.team_id, t.name as teamName, " +
	"t.league_id from users u left join teams t on (u.user_id = t.owner_id) where u.user_id = $1"

const GetWeekMatchesStmtSql = "select match_id, w.league_id, w.season, played, week_number, match_date, home_team_id, away_team_id, home_goals, away_goals," +
	" t0.name as home_team_name, t1.name as away_team_name " +
	"FROM (SELECT match_id, league_id, season, played, week_number, match_date, home_team_id, away_team_id, home_goals, away_goals " +
	"FROM matches where league_id = $1 and week_number = $2 and season = $3) w " +
	"join teams t0 on (w.home_team_id = t0.team_id) join teams t1 on (w.away_team_id = t1.team_id)"

const GetWeekMatchesByIdStmtSql = "select match_id, w.league_id, w.season, played, week_number, match_date, home_team_id, away_team_id, home_goals, away_goals," +
	" t0.name as home_team_name, t1.name as away_team_name " +
	"FROM (SELECT match_id, league_id, season, played, week_number, match_date, home_team_id, away_team_id, home_goals, away_goals " +
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
	" creative, technical) values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)"

const InsertDefaultLineupPositionsStmtSql = "INSERT INTO default_lineup_positions (team_id, player_id, position) values ($1, $2, $3)"

const InsertMatchEventStmtSql = "INSERT INTO match_events (match_id, event_type, team_owner, other_team, minute, team_players, team_player_names, other_team_players, other_team_player_names, home_goals, away_goals) " +
	"VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)"

const InsertMatchInstantStmtSql = "" +
	"INSERT INTO match_instants(match_id, minute, home_goals, away_goals, home_qua, away_qua, midfield_diff, homeattack_right_diff, homeattack_left_diff, homeattack_center_diff, awayattack_right_diff, awayattack_left_diff, awayattack_center_diff) " +
	"VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)"

const RegisterUserStmtSql = "INSERT INTO users (name, provider, email, " +
	"country) VALUES ($1, $2, $3, $4) returning user_id"
const UnAssignPlayersStmtSql = "UPDATE players SET team_id = NULL WHERE team_id = $1"
const UpdateTeamDetailsStmtSql = "UPDATE teams SET name = $1, owner_id = $2 WHERE team_id = $3	"

const UpdateTeamClassificationStmtSql = "UPDATE classification set matches = matches+$1, points = points+$2, wins = wins+$3, " +
	"losses = losses+$4, draws = draws+$5, goals_scored = goals_scored+$6, goals_received = goals_received+$7 " +
	"where league_id = $8 and team_id = $9"

const UpdateMatchStmtSql = "UPDATE matches SET home_goals = $1, away_goals = $2, played = $3 WHERE match_id = $4"

func GetRandomName(dbCtx *DatabaseContext) (string, string, error) {
	stmtNames, err := dbCtx.Db.Prepare("select name from names_es order by random() limit 1")
	defer stmtNames.Close()
	stmtLastNames, err := dbCtx.Db.Prepare("select lastname from lastnames_es order by random() limit 1")
	defer stmtLastNames.Close()
	if err != nil {
		return "", "", err
	}
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

func CreatePlayers(dbCtx *DatabaseContext, teamId int64) {
	insertStmt, errInsert := dbCtx.Db.Prepare(InsertNewPlayerStmtSql)
	if errInsert != nil {
		log.Printf("error inserting new players %s", errInsert)
	} else {
		doCreatePlayers(dbCtx, teamId, insertStmt)
	}
	defer insertStmt.Close()
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

func doCreatePlayers(dbCtx *DatabaseContext, teamId int64, insertStmt *sql.Stmt) {
	for i := 0; i < 18; i++ {
		rand.Seed(time.Now().UnixNano())
		min := 30
		max := 60
		firstName, lastName, _ := GetRandomName(dbCtx)
		insertStmt.Exec(&teamId, &firstName, &lastName, time.Now().UnixMilli(),
			rand.Intn(max-min+1)+min,
			rand.Intn(max-min+1)+min,
			rand.Intn(max-min+1)+min,
			rand.Intn(max-min+1)+min,
			rand.Intn(max-min+1)+min,
			rand.Intn(max-min+1)+min,
			rand.Intn(max-min+1)+min,
			false,
			false,
			false,
			false)
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
	defer insertStmt.Close()
	if err2 != nil {
		return nil, err2
	}
	doCreatePlayers(dbCtx, teamId, insertStmt)

	//Update team details
	var ustmt *sql.Stmt
	ustmt, err = tx.Prepare(UpdateTeamDetailsStmtSql)
	defer ustmt.Close()
	tName, _, err := GetRandomName(dbCtx)
	if err != nil {
		return nil, err
	}
	teamName := tName + " Team"
	_, err = ustmt.Exec(teamName, userId, teamId)
	if err != nil {
		return nil, err
	}

	//Get team details
	var teamStmt *sql.Stmt
	teamStmt, err = tx.Prepare(GetTeamDetailsStmtSql)
	defer teamStmt.Close()
	if err != nil {
		return nil, err
	}
	team, err := GetTeamDetails(dbCtx, teamStmt, teamId)
	if err = tx.Commit(); err != nil {
		return nil, err
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
	defer rows.Close()
	if err != nil {
		return nil, err
	}

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

	rows, err := dbCtx.Db.Query(GetSimulableMatchesStmtSql, currentSeason)
	defer rows.Close()
	if err != nil {
		log.Errorf("error getting matches for simulation: %s", err)
		return nil, err
	}
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
	err := dbCtx.Db.QueryRow(GetUserStmtSql, userId).Scan(&name, &country, &provider, &email, &teamId, &teamName, &leagueId)
	if err != nil {
		return nil, err
	}
	return &LoggedUser{
		Id:         strconv.FormatInt(userId, 10),
		Name:       name,
		Provider:   provider,
		Email:      email,
		Country:    country,
		TeamId:     teamId,
		TeamName:   teamName,
		TeamLeague: leagueId,
	}, nil
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
	err := dbCtx.Db.QueryRow(GetTeamDetailsStmtByOwnerSql, userId).Scan(&teamId, &country, &leagueId, &name)
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
	}, nil
}

func GetTeamById(teamId int64, dbCtx *DatabaseContext) (*Team, error) {
	var userId sql.NullInt64
	var userName sql.NullString
	var country string
	var leagueId int64
	var leagueName string
	var name string
	err := dbCtx.Db.QueryRow(GetTeamStmtByIdSql, teamId).Scan(&userId, &userName, &country, &leagueId, &leagueName, &name)
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
	}, nil
}

func GetDivisionDetails(dbCtx *DatabaseContext, leagueId int64) (*DivisionDetails, error) {
	leagueName, err := GetDivisionName(dbCtx, leagueId)
	if err != nil {
		return nil, err
	}
	currentSeason := GetCurrentSeason(dbCtx)

	rows, err := dbCtx.Db.Query(GetDivisionDetailsStmtSql, leagueId, currentSeason)
	defer rows.Close()
	if err != nil {
		return nil, err
	}

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
	defer rows.Close()
	if err != nil {
		return nil, err
	}
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
	var weekNumber int
	var matchDate time.Time
	var homeTeamId int64
	var awayTeamId int64
	var homeGoals int
	var awayGoals int
	var homeTeamName string
	var awayTeamName string
	rows.Scan(&matchId, &leagueId, &season, &played, &weekNumber, &matchDate, &homeTeamId, &awayTeamId, &homeGoals, &awayGoals,
		&homeTeamName, &awayTeamName)
	return &WeekMatch{
		MatchId:    matchId,
		DivisionId: leagueId,
		Season:     season,
		WeekNumber: weekNumber,
		MatchDate:  matchDate,
		Played:     played,
		Home:       homeTeamId,
		HomeName:   homeTeamName,
		Away:       awayTeamId,
		AwayName:   awayTeamName,
		HomeGoals:  homeGoals,
		AwayGoals:  awayGoals,
	}
}

func GetDivisionName(dbCtx *DatabaseContext, leagueId int64) (string, error) {
	divs, err := dbCtx.Db.Query(GetDivisionNameStmtSql, leagueId)
	defer divs.Close()
	if err != nil {
		return "", err
	}
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
	defer rows.Close()
	if err != nil {
		return nil, err
	}
	if !rows.Next() {
		return nil, nil
	}
	match := readWeekMatch(rows)
	match.Events = GetMatchEvents(dbCtx, matchId)
	match.MatchInstants = GetMatchInstants(dbCtx, matchId)
	return match, nil
}

func GetMatchEvents(dbCtx *DatabaseContext, matchId int64) *[]Event {
	rows, err := dbCtx.Db.Query(GetWeekMatchesEventsByIdStmtSql, matchId)
	defer rows.Close()
	if err != nil {
		return nil
	}
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
			event.TeamOwnerPlayers = players
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
	defer rows.Close()
	if err != nil {
		return nil
	}
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
			HomeAttackCenterDiff: homeAttackRightDiff,
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
	defer rows.Close()
	var userId = int64(-1)
	if rows.Next() {
		rows.Scan(&userId)
	}
	return userId, err
}

func GetCurrentSeason(dbCtx *DatabaseContext) int {
	var season int
	dbCtx.Db.QueryRow(GetCurrentSeasonStmtSql).Scan(&season)
	return season
}

func GetPlayerFullName(dbCtx *DatabaseContext, playerId int64) (string, error) {
	var firstName string
	var lastName string
	row, err := dbCtx.Db.Query(GetPlayerFullNameStmtSql, playerId)
	defer row.Close()
	if err != nil {
		return "", err
	}
	if row.Next() {
		row.Scan(&firstName, &lastName)
	}
	return firstName + " " + lastName, nil
}

func GetPlayersFromTeamId(dbCtx *DatabaseContext, teamId int64) (*[]*Player, error) {
	rows, err := dbCtx.Db.Query(GetPlayersFromTeamStmtSql, teamId)
	defer rows.Close()
	if err != nil {
		return nil, err
	}
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
	for rows.Next() {
		err := rows.Scan(&playerId, &teamId, &firstName, &lastName, &birthDay, &goalKeeper, &defense, &lateral, &pass,
			&midfield, &dribbling, &shot, &header, &quick, &creative, &technical)
		if err != nil {
			log.Fatal(err)
		} else {
			players = append(players, &Player{
				Id:         playerId,
				TeamId:     teamId,
				FirstName:  firstName,
				LastName:   lastName,
				BirthDay:   time.UnixMilli(birthDay),
				GoalKeeper: goalKeeper,
				Defense:    defense,
				Lateral:    lateral,
				Pass:       pass,
				Midfield:   midfield,
				Dribbling:  dribbling,
				Shot:       shot,
				Header:     header,
				Quick:      quick,
				Creative:   creative,
				Technical:  technical,
			})
		}
	}
	return &players, nil
}

func InsertMatches(dbCtx *DatabaseContext, weekMatches [][]*WeekMatch) error {
	stmt, err := dbCtx.Db.Prepare(InsertMatch)
	defer stmt.Close()
	if err != nil {
		log.Errorf("error loading insert match statement: %s", err)
		return err
	}
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

func InsertLineup(dbCtx *DatabaseContext, teamId int64, lineup map[int][]int64, asDefault bool) error {
	insertStmt, errInsert := dbCtx.Db.Prepare(InsertDefaultLineupPositionsStmtSql)
	defer insertStmt.Close()
	if errInsert != nil {
		log.Errorf("error inserting new default lineup %s", errInsert)
		return errInsert
	}
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
	stmt, _ := dbCtx.Db.Prepare(InsertTeamToClassificationStmtSql)
	defer stmt.Close()
	for _, t := range teams {
		stmt.Exec(leagueId, season, t)
	}
}

func GetDivisionTeams(dbCtx *DatabaseContext, leagueId int64) []int64 {
	teams := make([]int64, 0)
	rows, _ := dbCtx.Db.Query(GetDivisionTeamsStmtSql, leagueId)
	defer rows.Close()
	for rows.Next() {
		var t int64
		rows.Scan(&t)
		teams = append(teams, t)
	}
	return teams
}
