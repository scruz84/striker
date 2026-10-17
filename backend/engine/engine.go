package engine

import (
	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"sync"
	"time"
)

// maxConcurrentSimulations bounds how many matches simulate at once, so a burst of
// simulable matches doesn't spawn unbounded goroutines each holding a DB connection.
const maxConcurrentSimulations = 8

// GenerateMatches generates fixtures for every active country that needs them - a standalone
// safety net covering every country at once (see handlers.InitializeGame/
// RecreateLeaguesKeepingUsers, right after a fresh world is seeded), separate from the
// per-country GenerateMatchesForLeagues call each country's own rollover already makes (see
// advanceCountrySeasonIfComplete). Whether a given league actually needs fixtures is decided
// per-league, inside generateMatchesForOneLeague's own lock - there's no single "does anyone
// have upcoming matches" check left to make up front, since that would have to mean something
// different for every country.
func GenerateMatches(dbCtx *common.DatabaseContext) {
	countries, err := common.GetActiveCountries(dbCtx)
	if err != nil {
		log.Errorf("error listing active countries to generate matches: %s", err)
		return
	}
	for _, country := range countries {
		leagueIds, err := common.GetLeagueIdsForCountry(dbCtx, country)
		if err != nil {
			log.Errorf("error listing leagues for %s to generate matches: %s", country, err)
			continue
		}
		if err := GenerateMatchesForLeagues(dbCtx, leagueIds); err != nil {
			log.Errorf("error generating matches for %s: %s", country, err)
		}
	}
}

// GenerateMatchesForLeagues generates the current season's fixtures for exactly the given
// leagues (which must all belong to the same country - every caller already has them scoped
// that way, see advanceCountrySeasonIfComplete, handlers.AdminAddCountryHandler/
// AdminAddDivisionHandler). Needed on its own, separate from GenerateMatches, when a league is
// added mid-season: every other league in the game already having fixtures scheduled would
// otherwise make a single "does anyone need fixtures" check skip straight past the new one.
func GenerateMatchesForLeagues(dbCtx *common.DatabaseContext, leagueIds []int64) error {
	if len(leagueIds) == 0 {
		return nil
	}
	season, err := common.GetCurrentSeasonForLeague(dbCtx, leagueIds[0])
	if err != nil {
		return err
	}
	country, err := common.GetCountryForLeague(dbCtx, leagueIds[0])
	if err != nil {
		return err
	}
	generateMatchesForLeagueIds(dbCtx, leagueIds, country, season)
	return nil
}

// generateMatchesForLeagueIds dates each league's fixtures on its own schedule override if it
// has one (see common.GetLeagueScheduleOverrides), falling back to the app-wide default (see
// common.GetMatchSchedule) otherwise.
func generateMatchesForLeagueIds(dbCtx *common.DatabaseContext, leagueIds []int64, country string, season int) {
	defaultSchedule := common.GetMatchSchedule(dbCtx)
	overrides, err := common.GetLeagueScheduleOverrides(dbCtx)
	if err != nil {
		log.Errorf("error loading per-league schedule overrides, using the default for every league: %s", err)
		overrides = nil
	}

	// Searched from once, before any league's fixtures are inserted: a season closes while its last
	// round can still be dated in the future (matches are simulated 15 minutes ahead), and the new
	// season must start after it. Only earlier seasons count, so a league added mid-season isn't
	// pushed past the end of everyone else's current fixtures. Scoped to country - otherwise this
	// could pick up another, still-ongoing country's own future-dated fixtures as the "latest
	// previous" date (see common.GetLatestMatchDateBeforeSeason's own doc comment).
	latestPrevious, err := common.GetLatestMatchDateBeforeSeason(dbCtx, country, season)
	if err != nil {
		log.Errorf("error finding the previous season's last match date, scheduling from now: %s", err)
	}
	from := FixtureStart(time.Now(), latestPrevious)

	for _, leagueId := range leagueIds {
		schedule := defaultSchedule
		if override, ok := overrides[leagueId]; ok {
			schedule = override
		}
		if err := generateMatchesForOneLeague(dbCtx, leagueId, season, schedule, from); err != nil {
			log.Errorf("error generating matches for league %d: %s", leagueId, err)
		}
	}
}

