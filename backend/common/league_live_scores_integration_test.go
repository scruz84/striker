package common

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"
)

// leagueLiveScoresFixture seeds an isolated league with 4 teams and returns helpers to add
// matches to it. Needs Postgres (STRIKER_TEST_DB_DSN) and Redis, like the rest of the live tests.
func leagueLiveScoresFixture(t *testing.T) (*DatabaseContext, int64, func(week int, kickoff time.Time, played bool, homeGoals, awayGoals int) int64) {
	t.Helper()
	dsn := os.Getenv("STRIKER_TEST_DB_DSN")
	if dsn == "" {
		t.Skip("STRIKER_TEST_DB_DSN not set")
	}
	hub := liveHubForTest(t)
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("opening database: %s", err)
	}
	t.Cleanup(func() { db.Close() })
	dbCtx := &DatabaseContext{Db: db, LiveHub: hub, ObjectCache: &memoryCache{values: map[string]string{}}}
	if err := EnsureSchema(dbCtx); err != nil {
		t.Fatalf("EnsureSchema: %s", err)
	}

	mustScan := func(dest *int64, query string, args ...any) {
		t.Helper()
		if err := db.QueryRow(query, args...).Scan(dest); err != nil {
			t.Fatalf("%s: %s", query, err)
		}
	}
	if _, err := db.Exec(`INSERT INTO countries (name) VALUES ('zl') ON CONFLICT DO NOTHING`); err != nil {
		t.Fatalf("seeding country: %s", err)
	}
	var leagueId, teamA, teamB int64
	mustScan(&leagueId, `INSERT INTO leagues (country, group_number, name) VALUES ('zl', 1, 'Live') RETURNING league_id`)
	mustScan(&teamA, `INSERT INTO teams (name, country) VALUES ('Alpha', 'zl') RETURNING team_id`)
	mustScan(&teamB, `INSERT INTO teams (name, country) VALUES ('Beta', 'zl') RETURNING team_id`)

	addMatch := func(week int, kickoff time.Time, played bool, homeGoals, awayGoals int) int64 {
		t.Helper()
		var id int64
		mustScan(&id, `INSERT INTO matches (league_id, season, week_number, played, match_date, home_team_id, away_team_id, home_goals, away_goals)
			VALUES ($1, 700, $2, $3, $4, $5, $6, $7, $8) RETURNING match_id`, leagueId, week, played, kickoff, teamA, teamB, homeGoals, awayGoals)
		channel := LeagueMatchChannel(id)
		t.Cleanup(func() { hub.client.Del(context.Background(), historyKey(channel), lastPublishKey(channel)) })
		return id
	}
	return dbCtx, leagueId, addMatch
}

