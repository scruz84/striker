package engine

import (
	"fmt"
	"sync"
	"testing"
)

// TestBestXi_ClosedOnceWhenTheMatchdayEnds (needs STRIKER_TEST_DB_DSN; openspec/specs/best-xi):
// a simulated matchday gets its team of the week once its post-match effects are applied, and
// replicas closing it at once store it and notify the owner a single time.
func TestBestXi_ClosedOnceWhenTheMatchdayEnds(t *testing.T) {
	e := newSimEnv(t, uniqueCountryCode())
	owner := e.scan(`INSERT INTO users (name, provider, email) VALUES ('xi', 'dev', $1) RETURNING user_id`,
		fmt.Sprintf("bestxi-%d@example.test", e.home))
	e.exec(`UPDATE teams SET owner_id = $1 WHERE team_id = $2`, owner, e.home)

	matchId := e.leagueMatch(0)
	if _, err := simulateMatch(e.dbCtx, e.weekMatch(matchId)); err != nil {
		t.Fatalf("simulateMatch: %s", err)
	}
	if n := e.scan(`SELECT count(*) FROM best_xi_rounds WHERE league_id = $1`, e.league); n != 0 {
		t.Fatalf("team of the week stored before the match's post-match effects: %d rounds", n)
	}

	// The replay is over (kickoff three hours ago): every replica's recovery pass and replay
	// completion runs at once.
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ApplyDuePostMatchEffects(e.dbCtx)
			maybeCompleteBestXi(e.dbCtx, matchId)
		}()
	}
	wg.Wait()

	if n := e.scan(`SELECT count(*) FROM best_xi_rounds WHERE league_id = $1 AND season = 1 AND week_number = 1`, e.league); n != 1 {
		t.Fatalf("rounds = %d, want 1", n)
	}
	if n := e.scan(`SELECT count(*) FROM best_xi_players WHERE league_id = $1 AND season = 1 AND week_number = 1`, e.league); n < 1 || n > 11 {
		t.Errorf("players = %d, want 1 to 11", n)
	}
	homePicks := e.scan(`SELECT count(*) FROM best_xi_players WHERE league_id = $1 AND week_number = 1 AND team_id = $2`, e.league, e.home)
	notified := e.scan(`SELECT count(*) FROM notifications WHERE user_id = $1 AND message_key = 'notifications.messages.teamOfTheWeek'`, owner)
	if (homePicks > 0 && notified != 1) || (homePicks == 0 && notified != 0) {
		t.Errorf("owner notified %d times with %d of his players picked, want once if any", notified, homePicks)
	}
}