// generateMatchesForOneLeague generates and inserts leagueId's season fixtures inside one
// transaction, guarded by a non-blocking advisory lock keyed to leagueId - the same
// pg_try_advisory_xact_lock pattern common.SetDefaultLineup already uses for a one-shot
// idempotent write it can't let two callers both perform. Two engine replicas racing to generate
// the same league's fixtures is a real deployment shape, not a hypothetical one (see
// docker-compose.yml, common/schema.go's cross-process migration lock, and the Grafana
// per-replica dashboards) - without this, both could pass a plain "does this league already have
// upcoming matches" read before either commits, and both insert a full duplicate schedule. The
// "already has upcoming matches" check is re-run inside the lock (not just once up front in
// GenerateMatches/GenerateMatchesForLeagues), so a second caller that loses the race still sees
// the winner's just-committed fixtures and correctly no-ops instead of only relying on who
// acquired the lock first.
func generateMatchesForOneLeague(dbCtx *common.DatabaseContext, leagueId int64, season int, schedule common.MatchSchedule, from time.Time) error {
	tx, err := dbCtx.Db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var acquired bool
	if err := tx.QueryRow("SELECT pg_try_advisory_xact_lock($1)", leagueId).Scan(&acquired); err != nil {
		return err
	}
	if !acquired {
		return nil
	}

	var hasUpcoming bool
	if err := tx.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM matches WHERE league_id = $1 AND season = $2 AND match_date > now())",
		leagueId, season).Scan(&hasUpcoming); err != nil {
		return err
	}
	if hasUpcoming {
		return nil
	}

	teams, err := tx.Query(common.GetTeamIdsFromLeagueSql, leagueId)
	if err != nil {
		return err
	}
	teamIds := make([]int64, 0)
	for teams.Next() {
		var teamId int64
		if err := teams.Scan(&teamId); err != nil {
			teams.Close()
			return err
		}
		teamIds = append(teamIds, teamId)
	}
	teams.Close()
	log.Infof("Team ids: %v", teamIds)

	weekMatches := randomWeekMatches2(&leagueId, season, &teamIds, schedule, from)
	if err := common.InsertMatchesTx(tx, weekMatches); err != nil {
		return err
	}
	return tx.Commit()
}

// Algorithm from: https://www.competize.com/blog/hacer-fixture-crear-calendario-liga-excel/
func randomWeekMatches2(leagueId *int64, season int, teamIds *[]int64, schedule common.MatchSchedule, from time.Time) [][]*common.WeekMatch {
	// Here we have a random sort for the teams
	rand.New(rand.NewSource(time.Now().UnixNano()))
	rand.Shuffle(len(*teamIds), func(i, j int) {
		(*teamIds)[i], (*teamIds)[j] = (*teamIds)[j], (*teamIds)[i]
	})

	nWeeks := (len(*teamIds) - 1) * 2
	nMatchesPerWeek := len(*teamIds) / 2
	weeksMatches := make([][]*common.WeekMatch, nWeeks)

	teamIdxForMatchFromWeek := 0
	teamIdxForRestMatchesFromWeek := len(*teamIds) - 2
	latestTeamId := &(*teamIds)[len(*teamIds)-1]
	for weekIdx := 0; weekIdx < nWeeks; weekIdx++ {
		weeksMatches[weekIdx] = make([]*common.WeekMatch, nMatchesPerWeek)
		for weekIndex := 0; weekIndex < nMatchesPerWeek; weekIndex++ {
			idx := weekIdx + 1
			home := (*teamIds)[teamIdxForMatchFromWeek]
			weeksMatches[weekIdx][weekIndex] = &common.WeekMatch{
				DivisionId: *leagueId,
				Season:     season,
				WeekNumber: idx,
				Home:       home,
			}
			if weekIndex == 0 {
				if weekIdx%2 == 0 {
					newAway := *latestTeamId
					weeksMatches[weekIdx][weekIndex].Away = newAway
				} else {
					weeksMatches[weekIdx][weekIndex].Away = (weeksMatches[weekIdx][weekIndex]).Home
					weeksMatches[weekIdx][weekIndex].Home = *latestTeamId
				}
			} else {
				if weekIdx < nWeeks/2 {
					weeksMatches[weekIdx][weekIndex].Away = (*teamIds)[teamIdxForRestMatchesFromWeek]
				} else {
					// second round
					weeksMatches[weekIdx][weekIndex].Away = (weeksMatches[weekIdx][weekIndex]).Home
					weeksMatches[weekIdx][weekIndex].Home = (*teamIds)[teamIdxForRestMatchesFromWeek]
				}
				if teamIdxForRestMatchesFromWeek == 0 {
					teamIdxForRestMatchesFromWeek = len(*teamIds) - 2
				} else {
					teamIdxForRestMatchesFromWeek = (teamIdxForRestMatchesFromWeek - 1) % (len(*teamIds) - 1)
				}
			}
			teamIdxForMatchFromWeek = (teamIdxForMatchFromWeek + 1) % (len(*teamIds) - 1)
		}
	}
	roundDates := ComputeFixtureDates(from, schedule, nWeeks)
	for weekIdx, matches := range weeksMatches {
		for _, match := range matches {
			match.MatchDate = roundDates[weekIdx]
		}
	}
	return weeksMatches
}

