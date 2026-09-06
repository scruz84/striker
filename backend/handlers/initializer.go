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
	deleteTable(dbCtx, "season_history")
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
	countries := createCountriesList(dbCtx, seededCountries)

	//create teams and leagues
	createTeams(dbCtx, countries, common.GetWorldConfig(dbCtx))

	//create matches
	engine.GenerateMatches(dbCtx)

	//create the first season's cups
	engine.StartCupsForNewSeason(dbCtx, 1)
}

// RecreateLeaguesKeepingUsers wipes and regenerates every league/team/match (using the current
// world config - see common.GetWorldConfig - for whichever countries are currently active) the
// same way InitializeGame does, but preserves registered logins: each existing user is handed a
// fresh team afterward via common.AssignTeam, in their existing country. Team history, trophies
// and squads are still lost - only the login/account itself survives. Unlike InitializeGame this
// updates the season row in place (see common.ResetCurrentSeason) rather than deleting it, so an
// admin-configured global match schedule (see common.GetMatchSchedule) isn't silently reset to
// its default.
func RecreateLeaguesKeepingUsers(dbCtx *common.DatabaseContext) error {
	activeCountries, err := common.GetActiveCountries(dbCtx)
	if err != nil {
		return err
	}
	if len(activeCountries) == 0 {
		activeCountries = seededCountries
	}

	users, err := common.ListUsersWithCountry(dbCtx)
	if err != nil {
		return err
	}

	deleteTable(dbCtx, "season_history")
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
	// countries is NOT wiped here - users.country references it (ON DELETE unset), and users are
	// kept, so the existing country rows must stay in place; activeCountries (read above, before
	// any of this deletion) is reused as-is for createTeams below.

	if err := common.ResetCurrentSeason(dbCtx); err != nil {
		return err
	}

	createTeams(dbCtx, activeCountries, common.GetWorldConfig(dbCtx))

	engine.GenerateMatches(dbCtx)
	engine.StartCupsForNewSeason(dbCtx, 1)

	for _, u := range users {
		if _, err := common.AssignTeam(dbCtx, u.UserId, u.Country); err != nil {
			log.Errorf("recreate-leagues: failed to assign a fresh team to user %d (%s): %s", u.UserId, u.Country, err)
		}
	}
	return nil
}

// seededCountries lists every country the game world is generated for - each needs a matching
// names_<code>/lastnames_<code> table pair (see common.GetRandomName and
// resources/sql/init19_multi_country_names.sql).
var seededCountries = []string{"es", "en", "fr", "de"}

// randomSeededCountry picks one of seededCountries at random - used to assign a new Google
// sign-in (see DoExternalLogin) to one of the game's countries, since there's no sign-up step
// where a new user picks one themselves.
func randomSeededCountry() string {
	return seededCountries[rand.Intn(len(seededCountries))]
}

// createCountriesList inserts each of the given country codes into the countries table,
// returning the ones that succeeded - used both for a full init (seededCountries) and a
// leagues recreate that keeps whichever countries were already active (see
// RecreateLeaguesKeepingUsers).
func createCountriesList(dbCtx *common.DatabaseContext, codes []string) []string {
	log.Debug("initializing countries")
	insertStmt, errInsert := dbCtx.Db.Prepare("INSERT INTO countries (name) values ($1) returning country_id")
	if errInsert != nil {
		log.Printf("error inserting new countries %s", errInsert)
		return nil
	}
	defer insertStmt.Close()

	countries := make([]string, 0, len(codes))
	for _, country := range codes {
		countryId := int64(-1)
		if err := insertStmt.QueryRow(country).Scan(&countryId); err != nil {
			log.Printf("error inserting country %s: %s", country, err)
			continue
		}
		countries = append(countries, country)
	}
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

// createTeams creates leagues and teams for every country, shaped by tiers (see
// common.GetWorldConfig) - tier 1 is the top division, each tier below it splits into more
// groups; see engine.ComputePromotionRelegation for how teams move between them.
func createTeams(dbCtx *common.DatabaseContext, countries []string, tiers []common.TierConfig) {
	log.Debug("initializing teams")
	for _, country := range countries {
		createCountryLeagues(dbCtx, country, tiers)
	}
}

// createCountryLeagues creates one country's leagues and teams, shaped by tiers - the
// incremental counterpart to createTeams' full loop, used to activate a single new country
// mid-season without touching any other country (see handlers.AdminAddCountryHandler).
func createCountryLeagues(dbCtx *common.DatabaseContext, country string, tiers []common.TierConfig) {
	for _, tierConfig := range tiers {
		leagueIds := createLeague(dbCtx, country, tierConfig.Name, tierConfig.Tier, tierConfig.Groups)
		createTeamsForTier(dbCtx, country, leagueIds, tierConfig.Tier, tierConfig.TeamsPerLeague)
	}
}

func createTeamsForTier(dbCtx *common.DatabaseContext, country string, leagueIds []int64, tier int, teamsPerLeague int) {
	for _, leagueId := range leagueIds {
		for i := 0; i < teamsPerLeague; i++ {
			createTeam(dbCtx, country, leagueId, tier)
		}
		createClassification(dbCtx, leagueId)
	}
}

func createTeam(dbCtx *common.DatabaseContext, country string, leagueId int64, tier int) {
	teamId := int64(-1)
	insertStmt, errInsert := dbCtx.Db.Prepare(
		"INSERT INTO teams (country, league_id, name, stadium_capacity) values ($1, $2, $3, $4) returning team_id")
	teamName := common.RandomTeamName(country)
	if errInsert != nil {
		log.Printf("error inserting new teams %s", errInsert)
	}
	capacity := common.RandomStadiumCapacity(tier, rand.New(rand.NewSource(time.Now().UnixNano())))
	_ = insertStmt.QueryRow(&country, &leagueId, &teamName, &capacity).Scan(&teamId)
	common.CreatePlayers(dbCtx, teamId, country)
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
	if err := common.EnsureSchema(dbCtx); err != nil {
		log.Panicf("Error creating schema: %s", err)
	}
	InitializeGame(dbCtx)
}
