package common

import "testing"

func projectionSquad() []*Player {
	squad := []*Player{keeperPlayer(1, 90, 80, 28)}
	for id := int64(2); id <= 11; id++ {
		squad = append(squad, fieldPlayer(id, 70, 80, 28))
	}
	squad = append(squad, keeperPlayer(12, 60, 80, 28), fieldPlayer(13, 65, 80, 28), fieldPlayer(14, 55, 80, 28))
	return squad
}

func projectionLineup(squad []*Player) *Lineup {
	formation, positions := SelectBestFormationAndLineup(squad[:11], AiLevelDirector, nil)
	return &Lineup{Formation: formation, Positions: orderedLineupPositions(positions)}
}

func TestProjectKickoffReplacements_InjuredStarterGetsTheBestAvailableReplacement(t *testing.T) {
	squad := projectionSquad()
	lineup := projectionLineup(squad)
	var injured LineupPosition
	for _, p := range lineup.Positions {
		if p.Position != GOALKEEPER {
			injured = p
			break
		}
	}
	before := append([]LineupPosition(nil), lineup.Positions...)

	got := ProjectKickoffReplacements(lineup, squad, map[int64]bool{injured.PlayerId: true})

	if len(got) != 1 || got[0].PlayerOutId != injured.PlayerId || got[0].Position != injured.Position {
		t.Fatalf("expected one replacement for the injured starter %+v, got %v", injured, got)
	}
	if got[0].PlayerInId != 13 {
		t.Fatalf("expected the best available outfield reserve (13), got %d", got[0].PlayerInId)
	}
	for i := range before {
		if lineup.Positions[i] != before[i] {
			t.Fatalf("expected the saved lineup unchanged, got %v", lineup.Positions)
		}
	}
}

func TestProjectKickoffReplacements_NothingUnavailableMeansNoProjection(t *testing.T) {
	squad := projectionSquad()
	lineup := projectionLineup(squad)

	if got := ProjectKickoffReplacements(lineup, squad, nil); len(got) != 0 {
		t.Fatalf("expected no replacements, got %v", got)
	}
	if got := ProjectKickoffReplacements(lineup, squad, map[int64]bool{14: true}); len(got) != 0 {
		t.Fatalf("expected no replacements for an unavailable bench player, got %v", got)
	}
	if got := ProjectKickoffReplacements(nil, squad, map[int64]bool{2: true}); got != nil {
		t.Fatalf("expected nil for no lineup, got %v", got)
	}
}

func TestProjectKickoffReplacements_PairsEachInjuredStarterWithADifferentReplacement(t *testing.T) {
	squad := projectionSquad()
	lineup := projectionLineup(squad)
	unavailable := map[int64]bool{}
	for _, p := range lineup.Positions {
		if p.Position != GOALKEEPER && len(unavailable) < 2 {
			unavailable[p.PlayerId] = true
		}
	}

	got := ProjectKickoffReplacements(lineup, squad, unavailable)

	if len(got) != 2 {
		t.Fatalf("expected two replacements, got %v", got)
	}
	if got[0].PlayerInId == got[1].PlayerInId || unavailable[got[0].PlayerInId] || unavailable[got[1].PlayerInId] {
		t.Fatalf("expected two distinct, available replacements, got %v", got)
	}
}

func TestProjectKickoffReplacements_MatchesTheEnginesRefill(t *testing.T) {
	squad := projectionSquad()
	lineup := projectionLineup(squad)
	unavailable := map[int64]bool{}
	for _, p := range lineup.Positions {
		if p.Position != GOALKEEPER && len(unavailable) < 3 {
			unavailable[p.PlayerId] = true
		}
	}
	byPosition := map[int][]int64{}
	for _, p := range lineup.Positions {
		byPosition[p.Position] = append(byPosition[p.Position], p.PlayerId)
	}
	ExcludeUnavailableAndBackfill(&byPosition, squad, unavailable)
	onPitch := map[int64]bool{}
	for _, ids := range byPosition {
		for _, id := range ids {
			onPitch[id] = true
		}
	}

	for _, r := range ProjectKickoffReplacements(lineup, squad, unavailable) {
		if !onPitch[r.PlayerInId] || onPitch[r.PlayerOutId] {
			t.Fatalf("expected the projection to name who the refill fields, got %+v", r)
		}
	}
}

func TestProjectKickoffReplacements_NoReplacementWhenTheSquadRanOut(t *testing.T) {
	squad := projectionSquad()[:11]
	lineup := projectionLineup(squad)
	var injured int64
	for _, p := range lineup.Positions {
		if p.Position != GOALKEEPER {
			injured = p.PlayerId
			break
		}
	}

	if got := ProjectKickoffReplacements(lineup, squad, map[int64]bool{injured: true}); len(got) != 0 {
		t.Fatalf("expected no replacement when nobody is left to field, got %v", got)
	}
}

func TestApplyProjectedReplacements_SwapsTheReplacementIntoTheSlotWithoutTouchingTheInput(t *testing.T) {
	lineup := &Lineup{Positions: []LineupPosition{{PlayerId: 1, Position: GOALKEEPER}, {PlayerId: 2, Position: DEFENSE}, {PlayerId: 3, Position: STRIKER}}}

	got := ApplyProjectedReplacements(lineup, []ProjectedReplacement{{PlayerOutId: 2, PlayerInId: 9, Position: DEFENSE}})

	if got.Positions[1].PlayerId != 9 || got.Positions[0].PlayerId != 1 || got.Positions[2].PlayerId != 3 {
		t.Fatalf("got %+v", got.Positions)
	}
	if lineup.Positions[1].PlayerId != 2 {
		t.Fatal("the input lineup must not be modified")
	}
}
