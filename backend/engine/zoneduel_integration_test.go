package engine

import (
	"math"
	"testing"

	"github.com/scruz84/striker/common"
)

// TestProjectZoneDuel_MatchesARealSimulation runs against a real Postgres (STRIKER_TEST_DB_DSN;
// skipped when unset): the projection of two lineups equals the zone ratings the first minute of a
// real simulation of those same lineups records.
func TestProjectZoneDuel_MatchesARealSimulation(t *testing.T) {
	dbCtx, db, seed := matchViewerParityEnv(t)
	home, away := seed("zpd1")
	// give every player different skills so the comparison is not trivially uniform
	if _, err := db.Exec(`UPDATE players SET defense = 30 + (player_id * 7) % 60, midfield = 30 + (player_id * 11) % 60,
		shot = 30 + (player_id * 13) % 60, "lateral" = 30 + (player_id * 17) % 60, goal_keeper = 30 + (player_id * 19) % 60
		WHERE team_id IN ($1, $2)`, home, away); err != nil {
		t.Fatalf("varying skills: %s", err)
	}

	homeLineup, err := common.BestLineupForTeam(dbCtx, home)
	if err != nil {
		t.Fatal(err)
	}
	awayLineup, err := common.BestLineupForTeam(dbCtx, away)
	if err != nil {
		t.Fatal(err)
	}
	lineups := map[int64]*common.Lineup{home: homeLineup, away: awayLineup}

	simulation, err := runMatchSimulation(dbCtx, 1, 0, home, away, &lineups, nil, nil, true, nil)
	if err != nil {
		t.Fatalf("runMatchSimulation: %s", err)
	}
	if len(*simulation.MatchInstants) == 0 {
		t.Fatal("the simulation recorded no instants")
	}
	first := (*simulation.MatchInstants)[0]

	homeSquad, err := common.GetPlayersFromTeamId(dbCtx, home)
	if err != nil {
		t.Fatal(err)
	}
	awaySquad, err := common.GetPlayersFromTeamId(dbCtx, away)
	if err != nil {
		t.Fatal(err)
	}
	projection := ProjectZoneDuel(
		ZoneDuelSide{Lineup: homeLineup, Squad: *homeSquad},
		ZoneDuelSide{Lineup: awayLineup, Squad: *awaySquad},
	)

	same := func(name string, got, want float32) {
		t.Helper()
		if math.Abs(float64(got-want)) > 0.0001 {
			t.Errorf("%s: projected %v, simulated %v", name, got, want)
		}
	}
	for _, c := range []struct {
		name      string
		got, want common.TeamQualification
	}{{"home", projection.Own, first.HomeQua}, {"away", projection.Opponent, first.AwayQua}} {
		same(c.name+" left defense", c.got.LeftDefense, c.want.LeftDefense)
		same(c.name+" defense", c.got.Defense, c.want.Defense)
		same(c.name+" right defense", c.got.RightDefense, c.want.RightDefense)
		same(c.name+" left midfield", c.got.LeftMidfield, c.want.LeftMidfield)
		same(c.name+" midfield", c.got.Midfield, c.want.Midfield)
		same(c.name+" right midfield", c.got.RightMidfield, c.want.RightMidfield)
		same(c.name+" left attack", c.got.LeftAttack, c.want.LeftAttack)
		same(c.name+" attack", c.got.Attack, c.want.Attack)
		same(c.name+" right attack", c.got.RightAttach, c.want.RightAttach)
	}
	if first.HomeQua.Midfield == 0 {
		t.Fatal("the simulated midfield is zero: the comparison would be vacuous")
	}
}
