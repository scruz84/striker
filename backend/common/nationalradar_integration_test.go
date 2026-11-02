package common

import "testing"

// TestGetNationalRadar_BuildsPoolsFromTheDatabase runs against a real Postgres (see
// openTransfersTestDB): a country's pool is its NationalSquadPoolSize best players by skill on a
// club's senior squad, so the 45th-best is out and the best is in, free agents are never in it,
// and the snapshot is cached until reset.
func TestGetNationalRadar_BuildsPoolsFromTheDatabase(t *testing.T) {
	db, dbCtx := openTransfersTestDB(t)
	seedTransfersCountry(t, db, "zk")
	// The test database persists across runs: an earlier run's 'zk' players (rated up to 99 by
	// the cache check below) would still outrank this run's best, so drop them from the pool.
	if _, err := db.Exec(`UPDATE players SET birth_country = NULL WHERE birth_country = 'zk'`); err != nil {
		t.Fatal(err)
	}
	team := seedTransfersTeam(t, db, "Radar Team", "zk")

	ids := make([]int64, 0)
	for i := 0; i < NationalSquadPoolSize+5; i++ {
		id := seedTransfersPlayer(t, db, team, "zk")
		// Distinct overall ratings: player i is i points above the baseline of 50 in every skill.
		if _, err := db.Exec(`UPDATE players SET goal_keeper = $2, defense = $2, "lateral" = $2, pass = $2, midfield = $2,
			dribbling = $2, shot = $2 WHERE player_id = $1`, id, 50+i); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	freeAgent := seedTransfersPlayer(t, db, team, "zk")
	if _, err := db.Exec(`UPDATE players SET team_id = NULL, shirt_number = NULL, goal_keeper = 99, defense = 99, "lateral" = 99, pass = 99,
		midfield = 99, dribbling = 99, shot = 99 WHERE player_id = $1`, freeAgent); err != nil {
		t.Fatal(err)
	}

	ResetNationalRadarCache()
	radar, err := GetNationalRadar(dbCtx)
	if err != nil {
		t.Fatalf("GetNationalRadar: %s", err)
	}
	best := ids[len(ids)-1]
	if entry, ok := radar.Lookup(best); !ok || entry.Country != "zk" || entry.Rank != 0 {
		t.Errorf("the country's best player should be rank 0 of its pool, got %+v, %v", entry, ok)
	}
	if _, ok := radar.Lookup(ids[0]); ok {
		t.Error("the weakest of 45 players is outside a 40-player pool")
	}
	if _, ok := radar.Lookup(freeAgent); ok {
		t.Error("a free agent is never in a national-team pool")
	}

	// Cached: a change in the database isn't seen until the cache is reset.
	if _, err := db.Exec(`UPDATE players SET goal_keeper = 99, defense = 99, "lateral" = 99, pass = 99, midfield = 99, dribbling = 99,
		shot = 99 WHERE player_id = $1`, ids[0]); err != nil {
		t.Fatal(err)
	}
	cached, _ := GetNationalRadar(dbCtx)
	if _, ok := cached.Lookup(ids[0]); ok {
		t.Error("the radar should be served from its cache")
	}
	ResetNationalRadarCache()
	fresh, _ := GetNationalRadar(dbCtx)
	if _, ok := fresh.Lookup(ids[0]); !ok {
		t.Error("after a reset the radar should see the player who just became the country's best")
	}
}
