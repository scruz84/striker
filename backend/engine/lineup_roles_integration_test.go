package engine

import (
	"testing"

	"github.com/scruz84/striker/common"
)

// End to end (openspec add-captain-and-set-piece-takers; needs STRIKER_TEST_DB_DSN): a CPU side's
// kickoff snapshot names its oldest outfield starter as captain with automatic takers, a human's
// saved roles are kept with the snapshot, and the played match's public view shows both captains
// but neither side's takers.
func TestSimulateMatch_RolesAreStoredWithTheSnapshot(t *testing.T) {
	e := newSimEnv(t, "rl1")
	e.exec(`UPDATE teams SET ai_level = $2, owner_id = NULL WHERE team_id = $1`, e.home, common.AiLevelManager)
	matchId := e.leagueMatch(0)

	lineups, err := common.GetLineupsForMatch(e.dbCtx, matchId, &[]int64{e.home, e.away})
	if err != nil {
		t.Fatal(err)
	}
	away := (*lineups)[e.away]
	var outfield []int64
	for _, p := range away.Positions {
		if p.Position != common.GOALKEEPER {
			outfield = append(outfield, p.PlayerId)
		}
	}
	captain, taker := outfield[0], outfield[1]
	humanRoles := common.LineupRoles{CaptainId: &captain, PenaltyTakerId: &taker, FreeKickTakerId: &taker}
	if err := common.SetLineupForMatch(e.dbCtx, matchId, e.away, away.Formation, common.MENTALITY_BALANCED, humanRoles, away.Positions, nil, nil); err != nil {
		t.Fatal(err)
	}

	if _, err := simulateMatch(e.dbCtx, e.weekMatch(matchId)); err != nil {
		t.Fatalf("simulateMatch: %s", err)
	}

	stored, err := common.GetLineupsForMatch(e.dbCtx, matchId, &[]int64{e.home, e.away})
	if err != nil {
		t.Fatal(err)
	}
	cpu := (*stored)[e.home]
	squad, err := common.GetPlayersFromTeamId(e.dbCtx, e.home)
	if err != nil {
		t.Fatal(err)
	}
	want := common.AiCaptain(*squad, cpu.Positions)
	if cpu.CaptainId == nil || want == nil || *cpu.CaptainId != *want {
		t.Errorf("CPU snapshot captain = %v, want its oldest outfield starter %v", cpu.CaptainId, want)
	}
	if cpu.PenaltyTakerId != nil || cpu.FreeKickTakerId != nil {
		t.Errorf("CPU snapshot named takers %+v, want automatic", cpu.LineupRoles)
	}
	human := (*stored)[e.away]
	if human.CaptainId == nil || *human.CaptainId != captain || human.PenaltyTakerId == nil || *human.PenaltyTakerId != taker {
		t.Errorf("human snapshot roles = %+v, want them kept", human.LineupRoles)
	}

	match, err := common.WeekMatchById(e.dbCtx, matchId)
	if err != nil || match == nil {
		t.Fatalf("WeekMatchById: %v", err)
	}
	for name, l := range map[string]*common.Lineup{"home": match.HomeLineup, "away": match.AwayLineup} {
		if l.CaptainId == nil {
			t.Errorf("public %s lineup lost its captain", name)
		}
		if l.PenaltyTakerId != nil || l.FreeKickTakerId != nil {
			t.Errorf("public %s lineup shows its takers: %+v", name, l.LineupRoles)
		}
	}
}
