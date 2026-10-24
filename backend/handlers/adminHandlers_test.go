package handlers

import (
	"encoding/json"
	"testing"

	"github.com/scruz84/striker/common"
	"github.com/scruz84/striker/engine"
)

func TestBuildFinishStuckMatchesResponse_Shape(t *testing.T) {
	response := buildFinishStuckMatchesResponse(
		engine.DefaultResultsResult{LeagueMatches: 2, Friendlies: 1, CupTies: 3},
		engine.StuckSimulationsResult{LeagueMatches: 4, Friendlies: 5, CupTies: 6},
		common.SimulationHealth{FailingCounts: common.FixtureKindCounts{LeagueMatches: 7, Friendlies: 8, CupTies: 9}},
	)

	raw, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("marshaling: %s", err)
	}
	var got map[string]map[string]int
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshaling %s: %s", raw, err)
	}
	want := map[string]map[string]int{
		"resimulated":  {"leagueMatches": 4, "friendlies": 5, "cupTies": 6},
		"defaulted":    {"leagueMatches": 2, "friendlies": 1, "cupTies": 3},
		"stillFailing": {"leagueMatches": 7, "friendlies": 8, "cupTies": 9},
	}
	for group, counts := range want {
		for kind, n := range counts {
			if got[group][kind] != n {
				t.Errorf("%s.%s = %d, want %d (response %s)", group, kind, got[group][kind], n, raw)
			}
		}
	}
}
