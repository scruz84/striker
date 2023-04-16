package handlers

import (
	"footmanager/fmcommon"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"net/http"
	"strconv"
)

type ActionResult struct {
	Result string `json:"result"`
}

func Initialize(dbCtx *fmcommon.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Infof("[goroutine - %d] - Starting to initialize the database", fmcommon.CurGoroutineID())

		//remove content
		deleteTable(dbCtx, "lineup_positions")
		deleteTable(dbCtx, "lineup")
		deleteTable(dbCtx, "default_lineup_positions")
		deleteTable(dbCtx, "match_events")
		deleteTable(dbCtx, "matches")
		deleteTable(dbCtx, "classification")
		deleteTable(dbCtx, "players")
		deleteTable(dbCtx, "teams")
		deleteTable(dbCtx, "leagues")
		deleteTable(dbCtx, "users")
		deleteTable(dbCtx, "countries")

		//countries
		countries := createCountries(dbCtx)

		//create teams
		createTeams(dbCtx, countries)

		c.JSON(http.StatusOK, &ActionResult{"Initialized!"})
	}
}

func createCountries(dbCtx *fmcommon.DatabaseContext) []string {
	log.Debug("initializing countries")
	insertStmt, errInsert := dbCtx.Db.Prepare("INSERT INTO countries (name) values ($1) returning country_id")
	countryId := int64(-1)
	countries := make([]string, 0)
	if errInsert != nil {
		log.Printf("error inserting new countries %s", errInsert)
	} else {
		_ = insertStmt.QueryRow("es").Scan(&countryId)
		countries = append(countries, "es")
	}
	defer insertStmt.Close()
	return countries
}

func createLeague(dbCtx *fmcommon.DatabaseContext, country string, name string, groups int) []int64 {
	log.Debug("initializing country %s league %s with groups %d", country, name, groups)

	leagues := make([]int64, 0)
	insertStmt, errInsert := dbCtx.Db.Prepare("INSERT INTO leagues (country, group_number, name) values ($1, $2, $3) returning league_id")
	if errInsert != nil {
		log.Printf("error inserting new league %s", errInsert)
	} else {
		for g := 1; g <= groups; g++ {
			leagueId := int64(-1)
			tName := name + "." + strconv.Itoa(g)
			_ = insertStmt.QueryRow(&country, &g, &tName).Scan(&leagueId)
			leagues = append(leagues, leagueId)
		}
	}
	defer insertStmt.Close()
	return leagues
}

func createTeams(dbCtx *fmcommon.DatabaseContext, countries []string) {
	log.Debug("initializing teams")
	leagues := make([]int64, 0)
	for _, country := range countries {
		//create leagues
		leagues = append(leagues, createLeague(dbCtx, country, "I", 1)...)
		leagues = append(leagues, createLeague(dbCtx, country, "II", 4)...)
		leagues = append(leagues, createLeague(dbCtx, country, "III", 16)...)
		//leagues = append(leagues, createLeague(dbCtx, country, "IV", 64)...)
		//leagues = append(leagues, createLeague(dbCtx, country, "V", 256)...)
		//leagues = append(leagues, createLeague(dbCtx, country, "VI", 512)...)

		//create the teams
		for _, leagueId := range leagues {
			for i := 0; i < fmcommon.TeamsPerLeague; i++ {
				createTeam(dbCtx, country, leagueId)
			}
			createClassification(dbCtx, leagueId)
		}
	}
}

func createTeam(dbCtx *fmcommon.DatabaseContext, country string, leagueId int64) {
	teamId := int64(-1)
	insertStmt, errInsert := dbCtx.Db.Prepare("INSERT INTO teams (country, league_id, name) values ($1, $2, $3) returning team_id")
	name, _, _ := fmcommon.GetRandomName(dbCtx)
	teamName := name + " Team"
	if errInsert != nil {
		log.Printf("error inserting new teams %s", errInsert)
	}
	_ = insertStmt.QueryRow(&country, &leagueId, &teamName).Scan(&teamId)
	fmcommon.CreatePlayers(dbCtx, teamId)
	lineup := generateLineup(dbCtx, teamId)
	_ = fmcommon.InsertLineup(dbCtx, teamId, *lineup, true)
	defer insertStmt.Close()
}

func createClassification(dbCtx *fmcommon.DatabaseContext, leagueId int64) {
	teamIds := fmcommon.GetDivisionTeams(dbCtx, leagueId)
	fmcommon.InsertTeamsToClassification(dbCtx, leagueId, teamIds)
}

