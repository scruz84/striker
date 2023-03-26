package engine

import (
	"footmanager/fmcommon"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"time"
)

// GenerateMatches Generates the matches for the leagues
func GenerateMatches(dbCtx *fmcommon.DatabaseContext) {
	rows, err := dbCtx.Db.Query("select 1 as c from matches where match_date > now() limit 1")
	defer rows.Close()
	if err != nil {
		log.Errorf("error checking for upcoming matches: %s", err)
		return
	}

	if !rows.Next() {
		//there are no upcoming matches, so we have to generate new ones
		rows, err := dbCtx.Db.Query("select league_id from leagues")
		if err != nil {
			log.Errorf("error finding leagues: %s", err)
		}
		for rows.Next() {
			var leagueId int64
			_ = rows.Scan(&leagueId)
			teams, err := dbCtx.Db.Query(fmcommon.GetTeamIdsFromLeagueSql, leagueId)
			if err != nil {
				log.Errorf("error finding teams from league %d: %s", err)
			}
			teamIds := make([]int64, 0)
			for teams.Next() {
				var teamId int64
				_ = teams.Scan(&teamId)
				teamIds = append(teamIds, teamId)
			}
			defer teams.Close()

			log.Infof("Team ids: %s", teamIds)

			weekMatches := randomWeekMatches2(&leagueId, &teamIds)
			//Insert the matches
			_ = fmcommon.InsertMatches(dbCtx, weekMatches)
		}
	}
}

// Algorithm from: https://www.competize.com/blog/hacer-fixture-crear-calendario-liga-excel/
func randomWeekMatches2(leagueId *int64, teamIds *[]int64) [][]*fmcommon.WeekMatch {
	// Here we have a random sort for the teams
	rand.New(rand.NewSource(time.Now().UnixNano()))
	rand.Shuffle(len(*teamIds), func(i, j int) {
		(*teamIds)[i], (*teamIds)[j] = (*teamIds)[j], (*teamIds)[i]
	})

	nWeeks := (len(*teamIds) - 1) * 2
	nMatchesPerWeek := len(*teamIds) / 2
	weeksMatches := make([][]*fmcommon.WeekMatch, nWeeks)

	teamIdxForMatchFromWeek := 0
	teamIdxForRestMatchesFromWeek := len(*teamIds) - 2
	latestTeamId := &(*teamIds)[len(*teamIds)-1]
	now := time.Now()
	for weekIdx := 0; weekIdx < nWeeks; weekIdx++ {
		weeksMatches[weekIdx] = make([]*fmcommon.WeekMatch, nMatchesPerWeek)
		for weekIndex := 0; weekIndex < nMatchesPerWeek; weekIndex++ {
			idx := weekIdx + 1
			home := (*teamIds)[teamIdxForMatchFromWeek]
			weeksMatches[weekIdx][weekIndex] = &fmcommon.WeekMatch{
				DivisionId: *leagueId,
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
	for weekIdx, matches := range weeksMatches {
		weekDate := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute()+2, 0, 0, now.Location())
		//weekDate = weekDate.Add(time.Duration(weekIdx+1) * time.Hour * 24)
		weekDate = weekDate.Add(time.Duration(weekIdx+1) * time.Minute * 15)
		for _, match := range matches {
			match.MatchDate = weekDate
		}
	}
	return weeksMatches
}

// SimulateMatches simulate matches which has a match date 15 minutes prior current time.
func SimulateMatches(dbCtx *fmcommon.DatabaseContext) {
	matches, err := fmcommon.GetSimulableMatches(dbCtx)
	if funcName(err) {
		return
	}
	leagueIds := make(map[int64]bool)
	for _, match := range *matches {
		matchSimulation, err := simulateMatch(dbCtx, &match)
		leagueIds[match.DivisionId] = true
		if err != nil {
			log.Errorf("error simulating match: %s", err)
		} else {
			go UpdatePlayedMatch(dbCtx, matchSimulation)
		}
	}

}

func funcName(err error) bool {
	if err != nil {
		log.Errorf("error getting matches to simulate: %s", err)
		return true
	}
	return false
}
