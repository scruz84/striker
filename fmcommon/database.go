package fmcommon

import (
	"database/sql"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"strconv"
	"time"
)

const GetFreeTeamStmtSql = "SELECT team_id FROM teams WHERE owner_id IS NULL and country = $1 limit 1"

const GetLineupsForMatchStmt = "GetLineupsForMatchStmt"
const GetLineupsForMatchStmtSql = "select a.team_id, coalesce(b.player_id, a.player_id) player_id, coalesce(b.\"position\", a.\"position\") \"position\" from default_lineup_positions a left join (select c.team_id, d.* from lineup c join lineup_positions d on (c.lineup_id = d.lineup_id) where team_id in ($1, $2) and match_id = $3) b on (a.team_id = b.team_id) where a.team_id in ($1, $2) "

const GetDivisionNameStmtSql = "SELECT name from leagues WHERE league_id = $1"
const GetDivisionTeamsStmtSql = "SELECT team_id from teams WHERE league_id = $1"

const GetTeamDetailsStmtByOwner = "GetTeamDetailsStmtByOwner"
const GetTeamDetailsStmtByOwnerSql = "SELECT team_id, country, league_id, name FROM teams WHERE owner_id = $1"
const GetTeamDetailsStmtSql = "SELECT owner_id, country, league_id, name FROM teams WHERE team_id = $1"
const GetTeamIdsFromLeagueSql = "SELECT team_id FROM teams where league_id = $1"
const GetPlayersFromTeamStmt = "GetPlayersFromTeamStmt"
const GetPlayersFromTeamStmtSql = "SELECT player_id, p.team_id, first_name, last_name, birth_day, goal_keeper, defense, \"lateral\", pass, midfield, " +
	"dribbling, shot, \"header\", quick, creative, technical " +
	"FROM teams t JOIN players p ON t.team_id = p.team_id " +
	"WHERE t.team_id = $1"

const GetSimulableMatchesStmt = "GetSimulableMatchesStmt"
const GetSimulableMatchesStmtSql = "select match_id, league_id, week_number, match_date, home_team_id, away_team_id, home_goals, away_goals from matches where match_date > now() and match_date < now()+interval '15 minutes' and played is false order by league_id"

const GetUserIdByEmailStmt = "GetUserIdByEmailStmt"
const GetUserIdByEmailStmtSql = "SELECT user_id FROM users WHERE email = $1"

const GetUserStmt = "GetUserStmt"
const GetUserStmtSql = "SELECT u.name userName, u.country, u.provider, u.email, t.team_id, t.name as teamName, " +
	"t.league_id from users u left join teams t on (u.user_id = t.owner_id) where u.user_id = $1"

const InsertTeamToClassificationStmtSql = "INSERT INTO classification (league_id, team_id) values ($1, $2)"

const InsertMatch = "INSERT INTO matches (league_id, week_number, match_date, home_team_id, away_team_id) " +
	"VALUES ($1, $2, $3, $4, $5)"
const InsertNewPlayerStmtSql = "INSERT INTO players (team_id, first_name, last_name, birth_day, " +
	"goal_keeper, defense, \"lateral\", pass, midfield, dribbling, shot, header, quick," +
	" creative, technical) values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)"

const InsertDefaultLineupPositionsStmt = "InsertDefaultLineupPositionsStmt"
const InsertDefaultLineupPositionsStmtSql = "INSERT INTO default_lineup_positions (team_id, player_id, position) values ($1, $2, $3)"

const InsertMatchEventStmt = "InsertMatchEventStmt"
const InsertMatchEventStmtSql = "INSERT INTO match_events (match_id, event_type, team_owner, other_team, minute, team_players, other_team_players) VALUES ($1, $2, $3, $4, $5, $6, $7)"

const RegisterUserStmt = "RegisterUserStmt"
const RegisterUserStmtSql = "INSERT INTO users (name, provider, email, " +
	"country) VALUES ($1, $2, $3, $4) returning user_id"
const UnAssignPlayersStmtSql = "UPDATE players SET team_id = NULL WHERE team_id = $1"
const UpdateTeamDetailsStmtSql = "UPDATE teams SET name = $1, owner_id = $2 WHERE team_id = $3	"