func TestGetLeagueLiveScores_MatchdayBeingBroadcast(t *testing.T) {
	dbCtx, leagueId, addMatch := leagueLiveScoresFixture(t)
	now := time.Now()

	// Two matches of matchday 12 kicked off 40 minutes ago (simulated 3-2 and 4-0 already), one
	// match still 5 minutes from kickoff (simulated early), one 3 minutes late and not simulated.
	// All must be reported with the broadcast's score, never the stored one.
	kickoff := now.Add(-40 * time.Minute)
	later := addMatch(12, kickoff.Add(time.Second), true, 4, 0)
	first := addMatch(12, kickoff, true, 3, 2)
	early := addMatch(13, now.Add(5*time.Minute), true, 2, 2)
	late := addMatch(13, now.Add(-3*time.Minute), false, 0, 0)
	outside := addMatch(11, now.Add(-3*time.Hour), true, 1, 1)
	farAway := addMatch(14, now.Add(2*24*time.Hour), false, 0, 0)

	dbCtx.LiveHub.Publish(LeagueMatchChannel(first), LiveMatchMessage{Type: "instant", Instant: &MatchInstant{Minute: 63}, HomeGoals: 1, AwayGoals: 2})
	dbCtx.LiveHub.Publish(LeagueMatchChannel(later), LiveMatchMessage{Type: "halftime", HomeGoals: 0, AwayGoals: 0})
	dbCtx.LiveHub.Publish(LeagueMatchChannel(early), LiveMatchMessage{Type: "instant", Instant: &MatchInstant{Minute: 1}, HomeGoals: 9, AwayGoals: 9}) // must be ignored: not kicked off

	scores, err := GetLeagueLiveScores(dbCtx, leagueId)
	if err != nil {
		t.Fatalf("GetLeagueLiveScores: %s", err)
	}

	// Ordered by kickoff then match id: first (-40m), later (-40m + 1s), late (-3m), early (+5m).
	wantIds := []int64{first, later, late, early}
	if len(scores) != len(wantIds) {
		t.Fatalf("got %d entries (%+v), want %d; outside=%d farAway=%d", len(scores), scores, len(wantIds), outside, farAway)
	}
	for i, id := range wantIds {
		if scores[i].MatchId != id {
			t.Errorf("entry %d: match %d, want %d", i, scores[i].MatchId, id)
		}
	}

	if s := scores[0]; s.Status != LiveStatusLive || s.Minute != 63 || s.HomeGoals != 1 || s.AwayGoals != 2 || s.WeekNumber != 12 || s.HomeName != "Alpha" || s.AwayName != "Beta" {
		t.Errorf("final result already stored: got %+v, want live minute 63 at 1-2 (stored result is 3-2)", s)
	}
	if s := scores[1]; s.Status != LiveStatusHalftime || s.Minute != 45 {
		t.Errorf("half-time: got %+v", s)
	}
	for _, s := range scores[2:] {
		if s.Status != LiveStatusPending || s.Minute != 0 || s.HomeGoals != 0 || s.AwayGoals != 0 {
			t.Errorf("pending entry: got %+v, want pending minute 0 at 0-0", s)
		}
	}
}

func TestGetLeagueLiveScores_NoMatchdayInPlay(t *testing.T) {
	dbCtx, leagueId, addMatch := leagueLiveScoresFixture(t)
	addMatch(11, time.Now().Add(-3*time.Hour), true, 1, 1)
	addMatch(12, time.Now().Add(2*24*time.Hour), false, 0, 0)

	scores, err := GetLeagueLiveScores(dbCtx, leagueId)
	if err != nil || scores == nil || len(scores) != 0 {
		t.Errorf("got %+v, %v; want an empty, non-nil list", scores, err)
	}
	if unknown, err := GetLeagueLiveScores(dbCtx, -1); err != nil || unknown == nil || len(unknown) != 0 {
		t.Errorf("unknown league: got %+v, %v; want an empty, non-nil list", unknown, err)
	}
}

func TestGetLeagueLiveScores_BroadcastFinished(t *testing.T) {
	dbCtx, leagueId, addMatch := leagueLiveScoresFixture(t)
	id := addMatch(12, time.Now().Add(110*time.Minute*-1), true, 3, 2)
	dbCtx.LiveHub.Publish(LeagueMatchChannel(id), LiveMatchMessage{Type: "complete", HomeGoals: 3, AwayGoals: 2})

	scores, err := GetLeagueLiveScores(dbCtx, leagueId)
	if err != nil || len(scores) != 1 {
		t.Fatalf("got %+v, %v", scores, err)
	}
	if s := scores[0]; s.Status != LiveStatusFinished || s.Minute != 90 || s.HomeGoals != 3 || s.AwayGoals != 2 {
		t.Errorf("got %+v, want finished minute 90 at 3-2", s)
	}
}

