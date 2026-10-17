package engine

import (
	"time"

	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
)

// RescheduleUnplayedMatches re-dates every currently unplayed league match for leagueId in the
// current season onto schedule's weekdays/time, preserving each match's existing week_number
// ordering (the earliest unplayed week lands on the next matching occurrence from now, later
// weeks follow one round after another - the same logic ComputeFixtureDates already uses when
// fixtures are first generated). Called whenever an admin changes a league's effective schedule
// (its own override, or the app-wide default for a league with none) so the change is felt
// immediately rather than only by fixtures generated at the next season rollover (issue #67).
func RescheduleUnplayedMatches(dbCtx *common.DatabaseContext, leagueId int64, schedule common.MatchSchedule) error {
	season, err := common.GetCurrentSeasonForLeague(dbCtx, leagueId)
	if err != nil {
		return err
	}

	rows, err := dbCtx.Db.Query(
		"SELECT DISTINCT week_number FROM matches WHERE league_id = $1 AND season = $2 AND played = false ORDER BY week_number",
		leagueId, season)
	if err != nil {
		return err
	}
	weekNumbers := make([]int, 0)
	for rows.Next() {
		var week int
		if err := rows.Scan(&week); err != nil {
			rows.Close()
			return err
		}
		weekNumbers = append(weekNumbers, week)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	rows.Close()
	if len(weekNumbers) == 0 {
		return nil
	}

	// A match already simulated (or being simulated) can still be dated up to 15 minutes ahead -
	// the new rounds start after it, not on its minute.
	latestStarted, err := common.GetLatestStartedMatchDate(dbCtx, leagueId, season)
	if err != nil {
		return err
	}
	dates := ComputeFixtureDates(FixtureStart(time.Now(), latestStarted), schedule, len(weekNumbers))
	for i, week := range weekNumbers {
		if _, err := dbCtx.Db.Exec(
			"UPDATE matches SET match_date = $1 WHERE league_id = $2 AND season = $3 AND week_number = $4",
			dates[i], leagueId, season, week); err != nil {
			return err
		}
	}
	return nil
}

// RescheduleUnplayedMatchesForLeagues is RescheduleUnplayedMatches for several leagues at once,
// best-effort per league - one league's failure to reschedule shouldn't abandon the rest (used
// when the app-wide default schedule changes, affecting every league without its own override).
func RescheduleUnplayedMatchesForLeagues(dbCtx *common.DatabaseContext, leagueIds []int64, schedule common.MatchSchedule) {
	for _, leagueId := range leagueIds {
		if err := RescheduleUnplayedMatches(dbCtx, leagueId, schedule); err != nil {
			log.Errorf("error rescheduling unplayed matches for league %d: %s", leagueId, err)
		}
	}
}