// SimulateMatches simulate matches which has a match date 15 minutes prior current time.
func SimulateMatches(dbCtx *common.DatabaseContext) {
	matches, err := common.GetSimulableMatches(dbCtx)
	if err != nil {
		log.Errorf("error getting matches to simulate: %s", err)
		return
	}
	SimulateMatchesNow(dbCtx, *matches)
}

// SimulateMatchesNow force-simulates the given matches right away, regardless of their own
// match_date - unlike SimulateMatches, which only ever picks up whatever GetSimulableMatches'
// 15-minute kickoff window returns. Used to catch a newly created division up to the rest of its
// country's already-completed weeks (see handlers.AdminAddDivisionHandler, issue #68): those
// backfilled matches are dated in the future like any other freshly generated fixture, so they'd
// otherwise sit unplayed for days/weeks waiting for their own kickoff slot. Simulating several
// weeks' worth of matches in one batch is safe because classification updates are simple
// per-match additions (see UpdateClassification) - the order matches within the batch resolve in
// doesn't change the final standings.
func SimulateMatchesNow(dbCtx *common.DatabaseContext, matches []common.WeekMatch) {
	sem := make(chan struct{}, maxConcurrentSimulations)
	var wg sync.WaitGroup
	for _, match := range matches {
		log.Infof("simulating match %d", match.MatchId)
		wg.Add(1)
		sem <- struct{}{}
		go func(match common.WeekMatch) {
			defer wg.Done()
			defer func() { <-sem }()
			defer func() {
				if r := recover(); r != nil {
					log.Errorf("panic simulating match %d: %v", match.MatchId, r)
				}
			}()
			if _, err := simulateMatch(dbCtx, &match); err != nil {
				log.Errorf("error simulating match %d: %s", match.MatchId, err)
			}
		}(match)
	}
	wg.Wait()
}

// FinishStuckMatches re-simulates every league match a simulation pass claimed
// (in_simulation = true) but never finished simulating - the case a server crash mid-run leaves
// behind, since GetSimulableMatchesStmtSql's "and in_simulation is false" filter means the
// normal scheduler never revisits a claimed match on its own (see common.GetStuckMatches).
// Returns how many matches it re-simulated.
func FinishStuckMatches(dbCtx *common.DatabaseContext) int {
	matches, err := common.GetStuckMatches(dbCtx)
	if err != nil {
		log.Errorf("error getting stuck matches: %s", err)
		return 0
	}
	SimulateMatchesNow(dbCtx, *matches)
	return len(*matches)
}

// StuckSimulationsResult reports how many matches of each kind FinishStuckSimulations
// re-simulated.
type StuckSimulationsResult struct {
	LeagueMatches int
	Friendlies    int
}

// FinishStuckSimulations re-simulates every league match and friendly left stuck at
// in_simulation = true (played = false) by a simulation pass that never finished - typically a
// server crash mid-run. See FinishStuckMatches and FinishStuckFriendlies.
func FinishStuckSimulations(dbCtx *common.DatabaseContext) StuckSimulationsResult {
	return StuckSimulationsResult{
		LeagueMatches: FinishStuckMatches(dbCtx),
		Friendlies:    FinishStuckFriendlies(dbCtx),
	}
}