func TestGetLeagueLiveScores_BroadcastStateUnreadable(t *testing.T) {
	dbCtx, leagueId, addMatch := leagueLiveScoresFixture(t)
	// Kicked off 20 real minutes ago, simulated 2-1 with a first goal at minute 12 and the second
	// at minute 70. With no broadcast state to read, the estimate is minute 21 and only the first
	// goal counts.
	id := addMatch(12, time.Now().Add(-20*time.Minute), true, 2, 1)
	if _, err := dbCtx.Db.Exec(`INSERT INTO match_events (match_id, minute, event_type, home_goals, away_goals) VALUES
		($1, 12, 1, 1, 0), ($1, 70, 1, 2, 0)`, id); err != nil {
		t.Fatalf("seeding events: %s", err)
	}
	dbCtx.LiveHub = nil

	scores, err := GetLeagueLiveScores(dbCtx, leagueId)
	if err != nil || len(scores) != 1 {
		t.Fatalf("got %+v, %v", scores, err)
	}
	if s := scores[0]; s.Status != LiveStatusLive || s.Minute != 21 || s.HomeGoals != 1 || s.AwayGoals != 0 {
		t.Errorf("got %+v, want live minute 21 at 1-0", s)
	}
}

func TestGetLeagueLiveScores_ServedFromCacheWithinTTL(t *testing.T) {
	dbCtx, leagueId, addMatch := leagueLiveScoresFixture(t)
	id := addMatch(12, time.Now().Add(-30*time.Minute), true, 1, 0)
	dbCtx.LiveHub.Publish(LeagueMatchChannel(id), LiveMatchMessage{Type: "instant", Instant: &MatchInstant{Minute: 30}, HomeGoals: 0, AwayGoals: 0})

	first, err := GetLeagueLiveScores(dbCtx, leagueId)
	if err != nil || len(first) != 1 || first[0].Minute != 30 {
		t.Fatalf("first call: %+v, %v", first, err)
	}

	// The broadcast moves on and a second match appears, but a call within the TTL still sees
	// the cached response.
	dbCtx.LiveHub.Publish(LeagueMatchChannel(id), LiveMatchMessage{Type: "instant", Instant: &MatchInstant{Minute: 31}, HomeGoals: 1, AwayGoals: 0})
	addMatch(12, time.Now().Add(-30*time.Minute), true, 0, 0)
	second, err := GetLeagueLiveScores(dbCtx, leagueId)
	if err != nil || len(second) != 1 || second[0].Minute != 30 {
		t.Errorf("second call within the TTL: %+v, %v; want the cached minute 30", second, err)
	}

	// Once the cache entry is gone the new state shows.
	dbCtx.ObjectCache.DeleteValue(leagueLiveScoresCacheKey(leagueId))
	third, err := GetLeagueLiveScores(dbCtx, leagueId)
	if err != nil || len(third) != 2 {
		t.Errorf("third call after the cache expired: %+v, %v; want 2 entries", third, err)
	}
}

