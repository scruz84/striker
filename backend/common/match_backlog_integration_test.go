package common

import (
	"strings"
	"testing"
	"time"

	"go.opentelemetry.io/otel/attribute"

	"github.com/scruz84/striker/common/metrictest"
)

func backlogOf(t *testing.T, dbCtx *DatabaseContext) map[string]MatchBacklog {
	t.Helper()
	rows, err := CountMatchBacklog(dbCtx)
	if err != nil {
		t.Fatalf("CountMatchBacklog: %s", err)
	}
	byType := map[string]MatchBacklog{}
	for _, b := range rows {
		byType[b.Type] = b
	}
	return byType
}

// TestCountMatchBacklog_OverdueAndInSimulation seeds one league match 10 minutes past kickoff
// (overdue), one 2 minutes past (not yet), one claimed for simulation and one cup tie with no
// away team (a bye, never played), and checks what they add to the backlog. The world already
// holds other unplayed rows, so every figure is a difference from before the seeding.
func TestCountMatchBacklog_OverdueAndInSimulation(t *testing.T) {
	db, dbCtx := openTransfersTestDB(t)
	seedTransfersCountry(t, db, "mb")
	home := seedTransfersTeam(t, db, "Backlog Home FC", "mb")
	var leagueId, away int64
	if err := db.QueryRow(`SELECT league_id FROM teams WHERE team_id = $1`, home).Scan(&leagueId); err != nil {
		t.Fatalf("reading league: %s", err)
	}
	if err := db.QueryRow(`INSERT INTO teams (name, country, league_id, budget) VALUES ('Backlog Away FC', 'mb', $1, 1) RETURNING team_id`, leagueId).Scan(&away); err != nil {
		t.Fatalf("seeding away team: %s", err)
	}

	const season = 702
	var cupId int64
	if err := db.QueryRow(`INSERT INTO cups (country, season, name) VALUES ('mb', $1, 'Backlog Cup') RETURNING cup_id`, season).Scan(&cupId); err != nil {
		t.Fatalf("seeding cup: %s", err)
	}
	// The running engine would simulate seeded due matches, so remove them even if the test fails.
	t.Cleanup(func() {
		db.Exec(`DELETE FROM cup_ties WHERE cup_id = $1`, cupId)
		db.Exec(`DELETE FROM cups WHERE cup_id = $1`, cupId)
		db.Exec(`DELETE FROM matches WHERE league_id = $1`, leagueId)
	})

	before := backlogOf(t, dbCtx)

	insertMatch := func(week int, kickoff time.Time, inSimulation bool) {
		t.Helper()
		if _, err := db.Exec(`INSERT INTO matches (league_id, season, week_number, match_date, home_team_id, away_team_id, in_simulation)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`, leagueId, season, week, kickoff, home, away, inSimulation); err != nil {
			t.Fatalf("inserting match: %s", err)
		}
	}
	now := time.Now()
	insertMatch(1, now.Add(-10*time.Minute), false)
	insertMatch(2, now.Add(-2*time.Minute), false)
	insertMatch(3, now.Add(time.Hour), true)
	if _, err := db.Exec(`INSERT INTO cup_ties (cup_id, round, slot_number, home_team_id, away_team_id, match_date)
		VALUES ($1, 1, 1, $2, NULL, $3)`, cupId, home, now.Add(-10*time.Minute)); err != nil {
		t.Fatalf("inserting bye tie: %s", err)
	}

	after := backlogOf(t, dbCtx)
	if got := after["league"].Overdue - before["league"].Overdue; got != 1 {
		t.Errorf("league overdue grew by %d, want 1", got)
	}
	if got := after["league"].InSimulation - before["league"].InSimulation; got != 1 {
		t.Errorf("league in_simulation grew by %d, want 1", got)
	}
	if got := after["cup"].Overdue - before["cup"].Overdue; got != 0 {
		t.Errorf("a cup bye counted as overdue (cup overdue grew by %d)", got)
	}
	if got := after["friendly"].Overdue - before["friendly"].Overdue; got != 0 {
		t.Errorf("friendly overdue grew by %d, want 0", got)
	}

	// The gauges report the same figures the query returns.
	if err := registerMatchBacklogGauges(dbCtx); err != nil {
		t.Fatalf("registerMatchBacklogGauges: %s", err)
	}
	for _, typ := range []string{"league", "cup", "friendly"} {
		want := backlogOf(t, dbCtx)[typ]
		if got := metrictest.Sum(t, "striker.matches.overdue", attribute.String("type", typ)); got != float64(want.Overdue) {
			t.Errorf("striker.matches.overdue{type=%s} = %v, want %d", typ, got, want.Overdue)
		}
		if got := metrictest.Sum(t, "striker.matches.in_simulation", attribute.String("type", typ)); got != float64(want.InSimulation) {
			t.Errorf("striker.matches.in_simulation{type=%s} = %v, want %d", typ, got, want.InSimulation)
		}
	}
}

// TestCountMatchBacklog_UsesPartialIndexes: each branch reads its table through the unplayed
// partial index and never scans match history. Sequential scans are switched off for the plan
// only, because on a small table the planner rightly prefers them - the check is that the
// indexes are usable for the query, which is what keeps it cheap at 1 000 000 played matches.
func TestCountMatchBacklog_UsesPartialIndexes(t *testing.T) {
	db, _ := openTransfersTestDB(t)
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin: %s", err)
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`SET LOCAL enable_seqscan = off`); err != nil {
		t.Fatalf("disabling seq scans: %s", err)
	}
	rows, err := tx.Query(`EXPLAIN `+CountMatchBacklogStmtSql, int64(MatchOverdueAfter/time.Second))
	if err != nil {
		t.Fatalf("EXPLAIN: %s", err)
	}
	defer rows.Close()
	var plan strings.Builder
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			t.Fatalf("reading plan: %s", err)
		}
		plan.WriteString(line + "\n")
	}
	for _, index := range []string{"matches_unplayed_date_idx", "cup_ties_unplayed_date_idx", "friendly_matches_unplayed_date_idx"} {
		if !strings.Contains(plan.String(), index) {
			t.Errorf("plan does not use %s:\n%s", index, plan.String())
		}
	}
	for _, table := range []string{"matches", "cup_ties", "friendly_matches"} {
		if strings.Contains(plan.String(), "Seq Scan on "+table+" ") {
			t.Errorf("plan scans %s sequentially:\n%s", table, plan.String())
		}
	}
}
