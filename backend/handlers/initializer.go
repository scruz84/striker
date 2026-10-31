package handlers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/scruz84/striker/common"
	"github.com/scruz84/striker/engine"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"net/http"
	"strconv"
)

type ActionResult struct {
	Result string `json:"result"`
}

// Initialize wipes and reseeds the entire game (see InitializeGame) - destructive to every
// team any real user has built. Served at POST /v1/admin/initialize, so the JWT and RequireAdmin
// middleware of the admin group have already authenticated an admin by the time it runs; it does
// no identity check of its own. An empty database is bootstrapped with the --initialize CLI
// instead (docker compose --profile init run --rm init).
func Initialize(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Infof("[goroutine - %d] - Starting to initialize the database", common.CurGoroutineID())

		if err := InitializeGame(dbCtx); err != nil {
			log.Errorf("error initializing game: %s", err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		c.JSON(http.StatusOK, &ActionResult{"Initialized!"})
	}
}

// teamPlayerDependentTables lists, in dependency order, every non-user-scoped table that
// references teams/players/matches/cup_ties/friendly_matches/friendly_lineup without ON DELETE
// CASCADE -
// each must be cleared before whatever it references, or that delete fails on a foreign key
// violation (see deleteTables). Returned fresh on every call (rather than a shared package-level
// slice) so InitializeGame/RecreateLeaguesKeepingUsers can each safely append their own extra
// table names onto the result without risking one call's append corrupting a shared backing
// array.
//
// This one list is deliberately shared by both callers so they can't drift out of sync again the
// way they did for issue #7 and issue #26: both independently hit the same "friendly_matches
// (or friendly_lineup/default_lineup_substitutions) still referenced by teams" violation after a
// later feature added a new dependent table that only one of the two hand-maintained lists ever
// got updated for.
func teamPlayerDependentTables() []string {
	return []string{
		"season_history",
		"cup_tie_events",
		"cup_tie_lineup_positions",
		"cup_tie_lineup",
		"cup_ties",
		"cups",
		"team_transactions",
		"transfer_offer_log",
		"transfer_offers",
		"transfer_listings",
		"loan_listings",
		"player_loans",
		"player_unavailability",
		"lineup_positions",
		"lineup",
		"default_lineup_positions",
		"default_lineup_substitutions",
		"match_instants",
		"match_events",
		"matches",
		"classification",
		"friendly_lineup_positions",
		"friendly_lineup",
		"friendly_events",
		"friendly_instants",
		"friendly_matches",
		"friendly_challenges",
		"players",
	}
}

func InitializeGame(dbCtx *common.DatabaseContext) error {
	log.Info("initialize: wiping existing game data")
	tables := append([]string{"notifications"}, teamPlayerDependentTables()...)
	tables = append(tables, "teams", "leagues", "users", "countries", "season")
	if err := deleteTables(dbCtx, tables); err != nil {
		return err
	}

	log.Info("initialize: creating season")
	createSeason(dbCtx)

	log.Info("initialize: creating countries")
	countries := createCountriesList(dbCtx, common.SeededCountries())
	// See engine.ResetCurrentSeasonCache - without this, a season number cached from a previous
	// run (most plausible with an external Redis instance kept across a database reset) would
	// silently outlive the reset, and GenerateMatches below would tag fresh fixtures with it.
	// Each country's current_season column defaults to 1 at creation (see
	// init75_season_per_country.sql), matching the 1 reset into the cache here.
	for _, country := range countries {
		engine.ResetCurrentSeasonCache(dbCtx, country, 1)
	}

	log.Info("initialize: creating leagues, teams, and players - this is the slow part, one country at a time")
	createTeams(dbCtx, countries, common.GetWorldConfig(dbCtx))

	log.Info("initialize: generating this season's match fixtures")
	engine.GenerateMatches(dbCtx)

	log.Info("initialize: starting this season's cups")
	engine.StartCupsForNewSeason(dbCtx, 1)

	log.Info("initialize: done")
	return nil
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
		activeCountries = common.SeededCountries()
	}

	users, err := common.ListUsersWithCountry(dbCtx)
	if err != nil {
		return err
	}

	log.Info("recreate-leagues: wiping existing leagues/teams/players/matches")
	// See InitializeGame's teamPlayerDependentTables doc comment - this is exactly the foreign
	// key violation that used to abort the "teams" delete partway through, silently leaving a
	// half-wiped world (issue #7, and again as issue #26 once newer tables joined the dependency
	// chain without this list being updated to match).
	tables := append(teamPlayerDependentTables(), "teams", "leagues")
	// countries is NOT wiped here - users.country references it (ON DELETE unset), and users are
	// kept, so the existing country rows must stay in place; activeCountries (read above, before
	// any of this deletion) is reused as-is for createTeams below.
	if err := deleteTables(dbCtx, tables); err != nil {
		return err
	}

	if err := common.ResetCurrentSeason(dbCtx); err != nil {
		return err
	}
	// See engine.ResetCurrentSeasonCache and InitializeGame's own call to it above.
	for _, country := range activeCountries {
		engine.ResetCurrentSeasonCache(dbCtx, country, 1)
	}

	log.Info("recreate-leagues: creating leagues, teams, and players - this is the slow part, one country at a time")
	createTeams(dbCtx, activeCountries, common.GetWorldConfig(dbCtx))

	log.Info("recreate-leagues: generating this season's match fixtures")
	engine.GenerateMatches(dbCtx)
	log.Info("recreate-leagues: starting this season's cups")
	engine.StartCupsForNewSeason(dbCtx, 1)

	log.Infof("recreate-leagues: assigning fresh teams to %d existing user(s)", len(users))
	for _, u := range users {
		if _, err := common.AssignTeam(dbCtx, u.UserId, u.Country); err != nil {
			log.Errorf("recreate-leagues: failed to assign a fresh team to user %d (%s): %s", u.UserId, u.Country, err)
		}
	}
	log.Info("recreate-leagues: done")
	return nil
}