// "Entry in added time", "Winner scored in added time" and "Broadcast finished" with added time
// (division-live-scores): entries carry the broadcast's added minute, and a winner stored at 90+3
// is not revealed while the broadcast is at 90+1.
func TestGetLeagueLiveScores_AddedTime(t *testing.T) {
	dbCtx, leagueId, addMatch := leagueLiveScoresFixture(t)
	kickoff := time.Now().Add(-105 * time.Minute)
	inAddedTime := addMatch(12, kickoff, true, 2, 2)
	winnerLate := addMatch(12, kickoff.Add(time.Second), true, 2, 1)
	finished := addMatch(12, kickoff.Add(2*time.Second), true, 3, 2)
	halftime := addMatch(13, time.Now().Add(-47*time.Minute), true, 0, 1)

	dbCtx.LiveHub.Publish(LeagueMatchChannel(inAddedTime), LiveMatchMessage{Type: "instant", Instant: &MatchInstant{Minute: 90, AddedMinute: 2}, HomeGoals: 2, AwayGoals: 2})
	dbCtx.LiveHub.Publish(LeagueMatchChannel(winnerLate), LiveMatchMessage{Type: "instant", Instant: &MatchInstant{Minute: 90, AddedMinute: 1}, HomeGoals: 1, AwayGoals: 1})
	dbCtx.LiveHub.Publish(LeagueMatchChannel(finished), LiveMatchMessage{Type: "complete", Minute: 90, AddedMinute: 4, HomeGoals: 3, AwayGoals: 2})
	dbCtx.LiveHub.Publish(LeagueMatchChannel(halftime), LiveMatchMessage{Type: "halftime", Minute: 45, AddedMinute: 2, AwayGoals: 1})

	scores, err := GetLeagueLiveScores(dbCtx, leagueId)
	if err != nil || len(scores) != 4 {
		t.Fatalf("got %+v, %v", scores, err)
	}
	byId := map[int64]LeagueLiveScore{}
	for _, s := range scores {
		byId[s.MatchId] = s
	}
	if s := byId[inAddedTime]; s.Status != LiveStatusLive || s.Minute != 90 || s.AddedMinute != 2 || s.HomeGoals != 2 || s.AwayGoals != 2 {
		t.Errorf("entry in added time: got %+v, want live 90+2 at 2-2", s)
	}
	if s := byId[winnerLate]; s.Status != LiveStatusLive || s.Minute != 90 || s.AddedMinute != 1 || s.HomeGoals != 1 || s.AwayGoals != 1 {
		t.Errorf("winner in added time: got %+v, want live 90+1 at 1-1 (stored 2-1)", s)
	}
	if s := byId[finished]; s.Status != LiveStatusFinished || s.Minute != 90 || s.AddedMinute != 4 || s.HomeGoals != 3 {
		t.Errorf("finished after 90+4: got %+v", s)
	}
	if s := byId[halftime]; s.Status != LiveStatusHalftime || s.Minute != 45 || s.AddedMinute != 2 || s.AwayGoals != 1 {
		t.Errorf("half-time after 2 added minutes: got %+v, want halftime 45+2 at 0-1", s)
	}
}

// "Estimate in first-half added time" and "Estimate past the last added minute"
// (division-live-scores "Estimate before the first broadcast message"): with no broadcast state,
// the estimate follows the stored added time and never reports finished; the score is the one as
// of the estimated minute, added minutes included.
func TestGetLeagueLiveScores_EstimateWithAddedTime(t *testing.T) {
	dbCtx, leagueId, addMatch := leagueLiveScoresFixture(t)
	firstHalf := addMatch(12, time.Now().Add(-46*time.Minute), true, 1, 0)
	pastTheEnd := addMatch(12, time.Now().Add(-115*time.Minute), true, 2, 1)
	if _, err := dbCtx.Db.Exec(`UPDATE matches SET first_half_added_time = 3, second_half_added_time = 4 WHERE match_id IN ($1, $2)`, firstHalf, pastTheEnd); err != nil {
		t.Fatalf("setting added time: %s", err)
	}
	if _, err := dbCtx.Db.Exec(`INSERT INTO match_events (match_id, minute, added_minute, event_type, home_goals, away_goals) VALUES
		($1, 45, 1, 0, 1, 0), ($1, 45, 3, 0, 2, 0), ($2, 90, 0, 0, 1, 1), ($2, 90, 4, 0, 2, 1)`, firstHalf, pastTheEnd); err != nil {
		t.Fatalf("seeding events: %s", err)
	}
	dbCtx.LiveHub = nil

	scores, err := GetLeagueLiveScores(dbCtx, leagueId)
	if err != nil || len(scores) != 2 {
		t.Fatalf("got %+v, %v", scores, err)
	}
	byId := map[int64]LeagueLiveScore{}
	for _, s := range scores {
		byId[s.MatchId] = s
	}
	if s := byId[firstHalf]; s.Status != LiveStatusLive || s.Minute != 45 || s.AddedMinute != 2 || s.HomeGoals != 1 {
		t.Errorf("first-half added time: got %+v, want live 45+2 at 1-0 (the 45+3 goal not yet)", s)
	}
	if s := byId[pastTheEnd]; s.Status != LiveStatusLive || s.Minute != 90 || s.AddedMinute != 4 || s.HomeGoals != 2 || s.AwayGoals != 1 {
		t.Errorf("past the last added minute: got %+v, want live 90+4 at 2-1", s)
	}
}