const UpdateTeamClassificationStmt = "UpdateTeamClassificationStmt"
const UpdateTeamClassificationStmtSql = "UPDATE classification set matches = matches+$1, points = points+$2, wins = wins+$3, " +
	"losses = losses+$4, draws = draws+$5, goals_scored = goals_scored+$6, goals_received = goals_received+$7 " +
	"where league_id = $8 and team_id = $9"

const UpdateMatchStmt = "UpdateMatchStmt"
const UpdateMatchStmtSql = "UPDATE matches SET home_goals = $1, away_goals = $2, played = $3 WHERE match_id = $4"

func GetRandomName(dbCtx *DatabaseContext) (string, string, error) {
	stmtNames, err := dbCtx.Db.Prepare("select name from names_es order by random() limit 1")
	stmtLastNames, err := dbCtx.Db.Prepare("select lastname from lastnames_es order by random() limit 1")
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
	stmt := dbCtx.PreparedStmts[GetLineupsForMatchStmt]
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

	rows, err := stmt.Query((*teamIds)[0], (*teamIds)[1], matchId)
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
	stmt := dbCtx.PreparedStmts[GetSimulableMatchesStmt]
	matches := make([]WeekMatch, 0)

	var matchId int64
	var divisionId int64
	var weekNumber int
	var matchDate time.Time
	var home int64
	var away int64
	var homeGoals int
	var awayGoals int

	rows, err := stmt.Query()
	defer rows.Close()
	if err != nil {
		log.Errorf("error getting matches for simulation: %s", err)
		return nil, err
	}
	for rows.Next() {
		err := rows.Scan(&matchId, &divisionId, &weekNumber, &matchDate, &home, &away, &homeGoals, &awayGoals)
		if err != nil {
			log.Errorf("error getting match for simulation: %s", err)
		} else {
			matches = append(matches, WeekMatch{
				MatchId:    matchId,
				DivisionId: divisionId,
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
	stmt := dbCtx.PreparedStmts[GetUserStmt]
	var name string
	var country string
	var provider string
	var email string
	var teamId string
	var teamName string
	var leagueId string
	err := stmt.QueryRow(userId).Scan(&name, &country, &provider, &email, &teamId, &teamName, &leagueId)
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
	stmt := dbCtx.PreparedStmts[GetUserIdByEmailStmt]
	err := stmt.QueryRow(&email).Scan(&userId)
	return userId, err
}

func GetTeamByOwner(userId int64, dbCtx *DatabaseContext) (*Team, error) {
	var teamId int64
	var country string
	var leagueId int64
	var name string
	stmt := dbCtx.PreparedStmts[GetTeamDetailsStmtByOwner]
	err := stmt.QueryRow(userId).Scan(&teamId, &country, &leagueId, &name)
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

func GetDivisionDetails(dbCtx *DatabaseContext, leagueId int64) error {
	_, err := GetDivisionName(dbCtx, leagueId)
	if err != nil {
		return err
	}

	return nil
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

func GetPlayersFromTeamId(dbCtx *DatabaseContext, teamId int64) (*[]*Player, error) {
	stmt := dbCtx.PreparedStmts[GetPlayersFromTeamStmt]
	rows, err := stmt.Query(teamId)
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
			_, err := stmt.Exec(match.DivisionId, match.WeekNumber, match.MatchDate, match.Home, match.Away)
			if err != nil {
				log.Errorf("error inserting match: %s", err)
			}
		}
	}
	return nil
}

func InsertLineup(dbCtx *DatabaseContext, teamId int64, lineup map[int][]int64, asDefault bool) error {
	insertStmt, errInsert := dbCtx.Db.Prepare(InsertDefaultLineupPositionsStmtSql)
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
	defer insertStmt.Close()
	return nil
}

func InsertTeamsToClassification(dbCtx *DatabaseContext, leagueId int64, teams []int64) {
	stmt, _ := dbCtx.Db.Prepare(InsertTeamToClassificationStmtSql)
	defer stmt.Close()
	for _, t := range teams {
		stmt.Exec(leagueId, t)
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

func UpdateClassification(dbCtx *DatabaseContext, leagueId int64) {

}
