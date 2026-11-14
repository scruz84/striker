package engine

import (
	"fmt"
	"testing"
	"time"

	"github.com/scruz84/striker/common"
)

// A tie simulated by the real pass (SimulateCupTies, not simulateCupTie on a hand-built CupTie)
// knows its round: winning round 2 pays CupPrizePerRound * 2 under "Cup round 2 prize money"
// (issue #398 - GetSimulableCupTies never loaded the round, so every prize was 0). Needs
// STRIKER_TEST_DB_DSN.
func TestSimulateCupTies_PaysTheRoundPrize(t *testing.T) {
	db, dbCtx := openReplicaTestDb(t)
	scan := func(query string, args ...any) int64 {
		t.Helper()
		var v int64
		if err := db.QueryRow(query, args...).Scan(&v); err != nil {
			t.Fatalf("%s: %s", query, err)
		}
		return v
	}

	country := uniqueCountryCode()
	if _, err := db.Exec(`INSERT INTO countries (name, current_season) VALUES ($1, 1)`, country); err != nil {
		t.Fatalf("inserting country: %s", err)
	}
	home := scan(`INSERT INTO teams (name, country) VALUES ('Home', $1) RETURNING team_id`, country)
	away := scan(`INSERT INTO teams (name, country) VALUES ('Away', $1) RETURNING team_id`, country)
	common.CreatePlayers(dbCtx, home, country)
	common.CreatePlayers(dbCtx, away, country)
	cupId := scan(`INSERT INTO cups (country, season, name) VALUES ($1, 1, 'Cup') RETURNING cup_id`, country)
	kickoff := time.Now().Add(-2 * time.Hour)
	const round = 2
	tieId, err := common.CreateCupTie(dbCtx, cupId, round, 0, &home, &away, &kickoff)
	if err != nil {
		t.Fatalf("CreateCupTie: %s", err)
	}

	SimulateCupTies(dbCtx)

	if played := scan(`SELECT count(*) FROM cup_ties WHERE tie_id = $1 AND played`, tieId); played != 1 {
		t.Fatalf("tie %d not played", tieId)
	}
	winner := scan(`SELECT winner_team_id FROM cup_ties WHERE tie_id = $1`, tieId)
	var amount int64
	var description string
	if err := db.QueryRow(`SELECT amount, description FROM team_transactions WHERE team_id = $1 AND reason = $2`,
		winner, common.TRANSACTION_PRIZE_MONEY).Scan(&amount, &description); err != nil {
		t.Fatalf("reading the winner's round prize: %s", err)
	}
	if want := int64(CupPrizePerRound * round); amount != want {
		t.Errorf("round prize = %d, want %d", amount, want)
	}
	if want := fmt.Sprintf("Cup round %d prize money", round); description != want {
		t.Errorf("round prize description = %q, want %q", description, want)
	}
}
