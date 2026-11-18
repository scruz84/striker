package common

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// TestTeamCupTitlesMigration covers team-history "Existing history on upgrade": init111 clears a
// cup_result written under a regional cup's edition number, keeps a real domestic win, deletes a
// row that only existed for the wrong mark, and records the regional title in the league season
// its final was played in. Re-applying it changes nothing.
func TestTeamCupTitlesMigration(t *testing.T) {
	db, _ := openTransfersTestDB(t)
	w := seedSeasonHistoryWorld(t, db)
	ctx := context.Background()
	conn, err := db.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	region := fmt.Sprintf("test-%d", time.Now().UnixNano()%1e12)
	// A won its country's season-1 cup (a real domestic title).
	w.exec(`INSERT INTO cups (country, season, name, completed, champion_team_id) VALUES ($1, 1, 'Cup', true, $2)`, w.country, w.a)
	w.exec(`UPDATE season_history SET cup_result = 'Champion' WHERE team_id = $1 AND season = 1`, w.a)
	// B won edition 1 of a regional cup during season 2: its final comes after B's season-2 match.
	finalAt := time.Now().Add(-1 * time.Hour)
	w.exec(`INSERT INTO matches (league_id, season, match_date, home_team_id, away_team_id, played) VALUES ($1, 2, $2, $3, $4, true)`,
		w.upper, finalAt.Add(-24*time.Hour), w.b, w.d)
	cupId := w.scanInt(`INSERT INTO cups (region, season, name, completed, champion_team_id) VALUES ($1, 1, 'European Cup', true, $2) RETURNING cup_id`,
		region, w.b).Int64
	w.exec(`INSERT INTO cup_ties (cup_id, round, slot_number, home_team_id, away_team_id, match_date, played, winner_team_id)
		VALUES ($1, 1, 1, $2, $3, $4, true, $2)`, cupId, w.b, w.a, finalAt)
	// The old code wrote that title as 'Champion' under season 1 (the edition number), and a title
	// of C under its edition 7 created a row of its own.
	w.exec(`UPDATE season_history SET cup_result = 'Champion' WHERE team_id = $1 AND season = 1`, w.b)
	w.exec(`INSERT INTO season_history (team_id, season, cup_result) VALUES ($1, 7, 'Champion')`, w.c)

	check := func(when string) {
		t.Helper()
		if got := w.scanInt(`SELECT count(*) FROM season_history WHERE team_id = $1 AND season = 1 AND cup_result = 'Champion'`, w.a).Int64; got != 1 {
			t.Errorf("%s: A lost its real season-1 cup win", when)
		}
		if got := w.scanInt(`SELECT count(*) FROM season_history WHERE team_id = $1 AND season = 1 AND cup_result IS NULL AND final_position = 2`, w.b).Int64; got != 1 {
			t.Errorf("%s: B's season 1 should keep its league line and lose the wrong cup mark", when)
		}
		if got := w.scanInt(`SELECT international_cup_id FROM season_history WHERE team_id = $1 AND season = 2`, w.b); !got.Valid || got.Int64 != cupId {
			t.Errorf("%s: B's season 2 international_cup_id = %v, want %d", when, got, cupId)
		}
		if got := w.scanInt(`SELECT count(*) FROM season_history WHERE team_id = $1 AND season = 7`, w.c).Int64; got != 0 {
			t.Errorf("%s: C's row that only held the wrong mark should be deleted", when)
		}
		history, err := GetSeasonHistory(&DatabaseContext{Db: db}, w.b)
		if err != nil {
			t.Fatalf("GetSeasonHistory: %s", err)
		}
		var found bool
		for _, e := range history {
			if e.Season == 2 && e.InternationalCup != nil {
				found = e.InternationalCup.CupId == cupId && e.InternationalCup.Region == region && e.InternationalCup.Edition == 1
			}
			if e.Season == 1 && (e.LeagueTier == nil || *e.LeagueTier != 1) {
				t.Errorf("%s: B's season 1 leagueTier = %v, want 1", when, e.LeagueTier)
			}
		}
		if !found {
			t.Errorf("%s: GetSeasonHistory has no edition-1 title for B in season 2: %+v", when, history)
		}
	}

	if err := applySchemaFile(ctx, conn, "history/init111_team_cup_titles.sql"); err != nil {
		t.Fatalf("applying migration: %s", err)
	}
	check("after the migration")
	if err := applySchemaFile(ctx, conn, "history/init111_team_cup_titles.sql"); err != nil {
		t.Fatalf("re-applying migration: %s", err)
	}
	check("after re-applying")
}
