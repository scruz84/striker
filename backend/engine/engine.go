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

// GenerateMatches Generates the matches for the leagues
func GenerateMatches(dbCtx *common.DatabaseContext) {
	season, err := GetCurrentSeason(dbCtx)
	if err != nil {
		log.Errorf("error checking for upcoming matches: %s", err)
		return
	}
	rows, err := dbCtx.Db.Query("select 1 as c from matches where season = $1 and match_date > now() limit 1", season)
	defer rows.Close()
	if err != nil {
		log.Errorf("error checking for upcoming matches: %s", err)
		return
	}

	if !rows.Next() {
		//there are no upcoming matches, so we have to generate new ones
		leagueRows, err := dbCtx.Db.Query("select league_id from leagues")
		if err != nil {
			log.Errorf("error finding leagues: %s", err)
			return
		}
		defer leagueRows.Close()
		leagueIds := make([]int64, 0)
		for leagueRows.Next() {
			var leagueId int64
			_ = leagueRows.Scan(&leagueId)
			leagueIds = append(leagueIds, leagueId)
		}
		generateMatchesForLeagueIds(dbCtx, leagueIds, season)
	}
}

// GenerateMatchesForLeagues generates the current season's fixtures for exactly the given
// leagues, bypassing GenerateMatches' "does ANY league already have upcoming matches" guard -
// needed when a league is added mid-season (see handlers.AdminAddCountryHandler), where every
// other league already having fixtures scheduled would otherwise make GenerateMatches skip
// straight past the new one.
func GenerateMatchesForLeagues(dbCtx *common.DatabaseContext, leagueIds []int64) error {
	season, err := GetCurrentSeason(dbCtx)
	if err != nil {
		return err
	}
	generateMatchesForLeagueIds(dbCtx, leagueIds, season)
	return nil
}

// generateMatchesForLeagueIds dates each league's fixtures on its own schedule override if it
// has one (see common.GetLeagueScheduleOverrides), falling back to the app-wide default (see
// common.GetMatchSchedule) otherwise.
func generateMatchesForLeagueIds(dbCtx *common.DatabaseContext, leagueIds []int64, season int) {
	defaultSchedule := common.GetMatchSchedule(dbCtx)
	overrides, err := common.GetLeagueScheduleOverrides(dbCtx)
	if err != nil {
		log.Errorf("error loading per-league schedule overrides, using the default for every league: %s", err)
		overrides = nil
	}

	// Searched from once, before any league's fixtures are inserted: a season closes while its last
	// round can still be dated in the future (matches are simulated 15 minutes ahead), and the new
	// season must start after it. Only earlier seasons count, so a league added mid-season isn't
	// pushed past the end of everyone else's current fixtures.
	latestPrevious, err := common.GetLatestMatchDateBeforeSeason(dbCtx, season)
	if err != nil {
		log.Errorf("error finding the previous season's last match date, scheduling from now: %s", err)
	}
	from := FixtureStart(time.Now(), latestPrevious)

	for _, leagueId := range leagueIds {
		teams, err := dbCtx.Db.Query(common.GetTeamIdsFromLeagueSql, leagueId)
		if err != nil {
			log.Errorf("error finding teams from league %d: %s", leagueId, err)
			continue
		}
		teamIds := make([]int64, 0)
		for teams.Next() {
			var teamId int64
			_ = teams.Scan(&teamId)
			teamIds = append(teamIds, teamId)
		}
		teams.Close()

		log.Infof("Team ids: %v", teamIds)

		schedule := defaultSchedule
		if override, ok := overrides[leagueId]; ok {
			schedule = override
		}

		weekMatches := randomWeekMatches2(&leagueId, season, &teamIds, schedule, from)
		//Insert the matches
		_ = common.InsertMatches(dbCtx, weekMatches)
	}
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
