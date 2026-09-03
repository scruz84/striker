package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/scruz84/striker/common"
	"github.com/scruz84/striker/engine"
	log "github.com/sirupsen/logrus"
	"net/http"
	"strconv"
)

type ActionResult struct {
	Result string `json:"result"`
}

func Initialize(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Infof("[goroutine - %d] - Starting to initialize the database", common.CurGoroutineID())

		InitializeGame(dbCtx)

		c.JSON(http.StatusOK, &ActionResult{"Initialized!"})
	}
}

func InitializeGame(dbCtx *common.DatabaseContext) {
	//remove content
	deleteTable(dbCtx, "lineup_positions")
	deleteTable(dbCtx, "lineup")
	deleteTable(dbCtx, "default_lineup_positions")
	deleteTable(dbCtx, "match_instants")
	deleteTable(dbCtx, "match_events")
	deleteTable(dbCtx, "matches")
	deleteTable(dbCtx, "classification")
	deleteTable(dbCtx, "players")
	deleteTable(dbCtx, "teams")
	deleteTable(dbCtx, "leagues")
	deleteTable(dbCtx, "users")
	deleteTable(dbCtx, "countries")
	deleteTable(dbCtx, "season")

	//season
	createSeason(dbCtx)

	//countries
	countries := createCountries(dbCtx)

	//create teams and leagues
	createTeams(dbCtx, countries)

	//create matches
	engine.GenerateMatches(dbCtx)
}

