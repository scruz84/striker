package engine

import (
	"encoding/json"
	"fmt"
	"math"
	"testing"

	"github.com/scruz84/striker/common"
)

// Rivalries in the engine (openspec add-emergent-rivalries; needs STRIKER_TEST_DB_DSN).

func (e *simEnv) setRivalry(a, b int64, score float64) {
	e.t.Helper()
	e.exec(`INSERT INTO team_rivalries (team_low_id, team_high_id, country, score) VALUES (LEAST($1::bigint, $2::bigint), GREATEST($1::bigint, $2::bigint), $3, $4)
		ON CONFLICT (team_low_id, team_high_id) DO UPDATE SET score = EXCLUDED.score`, a, b, e.country, score)
}

func (e *simEnv) rivalryScore(a, b int64) float64 {
	e.t.Helper()
	score, err := common.GetRivalryScore(e.dbCtx, a, b)
	if err != nil {
		e.t.Fatal(err)
	}
	return score
}

// Level fixed at kickoff: a match between two clubs at 15 is played and stored as a Rival match.
func TestSimulateMatch_RivalryLevelFixedAtKickoff(t *testing.T) {
	e := newSimEnv(t, "rv1")
	e.setRivalry(e.home, e.away, 15)
	matchId := e.leagueMatch(0)
	if _, err := simulateMatch(e.dbCtx, e.weekMatch(matchId)); err != nil {
		t.Fatalf("simulateMatch: %s", err)
	}
	if level := e.scan(`SELECT rivalry_level FROM matches WHERE match_id = $1`, matchId); level != common.RIVALRY_RIVAL {
		t.Errorf("stored rivalry level %d, want Rival", level)
	}
	// Whatever the pair's score is now, the played match keeps showing the level it kicked off with.
	e.setRivalry(e.home, e.away, 30)
	if match, err := common.WeekMatchById(e.dbCtx, matchId); err != nil || match.Rivalry != common.RIVALRY_RIVAL {
		t.Errorf("played match shows rivalry %v (%v), want Rival", match, err)
	}
	upcoming := e.leagueMatch(0)
	e.exec(`UPDATE matches SET match_date = now() + interval '3 days' WHERE match_id = $1`, upcoming)
	if match, err := common.WeekMatchById(e.dbCtx, upcoming); err != nil || match.Rivalry != common.RIVALRY_CLASICO {
		t.Errorf("upcoming match shows rivalry %v (%v), want the current Clásico", match, err)
	}
}

// A match's rivalry points (and the notification of a level they raise) wait for its post-match
// effects, which wait for the replay: nothing changes while the pending record is not due.
func TestPostMatchEffects_RivalryPointsOnlyAfterTheReplay(t *testing.T) {
	e := newSimEnv(t, "rv2")
	users := map[int64]int64{}
	for _, teamId := range []int64{e.home, e.away} {
		users[teamId] = e.scan(`INSERT INTO users (name, provider, email) VALUES ('rv', 'dev', $1) RETURNING user_id`, fmt.Sprintf("rv2-%d@example.test", teamId))
		e.exec(`UPDATE teams SET owner_id = $1 WHERE team_id = $2`, users[teamId], teamId)
	}
	e.setRivalry(e.home, e.away, 7)
	matchId := e.leagueMatch(1)
	// 2-1 with one red card: (1 + 1) × 1.5 (two human clubs) = 3, taking the pair from 7 to Rival.
	payload, _ := json.Marshal(common.PostMatchPayload{Home: e.home, Away: e.away, HomeGoals: 2, AwayGoals: 1,
		Events: []common.Event{{EventType: common.RED_CARD, Minute: 50, TeamOwner: e.away}}})
	e.exec(`INSERT INTO pending_post_match_effects (kind, match_id, due_at, payload) VALUES ('league', $1, now() + interval '1 hour', $2)`, matchId, payload)
	notifications := func(userId int64) int64 {
		return e.scan(`SELECT count(*) FROM notifications WHERE user_id = $1 AND notification_type = $2`, userId, common.NOTIFICATION_RIVALRY)
	}

	ApplyDuePostMatchEffects(e.dbCtx)
	if score := e.rivalryScore(e.home, e.away); score != 7 {
		t.Fatalf("score %v before the replay ended, want 7", score)
	}
	if n := notifications(users[e.home]); n != 0 {
		t.Fatalf("%d rivalry notifications before the replay ended", n)
	}

	e.exec(`UPDATE pending_post_match_effects SET due_at = now() - interval '1 minute' WHERE match_id = $1`, matchId)
	ApplyDuePostMatchEffects(e.dbCtx)
	if score := e.rivalryScore(e.home, e.away); math.Abs(score-10) > 1e-6 {
		t.Errorf("score %v after the replay, want 10", score)
	}
	for teamId, userId := range users {
		if n := notifications(userId); n != 1 {
			t.Errorf("owner of team %d has %d rivalry notifications, want 1", teamId, n)
		}
	}
}

// Title race and decay at the rollover, applied once with several replicas running it at once.
func TestAdvanceCountrySeason_RivalriesTitleRaceAndDecayOnce(t *testing.T) {
	db, dbCtx := openReplicaTestDb(t)
	scan := func(query string, args ...any) int64 {
		t.Helper()
		var v int64
		if err := db.QueryRow(query, args...).Scan(&v); err != nil {
			t.Fatalf("%s: %s", query, err)
		}
		return v
	}
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := db.Exec(query, args...); err != nil {
			t.Fatalf("%s: %s", query, err)
		}
	}
	country := uniqueCountryCode()
	exec(`INSERT INTO countries (name, current_season) VALUES ($1, 1)`, country)
	league := scan(`INSERT INTO leagues (country, group_number, name, tier) VALUES ($1, 1, 'D1', 1) RETURNING league_id`, country)
	teams := make([]int64, 6)
	for i := range teams {
		teams[i] = scan(`INSERT INTO teams (name, country, league_id) VALUES ($1, $2, $3) RETURNING team_id`, fmt.Sprintf("RV%d-%d", league, i), country, league)
		exec(`INSERT INTO classification (league_id, team_id, season, points) VALUES ($1, $2, 1, $3)`, league, teams[i], 30-i)
	}
	exec(`INSERT INTO matches (league_id, season, week_number, match_date, home_team_id, away_team_id, played)
		VALUES ($1, 1, 1, now() - interval '3 hours', $2, $3, true)`, league, teams[0], teams[1])
	set := func(a, b int64, score float64) {
		exec(`INSERT INTO team_rivalries (team_low_id, team_high_id, country, score) VALUES (LEAST($1::bigint, $2::bigint), GREATEST($1::bigint, $2::bigint), $3, $4)`, a, b, country, score)
	}
	set(teams[2], teams[3], 20) // an old feud
	set(teams[0], teams[1], 10) // the title race's pair

	runConcurrently(replicas, func() { advanceCountrySeasonIfComplete(dbCtx, country) })

	score := func(a, b int64) float64 {
		s, err := common.GetRivalryScore(dbCtx, a, b)
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	// Title race: (10 + 4) × 0.6 = 8.4; old feud: 20 × 0.6 = 12 - each applied exactly once.
	if got := score(teams[0], teams[1]); math.Abs(got-8.4) > 1e-4 {
		t.Errorf("champion-runner-up pair after the rollover = %v, want 8.4", got)
	}
	if got := score(teams[2], teams[3]); math.Abs(got-12) > 1e-4 {
		t.Errorf("old feud after the rollover = %v, want 12", got)
	}
}
