package engine

import (
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/scruz84/striker/common"
)

// openspec add-home-advantage: the weather and the crowd are drawn once before kickoff; the same
// crowd sets the home advantage, the gate revenue and the report, and a failed save books nothing.

// failInstantInserts makes every insert into instantsTable for row id fail until the test ends, so
// the result save of that match rolls back after the simulation has run.
func failInstantInserts(e *simEnv, instantsTable string, idColumn string, id int64) func() {
	e.t.Helper()
	name := fmt.Sprintf("home_adv_fail_%s_%d", instantsTable, id)
	e.exec(`CREATE OR REPLACE FUNCTION ` + name + `() RETURNS trigger AS $$ BEGIN RAISE EXCEPTION 'forced save failure'; END $$ LANGUAGE plpgsql`)
	e.exec(fmt.Sprintf(`CREATE TRIGGER %s BEFORE INSERT ON %s FOR EACH ROW WHEN (NEW.%s = %d) EXECUTE FUNCTION %s()`,
		name, instantsTable, idColumn, id, name))
	drop := func() {
		_, _ = e.db.Exec(fmt.Sprintf(`DROP TRIGGER IF EXISTS %s ON %s`, name, instantsTable))
		_, _ = e.db.Exec(`DROP FUNCTION IF EXISTS ` + name + `()`)
	}
	e.t.Cleanup(drop)
	return drop
}

// "Crowd drawn before kickoff" and "Retried after a failed save": the first attempt's save fails
// and books no gate; the retry's stored attendance is the one in the gate revenue and the one the
// home advantage was computed from.
func TestSimulateMatch_CrowdDrawnBeforeKickoffAndRedrawnAfterAFailedSave(t *testing.T) {
	e := newSimEnv(t, "zh1")
	matchId := e.leagueMatch(0)
	stopFailing := failInstantInserts(e, "match_instants", "match_id", matchId)

	m := *e.weekMatch(matchId)
	if _, err := simulateMatch(e.dbCtx, &m); err == nil {
		t.Fatal("the first attempt's save was expected to fail")
	}
	if gate := e.transactions(e.home, common.TRANSACTION_TICKET_REVENUE); gate != 0 {
		t.Fatalf("a failed save booked %d gate receipts, want none", gate)
	}
	if played := e.scan(`SELECT count(*) FROM matches WHERE match_id = $1 AND played`, matchId); played != 0 {
		t.Fatal("the match was marked played although its save failed")
	}

	stopFailing()
	m = *e.weekMatch(matchId)
	sim, err := simulateMatch(e.dbCtx, &m)
	if err != nil {
		t.Fatalf("retry: %s", err)
	}

	var attendance int
	var weather string
	if err := e.db.QueryRow(`SELECT attendance, weather FROM matches WHERE match_id = $1`, matchId).Scan(&attendance, &weather); err != nil {
		t.Fatalf("reading the report flavor: %s", err)
	}
	if attendance <= 0 || weather == "" {
		t.Fatalf("expected a stored crowd and weather, got %d / %q", attendance, weather)
	}
	stadium, err := common.GetStadium(e.dbCtx, e.home)
	if err != nil {
		t.Fatalf("GetStadium: %s", err)
	}

	var amount int64
	var description string
	if err := e.db.QueryRow(`SELECT amount, description FROM team_transactions WHERE team_id = $1 AND reason = $2`,
		e.home, common.TRANSACTION_TICKET_REVENUE).Scan(&amount, &description); err != nil {
		t.Fatalf("reading the one gate receipt: %s", err)
	}
	if want := TicketRevenue(attendance, stadium.TicketPrice); amount != want {
		t.Errorf("gate receipt %d, want %d for the reported %d spectators", amount, want, attendance)
	}
	if !strings.Contains(description, fmt.Sprintf("(%d attendance)", attendance)) {
		t.Errorf("gate description %q does not name the reported %d spectators", description, attendance)
	}
	want := homeAdvantageMultiplier(attendance, stadium.Capacity)
	if math.Abs(float64(sim.HomeAdvantage-want)) > 1e-6 || sim.HomeAdvantage <= 1 {
		t.Errorf("home advantage %.4f, want %.4f for %d of %d seats", sim.HomeAdvantage, want, attendance, stadium.Capacity)
	}
}

