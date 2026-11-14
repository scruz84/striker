package common

import (
	"errors"
	"sync"
	"testing"
	"time"
)

// The weekly friendly cap holds for the challenger as well as the challenged team, a club never
// plays two friendlies at once, and a friendly only counts against its own week (issue #402).
// Needs STRIKER_TEST_DB_DSN.

type friendlyCapRig struct {
	dbCtx  *DatabaseContext
	league int64
}

func newFriendlyCapRig(t *testing.T, maxPerWeek int) *friendlyCapRig {
	t.Helper()
	dbCtx := accDb(t)
	db := dbCtx.Db
	accMust(t, func() error {
		_, err := db.Exec(`INSERT INTO season (transfer_window_open) SELECT true WHERE NOT EXISTS (SELECT 1 FROM season)`)
		return err
	}(), "seeding season")
	// The cap and the friendly days are global settings: restore them for the other tests.
	var oldMax, oldDays int
	accMust(t, db.QueryRow(`SELECT max_friendlies_per_week, friendly_schedule_weekdays FROM season`).Scan(&oldMax, &oldDays), "reading friendly settings")
	t.Cleanup(func() {
		_, _ = db.Exec(`UPDATE season SET max_friendlies_per_week = $1, friendly_schedule_weekdays = $2`, oldMax, oldDays)
	})
	accMust(t, SetMaxFriendliesPerWeek(dbCtx, maxPerWeek), "setting the cap")
	accMust(t, SetFriendlyScheduleWeekdays(dbCtx, AllWeekdaysMask), "allowing every day")
	return &friendlyCapRig{dbCtx: dbCtx, league: accSeedLeague(t, db)}
}

func (r *friendlyCapRig) club(t *testing.T, human bool) int64 {
	t.Helper()
	var ownerId *int64
	if human {
		id := accSeedUser(t, r.dbCtx.Db)
		ownerId = &id
	}
	return accSeedTeam(t, r.dbCtx, r.league, ownerId, 0)
}

// challengeId sends a challenge from challenger to a human opponent (so it waits) and returns it.
func (r *friendlyCapRig) challenge(t *testing.T, challenger, opponent int64) int64 {
	t.Helper()
	accMust(t, ChallengeToFriendly(r.dbCtx, challenger, opponent, nil, true), "challenging")
	return accInt(t, r.dbCtx.Db, `SELECT challenge_id FROM friendly_challenges WHERE challenger_team_id = $1 AND opponent_team_id = $2`, challenger, opponent)
}

func (r *friendlyCapRig) friendlies(t *testing.T, team int64) int64 {
	t.Helper()
	return accInt(t, r.dbCtx.Db, `SELECT count(*) FROM friendly_matches WHERE $1 IN (home_team_id, away_team_id)`, team)
}

// tomorrowAt is a valid kickoff: tomorrow (UTC) at hour:minute.
func tomorrowAt(hour, minute int) time.Time {
	d := time.Now().UTC().AddDate(0, 0, 1)
	return time.Date(d.Year(), d.Month(), d.Day(), hour, minute, 0, 0, time.UTC)
}

func TestAcceptFriendlyChallenge_ChallengerCapHolds(t *testing.T) {
	r := newFriendlyCapRig(t, 1)
	challenger, first, second := r.club(t, false), r.club(t, true), r.club(t, true)
	// Both sent while the challenger has no friendly yet.
	toFirst := r.challenge(t, challenger, first)
	toSecond := r.challenge(t, challenger, second)

	accMust(t, AcceptFriendlyChallenge(r.dbCtx, toFirst, first, tomorrowAt(12, 0)), "first acceptance")
	if err := AcceptFriendlyChallenge(r.dbCtx, toSecond, second, tomorrowAt(13, 0)); !errors.Is(err, ErrFriendlyLimitReached) {
		t.Fatalf("second acceptance = %v, want ErrFriendlyLimitReached", err)
	}
	if n := r.friendlies(t, challenger); n != 1 {
		t.Errorf("challenger plays %d friendlies, want 1", n)
	}
}

func TestAcceptFriendlyChallenge_ConcurrentAcceptancesKeepTheCap(t *testing.T) {
	r := newFriendlyCapRig(t, 1)
	challenger := r.club(t, false)
	const opponents = 3
	ids := make([]int64, opponents)
	teams := make([]int64, opponents)
	for i := range ids {
		teams[i] = r.club(t, true)
		ids[i] = r.challenge(t, challenger, teams[i])
	}
	var wg sync.WaitGroup
	for i := range ids {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_ = AcceptFriendlyChallenge(r.dbCtx, ids[i], teams[i], tomorrowAt(12+i, 0))
		}(i)
	}
	wg.Wait()
	if n := r.friendlies(t, challenger); n != 1 {
		t.Errorf("challenger plays %d friendlies after concurrent acceptances, want 1", n)
	}
}

func TestAcceptFriendlyChallenge_NoTwoFriendliesAtTheSameKickoff(t *testing.T) {
	r := newFriendlyCapRig(t, 3)
	opponent, a, b := r.club(t, true), r.club(t, false), r.club(t, false)
	fromA := r.challenge(t, a, opponent)
	fromB := r.challenge(t, b, opponent)
	kickoff := tomorrowAt(18, 30)

	accMust(t, AcceptFriendlyChallenge(r.dbCtx, fromA, opponent, kickoff), "first acceptance")
	if err := AcceptFriendlyChallenge(r.dbCtx, fromB, opponent, kickoff); !errors.Is(err, ErrFriendlyKickoffTaken) {
		t.Fatalf("second acceptance at the same kickoff = %v, want ErrFriendlyKickoffTaken", err)
	}
	accMust(t, AcceptFriendlyChallenge(r.dbCtx, fromB, opponent, kickoff.Add(time.Hour)), "acceptance an hour later")
	if n := r.friendlies(t, opponent); n != 2 {
		t.Errorf("opponent plays %d friendlies, want 2", n)
	}
}

func TestChallengeToFriendly_NextWeeksFriendlyDoesNotUseThisWeeksCap(t *testing.T) {
	r := newFriendlyCapRig(t, 1)
	challenger, other, opponent := r.club(t, false), r.club(t, false), r.club(t, true)
	nextWeek := StartOfWeek(time.Now()).AddDate(0, 0, 8).Add(12 * time.Hour)
	accMust(t, func() error {
		_, err := r.dbCtx.Db.Exec(`INSERT INTO friendly_matches (home_team_id, away_team_id, match_date) VALUES ($1, $2, $3)`, challenger, other, nextWeek)
		return err
	}(), "seeding next week's friendly")

	accMust(t, ChallengeToFriendly(r.dbCtx, challenger, opponent, nil, true), "challenging for this week")
	proposed := nextWeek.Add(time.Hour)
	if err := ChallengeToFriendly(r.dbCtx, challenger, r.club(t, true), &proposed, true); !errors.Is(err, ErrFriendlyLimitReached) {
		t.Fatalf("challenge proposing a kickoff in the full week = %v, want ErrFriendlyLimitReached", err)
	}
}