func createCountries(dbCtx *common.DatabaseContext) []string {
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

func createSeason(dbCtx *common.DatabaseContext) {
	insertStmt, _ := dbCtx.Db.Prepare("INSERT INTO season (current_season) values(1)")
	insertStmt.Exec()
	defer insertStmt.Close()
}

func createLeague(dbCtx *common.DatabaseContext, country string, name string, tier int, groups int) []int64 {
	log.Debugf("initializing country %s league %s tier %d with groups %d", country, name, tier, groups)

	leagues := make([]int64, 0)
	insertStmt, errInsert := dbCtx.Db.Prepare("INSERT INTO leagues (country, group_number, name, tier) " +
		"values ($1, $2, $3, $4) returning league_id")
	if errInsert != nil {
		log.Printf("error inserting new league %s", errInsert)
	} else {
		for g := 1; g <= groups; g++ {
			leagueId := int64(-1)
			tName := name + "." + strconv.Itoa(g)
			_ = insertStmt.QueryRow(&country, &g, &tName, &tier).Scan(&leagueId)
			leagues = append(leagues, leagueId)
		}
	}
	defer insertStmt.Close()
	return leagues
}

func createTeams(dbCtx *common.DatabaseContext, countries []string) {
	log.Debug("initializing teams")
	leagues := make([]int64, 0)
	for _, country := range countries {
		//create leagues - tier 1 is the top division, each tier below it splits into more
		//groups; see engine.ComputePromotionRelegation for how teams move between them.
		leagues = append(leagues, createLeague(dbCtx, country, "I", 1, 1)...)
		leagues = append(leagues, createLeague(dbCtx, country, "II", 2, 4)...)
		leagues = append(leagues, createLeague(dbCtx, country, "III", 3, 16)...)
		//leagues = append(leagues, createLeague(dbCtx, country, "IV", 4, 64)...)
		//leagues = append(leagues, createLeague(dbCtx, country, "V", 5, 256)...)
		//leagues = append(leagues, createLeague(dbCtx, country, "VI", 6, 512)...)

		//create the teams
		for _, leagueId := range leagues {
			for i := 0; i < common.TeamsPerLeague; i++ {
				createTeam(dbCtx, country, leagueId)
			}
			createClassification(dbCtx, leagueId)
		}
	}
}

func createTeam(dbCtx *common.DatabaseContext, country string, leagueId int64) {
	teamId := int64(-1)
	insertStmt, errInsert := dbCtx.Db.Prepare("INSERT INTO teams (country, league_id, name) values ($1, $2, $3) returning team_id")
	name, _, _ := common.GetRandomName(dbCtx)
	teamName := name + " Team"
	if errInsert != nil {
		log.Printf("error inserting new teams %s", errInsert)
	}
	_ = insertStmt.QueryRow(&country, &leagueId, &teamName).Scan(&teamId)
	common.CreatePlayers(dbCtx, teamId)
	lineup := generateLineup(dbCtx, teamId)
	_ = common.InsertLineup(dbCtx, teamId, *lineup, true)
	defer insertStmt.Close()
}

func createClassification(dbCtx *common.DatabaseContext, leagueId int64) {
	teamIds := common.GetDivisionTeams(dbCtx, leagueId)
	common.InsertTeamsToClassification(dbCtx, leagueId, 1, teamIds)
}

func generateLineup(dbCtx *common.DatabaseContext, teamId int64) *map[int][]int64 {
	players, err := common.GetPlayersFromTeamId(dbCtx, teamId)
	if err != nil {
		log.Errorf("error retrieving players from team %d: %s", teamId, err)
		return nil
	}

	lineup := make(map[int][]int64)
	lineup[common.GOALKEEPER] = make([]int64, 0)
	lineup[common.LATERAL_DEFENSE_LEFT] = make([]int64, 0)
	lineup[common.LATERAL_DEFENSE_RIGHT] = make([]int64, 0)
	lineup[common.DEFENSE] = make([]int64, 0)
	lineup[common.LATERAL_MIDFIELD_LEFT] = make([]int64, 0)
	lineup[common.LATERAL_MIDFIELD_RIGHT] = make([]int64, 0)
	lineup[common.MIDFIELD] = make([]int64, 0)
	lineup[common.LATERAL_STRIKER_LEFT] = make([]int64, 0)
	lineup[common.LATERAL_STRIKER_RIGHT] = make([]int64, 0)
	lineup[common.STRIKER] = make([]int64, 0)

	var playerSelectedIdx int
	playerSelectedIdx = getBetterPlayer(players, &lineup, common.GOALKEEPER, func(player common.Player) int { return player.GoalKeeper })
	players = common.RemoveIndexFromSlice(players, playerSelectedIdx)
	playerSelectedIdx = getBetterPlayer(players, &lineup, common.LATERAL_DEFENSE_LEFT, func(player common.Player) int { return common.GetLateralDefenseValue(player) })
	players = common.RemoveIndexFromSlice(players, playerSelectedIdx)
	playerSelectedIdx = getBetterPlayer(players, &lineup, common.DEFENSE, func(player common.Player) int { return player.Defense })
	players = common.RemoveIndexFromSlice(players, playerSelectedIdx)
	playerSelectedIdx = getBetterPlayer(players, &lineup, common.DEFENSE, func(player common.Player) int { return player.Defense })
	players = common.RemoveIndexFromSlice(players, playerSelectedIdx)
	playerSelectedIdx = getBetterPlayer(players, &lineup, common.LATERAL_DEFENSE_RIGHT, func(player common.Player) int { return common.GetLateralDefenseValue(player) })
	players = common.RemoveIndexFromSlice(players, playerSelectedIdx)
	playerSelectedIdx = getBetterPlayer(players, &lineup, common.LATERAL_MIDFIELD_LEFT, func(player common.Player) int { return common.GetLateralMidfieldValue(player) })
	players = common.RemoveIndexFromSlice(players, playerSelectedIdx)
	playerSelectedIdx = getBetterPlayer(players, &lineup, common.MIDFIELD, func(player common.Player) int { return common.GetMidfieldValue(player) })
	players = common.RemoveIndexFromSlice(players, playerSelectedIdx)
	playerSelectedIdx = getBetterPlayer(players, &lineup, common.MIDFIELD, func(player common.Player) int { return common.GetMidfieldValue(player) })
	players = common.RemoveIndexFromSlice(players, playerSelectedIdx)
	playerSelectedIdx = getBetterPlayer(players, &lineup, common.LATERAL_MIDFIELD_RIGHT, func(player common.Player) int { return common.GetLateralMidfieldValue(player) })
	players = common.RemoveIndexFromSlice(players, playerSelectedIdx)

	/*	playerSelectedIdx = getBetterPlayer(players, &lineup, common.LATERAL_STRIKER_LEFT, func(player common.Player) int { return common.GetLateralStrikerValue(player) })
		players = common.RemoveIndexFromSlice(players, playerSelectedIdx)
		playerSelectedIdx = getBetterPlayer(players, &lineup, common.LATERAL_STRIKER_RIGHT, func(player common.Player) int { return common.GetLateralStrikerValue(player) })
		players = common.RemoveIndexFromSlice(players, playerSelectedIdx)
	*/
	playerSelectedIdx = getBetterPlayer(players, &lineup, common.STRIKER, func(player common.Player) int { return common.GetStrikerValue(player) })
	players = common.RemoveIndexFromSlice(players, playerSelectedIdx)
	playerSelectedIdx = getBetterPlayer(players, &lineup, common.STRIKER, func(player common.Player) int { return common.GetStrikerValue(player) })
	players = common.RemoveIndexFromSlice(players, playerSelectedIdx)
	return &lineup
}

func getBetterPlayer(players *[]*common.Player, lineup *map[int][]int64, key int, keyValue func(common.Player) int) int {
	currentLevel := 0
	currentPlayerIdx := -1
	for idx, player := range *players {
		if currentLevel < keyValue(*player) {
			currentPlayerIdx = idx
			currentLevel = keyValue(*player)
		}
	}
	(*lineup)[key] = append((*lineup)[key], (*players)[currentPlayerIdx].Id)
	//players = common.RemoveIndexFromSlice(players, currentPlayerIdx)
	//players = newPlayers
	return currentPlayerIdx
}

func deleteTable(dbCtx *common.DatabaseContext, tableName string) {
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

func InitializeDatabase() {
	dbCtx, err := common.CreateDatabaseContext()
	if err != nil {
		log.Panicf("Error opening database: %s", err)
	}
	InitializeGame(dbCtx)
}