// randomSeededCountry picks one of common.SeededCountries at random - used to assign a new
// Google sign-in (see DoExternalLogin) to one of the game's countries, since there's no sign-up
// step where a new user picks one themselves.
func randomSeededCountry() string {
	seeded := common.SeededCountries()
	return seeded[rand.Intn(len(seeded))]
}

// createCountriesList inserts each of the given country codes into the countries table,
// returning the ones that succeeded - used both for a full init (common.SeededCountries) and a
// leagues recreate that keeps whichever countries were already active (see
// RecreateLeaguesKeepingUsers).
func createCountriesList(dbCtx *common.DatabaseContext, codes []string) []string {
	log.Infof("initializing %d countries: %v", len(codes), codes)
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

// createSeason inserts the singleton `season` config row, letting every column take its own
// default - current_season no longer lives here (see init75_season_per_country.sql); each
// country gets its own via its own `current_season DEFAULT 1` column, set the moment
// createCountriesList inserts it.
func createSeason(dbCtx *common.DatabaseContext) {
	insertStmt, _ := dbCtx.Db.Prepare("INSERT INTO season DEFAULT VALUES")
	insertStmt.Exec()
	defer insertStmt.Close()
}

func createLeague(dbCtx *common.DatabaseContext, country string, name string, tier int, groups int) []int64 {
	log.Infof("initializing country %s league %s tier %d with %d group(s)", country, name, tier, groups)

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
	for i, country := range countries {
		log.Infof("creating leagues and teams for country %s (%d/%d)", country, i+1, len(countries))
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
		"INSERT INTO teams (country, league_id, name, stadium_name, stadium_capacity, ai_level) values ($1, $2, $3, $4, $5, $6) returning team_id")
	teamName := common.RandomTeamName(country)
	stadiumName := common.RandomStadiumName(country)
	if errInsert != nil {
		log.Printf("error inserting new teams %s", errInsert)
	}
	capacity := common.DefaultStadiumCapacity
	// Every team starts unowned (see common.AssignTeam for how a human later claims one), so it
	// gets an AI level right away - see common.RandomAiLevel.
	aiLevel := common.RandomAiLevel(tier)
	_ = insertStmt.QueryRow(&country, &leagueId, &teamName, &stadiumName, &capacity, &aiLevel).Scan(&teamId)
	common.CreatePlayers(dbCtx, teamId, country)
	if err := common.GenerateTrainerCandidates(dbCtx, country, common.TrainerCandidatesPerTeam); err != nil {
		log.Errorf("error generating trainer candidates for team %d: %s", teamId, err)
	}
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

// deleteTable wipes one table, returning an error rather than just logging it - a caller
// deleting several FK-related tables in dependency order needs to know a delete failed (e.g. a
// referencing table missing from the list) instead of silently ploughing on into a half-wiped
// state, which is exactly how a missing entry here previously went unnoticed.
func deleteTable(dbCtx *common.DatabaseContext, tableName string) error {
	log.Tracef("deleting table %s", tableName)
	stmt, err := dbCtx.Db.Prepare("DELETE FROM " + tableName)
	if err != nil {
		return fmt.Errorf("preparing delete for table %s: %w", tableName, err)
	}
	defer stmt.Close()
	result, err := stmt.Exec()
	if err != nil {
		return fmt.Errorf("deleting from table %s: %w", tableName, err)
	}
	rows, _ := result.RowsAffected()
	log.Printf("removed %d tuples from %s", rows, tableName)
	return nil
}

// deleteTables wipes each table in order, stopping at the first failure - order matters here
// since a table with a foreign key to one later in the list must be cleared first.
func deleteTables(dbCtx *common.DatabaseContext, tableNames []string) error {
	for _, tableName := range tableNames {
		if err := deleteTable(dbCtx, tableName); err != nil {
			return err
		}
	}
	return nil
}

func InitializeDatabase() {
	dbCtx, err := common.CreateDatabaseContext()
	if err != nil {
		log.Panicf("Error opening database: %s", err)
	}
	if err := common.EnsureSchema(dbCtx); err != nil {
		log.Panicf("Error creating schema: %s", err)
	}
	if err := InitializeGame(dbCtx); err != nil {
		log.Panicf("Error initializing game: %s", err)
	}
}