// A cup tie and a friendly credit the crowd their report shows - the one drawn before kickoff.
func TestSimulateCupTieAndFriendly_CreditTheCrowdDrawnBeforeKickoff(t *testing.T) {
	e := newSimEnv(t, "zh2")
	stadium, err := common.GetStadium(e.dbCtx, e.home)
	if err != nil {
		t.Fatalf("GetStadium: %s", err)
	}

	cupId, tieId := e.cupTie(0)
	homeId, awayId := e.home, e.away
	kickoff := time.Now().Add(-3 * time.Hour)
	if err := simulateCupTie(e.dbCtx, &common.CupTie{TieId: tieId, CupId: cupId, Round: 2, HomeTeamId: &homeId, AwayTeamId: &awayId, MatchDate: &kickoff}); err != nil {
		t.Fatalf("simulateCupTie: %s", err)
	}
	tieAttendance := e.scan(`SELECT attendance FROM cup_ties WHERE tie_id = $1`, tieId)
	if gate := e.scan(`SELECT COALESCE(SUM(amount), 0) FROM team_transactions WHERE team_id = $1 AND reason = $2`, e.home, common.TRANSACTION_TICKET_REVENUE); gate != TicketRevenue(int(tieAttendance), stadium.TicketPrice) {
		t.Errorf("cup gate %d, want %d for the reported %d spectators", gate, TicketRevenue(int(tieAttendance), stadium.TicketPrice), tieAttendance)
	}

	friendlyId := e.scan(`INSERT INTO friendly_matches (home_team_id, away_team_id, match_date) VALUES ($1, $2, now() - interval '3 hours') RETURNING match_id`, e.home, e.away)
	friendly, err := common.GetFriendlyMatchById(e.dbCtx, friendlyId)
	if err != nil {
		t.Fatalf("GetFriendlyMatchById: %s", err)
	}
	if err := simulateFriendly(e.dbCtx, friendly); err != nil {
		t.Fatalf("simulateFriendly: %s", err)
	}
	friendlyAttendance := e.scan(`SELECT attendance FROM friendly_matches WHERE match_id = $1`, friendlyId)
	total := TicketRevenue(int(friendlyAttendance), stadium.TicketPrice)
	homeShare, awayShare := splitRevenueEvenly(total)
	if got := e.scan(`SELECT COALESCE(SUM(amount), 0) FROM team_transactions WHERE team_id = $1 AND reason = $2 AND description LIKE 'Friendly%'`, e.home, common.TRANSACTION_TICKET_REVENUE); got != homeShare {
		t.Errorf("friendly home share %d, want %d for the reported %d spectators", got, homeShare, friendlyAttendance)
	}
	if got := e.scan(`SELECT COALESCE(SUM(amount), 0) FROM team_transactions WHERE team_id = $1 AND reason = $2`, e.away, common.TRANSACTION_TICKET_REVENUE); got != awayShare {
		t.Errorf("friendly away share %d, want %d", got, awayShare)
	}
}

// World Cup group friendlies and World Cup knockout ties are recognised as neutral fixtures.
func TestWorldCupFixturesAreNeutral(t *testing.T) {
	e := newSimEnv(t, "zh3")
	groupFriendly := e.scan(`INSERT INTO friendly_matches (home_team_id, away_team_id, match_date) VALUES ($1, $2, now()) RETURNING match_id`, e.home, e.away)
	ordinaryFriendly := e.scan(`INSERT INTO friendly_matches (home_team_id, away_team_id, match_date) VALUES ($1, $2, now()) RETURNING match_id`, e.home, e.away)
	knockoutCup := e.scan(`INSERT INTO cups (country, season, name) VALUES ($1, 1, 'WC knockout') RETURNING cup_id`, e.country)
	ordinaryCup := e.scan(`INSERT INTO cups (country, season, name) VALUES ($1, 1, 'Cup') RETURNING cup_id`, e.country)
	worldCupId := e.scan(`INSERT INTO world_cups (season, name, knockout_cup_id, completed) VALUES (1, 'World Cup', $1, true) RETURNING world_cup_id`, knockoutCup)
	groupId := e.scan(`INSERT INTO world_cup_groups (world_cup_id, group_name) VALUES ($1, 'A') RETURNING group_id`, worldCupId)
	e.exec(`INSERT INTO world_cup_group_matches (group_id, round, match_id) VALUES ($1, 1, $2)`, groupId, groupFriendly)
	t.Cleanup(func() {
		_, _ = e.db.Exec(`DELETE FROM world_cup_group_matches WHERE group_id = $1`, groupId)
		_, _ = e.db.Exec(`DELETE FROM world_cup_groups WHERE group_id = $1`, groupId)
		_, _ = e.db.Exec(`DELETE FROM world_cups WHERE world_cup_id = $1`, worldCupId)
	})

	for _, c := range []struct {
		name  string
		check func() (bool, error)
		want  bool
	}{
		{"group friendly", func() (bool, error) { return common.IsWorldCupGroupMatch(e.dbCtx, groupFriendly) }, true},
		{"ordinary friendly", func() (bool, error) { return common.IsWorldCupGroupMatch(e.dbCtx, ordinaryFriendly) }, false},
		{"knockout cup", func() (bool, error) { return common.IsWorldCupKnockoutCup(e.dbCtx, knockoutCup) }, true},
		{"ordinary cup", func() (bool, error) { return common.IsWorldCupKnockoutCup(e.dbCtx, ordinaryCup) }, false},
	} {
		got, err := c.check()
		if err != nil {
			t.Fatalf("%s: %s", c.name, err)
		}
		if got != c.want {
			t.Errorf("%s: neutral = %v, want %v", c.name, got, c.want)
		}
	}
}