func generateLineup(dbCtx *fmcommon.DatabaseContext, teamId int64) *map[int][]int64 {
	players, err := fmcommon.GetPlayersFromTeamId(dbCtx, teamId)
	if err != nil {
		log.Errorf("error retrieving players from team %d: %s", teamId, err)
		return nil
	}

	lineup := make(map[int][]int64)
	lineup[fmcommon.GOALKEEPER] = make([]int64, 0)
	lineup[fmcommon.LATERAL_DEFENSE_LEFT] = make([]int64, 0)
	lineup[fmcommon.LATERAL_DEFENSE_RIGHT] = make([]int64, 0)
	lineup[fmcommon.DEFENSE] = make([]int64, 0)
	lineup[fmcommon.LATERAL_MIDFIELD_LEFT] = make([]int64, 0)
	lineup[fmcommon.LATERAL_MIDFIELD_RIGHT] = make([]int64, 0)
	lineup[fmcommon.MIDFIELD] = make([]int64, 0)
	lineup[fmcommon.LATERAL_STRIKER_LEFT] = make([]int64, 0)
	lineup[fmcommon.LATERAL_STRIKER_RIGHT] = make([]int64, 0)
	lineup[fmcommon.STRIKER] = make([]int64, 0)

	var playerSelectedIdx int
	playerSelectedIdx = getBetterPlayer(players, &lineup, fmcommon.GOALKEEPER, func(player fmcommon.Player) int { return player.GoalKeeper })
	players = fmcommon.RemoveIndexFromSlice(players, playerSelectedIdx)
	playerSelectedIdx = getBetterPlayer(players, &lineup, fmcommon.LATERAL_DEFENSE_LEFT, func(player fmcommon.Player) int { return fmcommon.GetLateralDefenseValue(player) })
	players = fmcommon.RemoveIndexFromSlice(players, playerSelectedIdx)
	playerSelectedIdx = getBetterPlayer(players, &lineup, fmcommon.DEFENSE, func(player fmcommon.Player) int { return player.Defense })
	players = fmcommon.RemoveIndexFromSlice(players, playerSelectedIdx)
	playerSelectedIdx = getBetterPlayer(players, &lineup, fmcommon.DEFENSE, func(player fmcommon.Player) int { return player.Defense })
	players = fmcommon.RemoveIndexFromSlice(players, playerSelectedIdx)
	playerSelectedIdx = getBetterPlayer(players, &lineup, fmcommon.LATERAL_DEFENSE_RIGHT, func(player fmcommon.Player) int { return fmcommon.GetLateralDefenseValue(player) })
	players = fmcommon.RemoveIndexFromSlice(players, playerSelectedIdx)
	playerSelectedIdx = getBetterPlayer(players, &lineup, fmcommon.LATERAL_MIDFIELD_LEFT, func(player fmcommon.Player) int { return fmcommon.GetLateralMidfieldValue(player) })
	players = fmcommon.RemoveIndexFromSlice(players, playerSelectedIdx)
	playerSelectedIdx = getBetterPlayer(players, &lineup, fmcommon.MIDFIELD, func(player fmcommon.Player) int { return fmcommon.GetMidfieldValue(player) })
	players = fmcommon.RemoveIndexFromSlice(players, playerSelectedIdx)
	playerSelectedIdx = getBetterPlayer(players, &lineup, fmcommon.MIDFIELD, func(player fmcommon.Player) int { return fmcommon.GetMidfieldValue(player) })
	players = fmcommon.RemoveIndexFromSlice(players, playerSelectedIdx)
	playerSelectedIdx = getBetterPlayer(players, &lineup, fmcommon.LATERAL_MIDFIELD_RIGHT, func(player fmcommon.Player) int { return fmcommon.GetLateralMidfieldValue(player) })
	players = fmcommon.RemoveIndexFromSlice(players, playerSelectedIdx)

	/*	playerSelectedIdx = getBetterPlayer(players, &lineup, fmcommon.LATERAL_STRIKER_LEFT, func(player fmcommon.Player) int { return fmcommon.GetLateralStrikerValue(player) })
		players = fmcommon.RemoveIndexFromSlice(players, playerSelectedIdx)
		playerSelectedIdx = getBetterPlayer(players, &lineup, fmcommon.LATERAL_STRIKER_RIGHT, func(player fmcommon.Player) int { return fmcommon.GetLateralStrikerValue(player) })
		players = fmcommon.RemoveIndexFromSlice(players, playerSelectedIdx)
	*/
	playerSelectedIdx = getBetterPlayer(players, &lineup, fmcommon.STRIKER, func(player fmcommon.Player) int { return fmcommon.GetStrikerValue(player) })
	players = fmcommon.RemoveIndexFromSlice(players, playerSelectedIdx)
	playerSelectedIdx = getBetterPlayer(players, &lineup, fmcommon.STRIKER, func(player fmcommon.Player) int { return fmcommon.GetStrikerValue(player) })
	players = fmcommon.RemoveIndexFromSlice(players, playerSelectedIdx)
	return &lineup
}

func getBetterPlayer(players *[]*fmcommon.Player, lineup *map[int][]int64, key int, keyValue func(fmcommon.Player) int) int {
	currentLevel := 0
	currentPlayerIdx := -1
	for idx, player := range *players {
		if currentLevel < keyValue(*player) {
			currentPlayerIdx = idx
			currentLevel = keyValue(*player)
		}
	}
	(*lineup)[key] = append((*lineup)[key], (*players)[currentPlayerIdx].Id)
	//players = fmcommon.RemoveIndexFromSlice(players, currentPlayerIdx)
	//players = newPlayers
	return currentPlayerIdx
}

func deleteTable(dbCtx *fmcommon.DatabaseContext, tableName string) {
	log.Tracef("deleting table %s", tableName)
	stmt, err := dbCtx.Db.Prepare("DELETE FROM " + tableName)
	if err != nil {
		log.Printf("error removing data from %s table. %s", tableName, err)
	}
	result, err := stmt.Exec()
	defer stmt.Close()
	if err != nil {
		log.Printf("error removing data from %s table. %s", tableName, err)
	} else {
		rows, _ := result.RowsAffected()
		log.Printf("removed %d tuples from %s", rows, tableName)
	}
}
