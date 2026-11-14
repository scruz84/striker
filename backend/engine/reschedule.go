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
//
// Only matches that have not started move: a round can be half played (its matches are simulated
// in batches up to 15 minutes before kickoff, and a failed one waits for its retry), and a played
// or in-simulation match keeps its date - moving it to the future would hide its result again as
// "still live" and hold the season rollover back (issue #400).
func RescheduleUnplayedMatches(dbCtx *common.DatabaseContext, leagueId int64, schedule common.MatchSchedule) error {
	season, err := common.GetCurrentSeasonForLeague(dbCtx, leagueId)
	if err != nil {
		return err
	}

	rows, err := dbCtx.Db.Query(
		"SELECT DISTINCT week_number FROM matches WHERE league_id = $1 AND season = $2 AND played = false AND in_simulation = false ORDER BY week_number",
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
			"UPDATE matches SET match_date = $1 WHERE league_id = $2 AND season = $3 AND week_number = $4 AND played = false AND in_simulation = false",
			dates[i], leagueId, season, week); err != nil {
			return err
		}
	}
	return nil
}

// RescheduleUnplayedMatchesForCountry re-dates every unplayed league match of country's current
// season, league by league, each onto its own effective schedule (see
// common.EffectiveLeagueSchedule: its per-league override if it has one, otherwise the country's
// league kickoff for its tier), starting from the next matching slot after now. Used by the admin
// "reschedule country" action and after a country's league kickoff changes. Best-effort per
// league, like RescheduleUnplayedMatchesForLeagues; returns how many leagues it was run for.
func RescheduleUnplayedMatchesForCountry(dbCtx *common.DatabaseContext, country string) (int, error) {
	leagueIds, err := common.GetLeagueIdsForCountry(dbCtx, country)
	if err != nil {
		return 0, err
	}
	if err := RescheduleUnplayedMatchesForLeagues(dbCtx, leagueIds); err != nil {
		return 0, err
	}
	return len(leagueIds), nil
}

// RescheduleUnplayedMatchesForLeagues is RescheduleUnplayedMatches for several leagues at once,
// each onto its own effective schedule, best-effort per league - one league's failure to
// reschedule shouldn't abandon the rest (used when the app-wide default schedule changes,
// affecting every league without its own override).
func RescheduleUnplayedMatchesForLeagues(dbCtx *common.DatabaseContext, leagueIds []int64) error {
	if len(leagueIds) == 0 {
		return nil
	}
	schedules, err := common.GetEffectiveLeagueSchedules(dbCtx, leagueIds)
	if err != nil {
		return err
	}
	for _, leagueId := range leagueIds {
		if err := RescheduleUnplayedMatches(dbCtx, leagueId, schedules[leagueId]); err != nil {
			log.Errorf("error rescheduling unplayed matches for league %d: %s", leagueId, err)
		}
	}
	return nil
}
