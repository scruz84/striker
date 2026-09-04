package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/scruz84/striker/common"
	"github.com/scruz84/striker/engine"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

type ActionResult struct {
	Result string `json:"result"`
}

// Initialize wipes and reseeds the entire game (see InitializeGame) - destructive to every
// team any real user has built. It stays reachable with no login at all only for a genuinely
// fresh deployment (zero registered users: there's nobody to be an admin for yet, and nothing
// of value to destroy) - the moment a first user exists, this requires an authenticated admin,
// checked here rather than via a route-group middleware so the bootstrap case can still work
// with nobody logged in.
func Initialize(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		userCount, err := common.CountUsers(dbCtx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		if userCount > 0 {
			userId := GetUserIdFromCookie(c)
			isAdmin := userId != nil
			if isAdmin {
				isAdmin, err = common.IsUserAdmin(dbCtx, *userId)
				isAdmin = isAdmin && err == nil
			}
			if !isAdmin {
				c.JSON(http.StatusForbidden, GetError(0, "admin access required"))
				return
			}
		}

		log.Infof("[goroutine - %d] - Starting to initialize the database", common.CurGoroutineID())

		InitializeGame(dbCtx)

		c.JSON(http.StatusOK, &ActionResult{"Initialized!"})
	}
}

func InitializeGame(dbCtx *common.DatabaseContext) {
	//remove content - tables added after the original set (cups/finances/transfers) reference
	//teams/players without ON DELETE CASCADE, so they must be cleared first or the later
	//deleteTable(dbCtx, "teams") call fails on a foreign key violation.
	deleteTable(dbCtx, "notifications")
	deleteTable(dbCtx, "cup_tie_events")
	deleteTable(dbCtx, "cup_ties")
	deleteTable(dbCtx, "cups")
	deleteTable(dbCtx, "team_transactions")
	deleteTable(dbCtx, "transfer_offers")
	deleteTable(dbCtx, "transfer_listings")
	deleteTable(dbCtx, "loan_listings")
	deleteTable(dbCtx, "player_loans")
	deleteTable(dbCtx, "player_unavailability")
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

	//create the first season's cups
	engine.StartCupsForNewSeason(dbCtx, 1)
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
	for _, country := range countries {
		//create leagues - tier 1 is the top division, each tier below it splits into more
		//groups; see engine.ComputePromotionRelegation for how teams move between them. Tier is
		//threaded through so each team's starting stadium can be scaled by it - see createTeam.
		createTeamsForTier(dbCtx, country, createLeague(dbCtx, country, "I", 1, 1), 1)
		createTeamsForTier(dbCtx, country, createLeague(dbCtx, country, "II", 2, 4), 2)
		createTeamsForTier(dbCtx, country, createLeague(dbCtx, country, "III", 3, 16), 3)
		//createTeamsForTier(dbCtx, country, createLeague(dbCtx, country, "IV", 4, 64), 4)
		//createTeamsForTier(dbCtx, country, createLeague(dbCtx, country, "V", 5, 256), 5)
		//createTeamsForTier(dbCtx, country, createLeague(dbCtx, country, "VI", 6, 512), 6)
	}
}

func createTeamsForTier(dbCtx *common.DatabaseContext, country string, leagueIds []int64, tier int) {
	for _, leagueId := range leagueIds {
		for i := 0; i < common.TeamsPerLeague; i++ {
			createTeam(dbCtx, country, leagueId, tier)
		}
		createClassification(dbCtx, leagueId)
	}
}

func createTeam(dbCtx *common.DatabaseContext, country string, leagueId int64, tier int) {
	teamId := int64(-1)
	insertStmt, errInsert := dbCtx.Db.Prepare(
		"INSERT INTO teams (country, league_id, name, stadium_capacity) values ($1, $2, $3, $4) returning team_id")
	name, _, _ := common.GetRandomName(dbCtx)
	teamName := name + " Team"
	if errInsert != nil {
		log.Printf("error inserting new teams %s", errInsert)
	}
	capacity := common.RandomStadiumCapacity(tier, rand.New(rand.NewSource(time.Now().UnixNano())))
	_ = insertStmt.QueryRow(&country, &leagueId, &teamName, &capacity).Scan(&teamId)
	common.CreatePlayers(dbCtx, teamId)
	players, err := common.GetPlayersFromTeamId(dbCtx, teamId)
	if err != nil {
		log.Errorf("error fetching newly created squad for team %d: %s", teamId, err)
	} else if err := common.InsertLineup(dbCtx, teamId, common.SelectBestLineup(*players), true); err != nil {
		log.Errorf("error seeding default lineup for team %d: %s", teamId, err)
	}
	defer insertStmt.Close()
}

func createClassification(dbCtx *common.DatabaseContext, leagueId int64) {
	teamIds := common.GetDivisionTeams(dbCtx, leagueId)
	common.InsertTeamsToClassification(dbCtx, leagueId, 1, teamIds)
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
