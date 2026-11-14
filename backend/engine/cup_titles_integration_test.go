package engine

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/scruz84/striker/common"
)

// TestAdvanceCup_RecordsEachTitleInItsSeason covers team-history "Domestic cup title" and
// "International cup title": a completed domestic cup is recorded under its own season, a regional
// cup under the league season the champion's country is playing (never the edition number), and
// the World Cup knockout writes no history at all.
func TestAdvanceCup_RecordsEachTitleInItsSeason(t *testing.T) {
	db, dbCtx := openReplicaTestDb(t)
	scan := func(query string, args ...any) sql.NullInt64 {
		t.Helper()
		var v sql.NullInt64
		if err := db.QueryRow(query, args...).Scan(&v); err != nil {
			t.Fatalf("%s: %s", query, err)
		}
		return v
	}
	country := uniqueCountryCode()
	if _, err := db.Exec(`INSERT INTO countries (name, current_season) VALUES ($1, 6)`, country); err != nil {
		t.Fatal(err)
	}
	team := func(name string) int64 {
		return scan(`INSERT INTO teams (name, country) VALUES ($1, $2) RETURNING team_id`, country+" "+name, country).Int64
	}
	// finishedCup creates a cup whose only round (the final) was won by winner well outside its live
	// window, and advances it.
	finishedCup := func(cup common.Cup, winner, loser int64) common.Cup {
		t.Helper()
		var country, region any
		if cup.Country != "" {
			country = cup.Country
		} else {
			region = cup.Region
		}
		cup.CupId = scan(`INSERT INTO cups (country, region, season, name) VALUES ($1, $2, $3, 'Cup') RETURNING cup_id`, country, region, cup.Season).Int64
		if _, err := db.Exec(`INSERT INTO cup_ties (cup_id, round, slot_number, home_team_id, away_team_id, match_date, played, winner_team_id)
			VALUES ($1, 1, 1, $2, $3, $4, true, $2)`, cup.CupId, winner, loser, time.Now().Add(-3*time.Hour)); err != nil {
			t.Fatal(err)
		}
		if err := advanceCup(dbCtx, cup); err != nil {
			t.Fatalf("advanceCup: %s", err)
		}
		if completed := scan(`SELECT CASE WHEN completed THEN 1 ELSE 0 END FROM cups WHERE cup_id = $1`, cup.CupId).Int64; completed != 1 {
			t.Fatalf("cup %d did not complete", cup.CupId)
		}
		return cup
	}

	regionalChampion, domesticChampion, worldChampion, loser := team("regional"), team("domestic"), team("world"), team("loser")
	regional := finishedCup(common.Cup{Region: fmt.Sprintf("test-%d", time.Now().UnixNano()%1e12), Season: 2}, regionalChampion, loser)
	finishedCup(common.Cup{Country: country, Season: 4}, domesticChampion, loser)
	finishedCup(common.Cup{Region: worldCupRegion, Season: 3}, worldChampion, loser)

	if got := scan(`SELECT international_cup_id FROM season_history WHERE team_id = $1 AND season = 6`, regionalChampion); !got.Valid || got.Int64 != regional.CupId {
		t.Errorf("regional champion's season 6 international_cup_id = %v, want %d", got, regional.CupId)
	}
	if n := scan(`SELECT count(*) FROM season_history WHERE team_id = $1 AND season = 2`, regionalChampion).Int64; n != 0 {
		t.Errorf("regional champion has a season-2 (edition number) history row")
	}
	if n := scan(`SELECT count(*) FROM season_history WHERE team_id = $1 AND season = 4 AND cup_result = 'Champion' AND international_cup_id IS NULL`, domesticChampion).Int64; n != 1 {
		t.Errorf("domestic champion's season-4 cup win was not recorded")
	}
	if n := scan(`SELECT count(*) FROM season_history WHERE team_id = $1`, worldChampion).Int64; n != 0 {
		t.Errorf("the World Cup knockout wrote %d history rows, want none", n)
	}
}
