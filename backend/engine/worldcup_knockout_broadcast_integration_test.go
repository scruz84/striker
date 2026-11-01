package engine

import (
	"math/rand"
	"testing"
	"time"

	"github.com/scruz84/striker/common"
)

// TestAdvanceWorldCup_WaitsForTheLastGroupBroadcast: the knockout bracket (and the call-ups saved
// with its first round) used to be created as soon as the last group match was simulated, up to
// ~2 hours before its broadcast ended, revealing who qualified (openspec
// delay-world-cup-knockout-draw). Runs against a real Postgres (STRIKER_TEST_DB_DSN; skipped when
// unset).
func TestAdvanceWorldCup_WaitsForTheLastGroupBroadcast(t *testing.T) {
	db, dbCtx := openReplicaTestDb(t)
	scan := func(query string, args ...any) int64 {
		t.Helper()
		var v int64
		if err := db.QueryRow(query, args...).Scan(&v); err != nil {
			t.Fatalf("%s: %s", query, err)
		}
		return v
	}

	worldCupId, err := common.CreateWorldCup(dbCtx, 9000+rand.Intn(100000), "Broadcast World Cup")
	if err != nil {
		t.Fatalf("CreateWorldCup: %s", err)
	}
	groupId, err := common.CreateWorldCupGroup(dbCtx, worldCupId, "A")
	if err != nil {
		t.Fatalf("CreateWorldCupGroup: %s", err)
	}
	teams := make([]int64, 2)
	for i := range teams {
		country := uniqueCountryCode()
		if _, err := db.Exec(`INSERT INTO countries (name) VALUES ($1) ON CONFLICT DO NOTHING`, country); err != nil {
			t.Fatalf("inserting country: %s", err)
		}
		teams[i] = scan(`INSERT INTO teams (name, country, is_national_team) VALUES ($1, $2, true) RETURNING team_id`, "NT "+country, country)
		if err := common.AddWorldCupGroupTeam(dbCtx, groupId, teams[i]); err != nil {
			t.Fatalf("AddWorldCupGroupTeam: %s", err)
		}
	}
	// Simulated, but kicked off 40 minutes ago: still on air.
	matchId := scan(`INSERT INTO friendly_matches (home_team_id, away_team_id, match_date, played, home_goals, away_goals)
		VALUES ($1, $2, now() - interval '40 minutes', true, 2, 1) RETURNING match_id`, teams[0], teams[1])
	if err := common.LinkWorldCupGroupMatch(dbCtx, groupId, 1, matchId); err != nil {
		t.Fatalf("LinkWorldCupGroupMatch: %s", err)
	}

	knockoutTies := func() int64 {
		return scan(`SELECT count(*) FROM cup_ties WHERE home_team_id = ANY($1) OR away_team_id = ANY($1)`, teams)
	}
	if err := advanceWorldCup(dbCtx, worldCupId); err != nil {
		t.Fatalf("advanceWorldCup: %s", err)
	}
	if n := knockoutTies(); n != 0 {
		t.Fatalf("knockout created with the last group match still on air: %d ties", n)
	}
	if stage := scan(`SELECT CASE WHEN stage = 'groups' AND knockout_cup_id IS NULL THEN 1 ELSE 0 END FROM world_cups WHERE world_cup_id = $1`, worldCupId); stage != 1 {
		t.Fatalf("World Cup %d left its group stage while the last group match was on air", worldCupId)
	}

	if _, err := db.Exec(`UPDATE friendly_matches SET match_date = $2 WHERE match_id = $1`, matchId, time.Now().Add(-common.LiveBroadcastWindow-time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := advanceWorldCup(dbCtx, worldCupId); err != nil {
		t.Fatalf("advanceWorldCup: %s", err)
	}
	if cups := scan(`SELECT count(DISTINCT cup_id) FROM cup_ties WHERE home_team_id = ANY($1) OR away_team_id = ANY($1)`, teams); cups != 1 {
		t.Errorf("after the last broadcast: %d knockout cups, want 1", cups)
	}
	// The World Cup must point at the cup its knockout ties belong to (SetWorldCupKnockoutStarted
	// once swapped its arguments and updated the wrong row).
	if ok := scan(`SELECT count(*) FROM world_cups wc WHERE wc.world_cup_id = $1 AND wc.stage = 'knockout' AND wc.knockout_cup_id = `+
		`(SELECT DISTINCT cup_id FROM cup_ties WHERE home_team_id = ANY($2) OR away_team_id = ANY($2))`, worldCupId, teams); ok != 1 {
		t.Errorf("World Cup %d should be in its knockout stage, pointing at its knockout cup", worldCupId)
	}
}
