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

// A starter who has left the club since the lineup was saved (issue #399) is replaced at kickoff
// like an injured one - he is not in the squad, so he would otherwise play with no skills.
func TestExcludeUnavailableAndBackfill_ReplacesAStarterWhoLeftTheClub(t *testing.T) {
	squad := projectionSquad()
	lineup := projectionLineup(squad)
	var sold LineupPosition
	for _, p := range lineup.Positions {
		if p.Position != GOALKEEPER {
			sold = p
			break
		}
	}
	remaining := make([]*Player, 0, len(squad)-1)
	for _, p := range squad {
		if p.Id != sold.PlayerId {
			remaining = append(remaining, p)
		}
	}
	byPosition := make(map[int][]int64)
	for _, p := range lineup.Positions {
		byPosition[p.Position] = append(byPosition[p.Position], p.PlayerId)
	}

	ExcludeUnavailableAndBackfill(&byPosition, remaining, nil)

	count := 0
	for _, ids := range byPosition {
		for _, id := range ids {
			count++
			if id == sold.PlayerId {
				t.Fatalf("the sold player %d still starts: %v", sold.PlayerId, byPosition)
			}
		}
	}
	if count != len(lineup.Positions) {
		t.Fatalf("expected a full XI of %d, got %d: %v", len(lineup.Positions), count, byPosition)
	}
	if ids := byPosition[sold.Position]; ids[len(ids)-1] != 13 {
		t.Fatalf("expected the best outfield reserve (13) in the sold player's slot, got %v", ids)
	}

	got := ProjectKickoffReplacements(lineup, remaining, nil)
	if len(got) != 1 || got[0].PlayerOutId != sold.PlayerId || got[0].PlayerInId != 13 {
		t.Fatalf("expected the projection to replace the sold player with 13, got %v", got)
	}
}

// A goalkeeper is exempt from the injury/suspension exclusion, but not from leaving the club.
func TestExcludeUnavailableAndBackfill_ReplacesAKeeperWhoLeftTheClub(t *testing.T) {
	squad := projectionSquad()
	lineup := projectionLineup(squad)
	byPosition := make(map[int][]int64)
	for _, p := range lineup.Positions {
		byPosition[p.Position] = append(byPosition[p.Position], p.PlayerId)
	}
	if keepers := byPosition[GOALKEEPER]; len(keepers) != 1 || keepers[0] != 1 {
		t.Fatalf("expected keeper 1 in the lineup, got %v", keepers)
	}
	remaining := squad[1:]

	ExcludeUnavailableAndBackfill(&byPosition, remaining, nil)

	if keepers := byPosition[GOALKEEPER]; len(keepers) != 1 || keepers[0] != 12 {
		t.Fatalf("expected the reserve keeper 12 in goal, got %v", keepers)
	}
	got := ProjectKickoffReplacements(lineup, remaining, nil)
	if len(got) != 1 || got[0].PlayerOutId != 1 || got[0].PlayerInId != 12 || got[0].Position != GOALKEEPER {
		t.Fatalf("expected the projection to put keeper 12 in for 1, got %v", got)
	}
}

// A squad that could not be resolved (empty) never strips the lineup.
func TestExcludeUnavailableAndBackfill_EmptySquadDepartsNobody(t *testing.T) {
	byPosition := map[int][]int64{GOALKEEPER: {1}, DEFENSE: {2, 3}}
	ExcludeUnavailableAndBackfill(&byPosition, nil, nil)
	if len(byPosition[GOALKEEPER]) != 1 || len(byPosition[DEFENSE]) != 2 {
		t.Fatalf("expected the lineup unchanged, got %v", byPosition)
	}
}

// A saved lineup thinned by players who left the club (their rows are deleted, see
// removePlayerFromTeamLineupsTx) is topped back up to its formation's full XI at kickoff, instead
// of kicking off short and forfeiting with a full squad on the books.
func TestExcludeUnavailableAndBackfillToFormation_TopsUpAThinnedLineup(t *testing.T) {
	squad := projectionSquad()
	lineup := projectionLineup(squad)
	byPosition := make(map[int][]int64)
	removed := 0
	for _, p := range lineup.Positions {
		if p.Position != GOALKEEPER && removed < 5 {
			removed++
			continue
		}
		byPosition[p.Position] = append(byPosition[p.Position], p.PlayerId)
	}

	ExcludeUnavailableAndBackfillToFormation(&byPosition, squad, nil, lineup.Formation)

	want := FormationPositionCounts(lineup.Formation)
	seen := make(map[int64]bool)
	for position, count := range want {
		if got := len(byPosition[position]); got != count {
			t.Fatalf("position %d: expected %d players, got %d (%v)", position, count, got, byPosition)
		}
		for _, id := range byPosition[position] {
			if seen[id] {
				t.Fatalf("player %d fielded twice: %v", id, byPosition)
			}
			seen[id] = true
		}
	}
}

// An empty saved lineup gets a full XI for its formation.
func TestExcludeUnavailableAndBackfillToFormation_FillsAnEmptyLineup(t *testing.T) {
	byPosition := make(map[int][]int64)

	ExcludeUnavailableAndBackfillToFormation(&byPosition, projectionSquad(), nil, FormationBalanced)

	total := 0
	for _, ids := range byPosition {
		total += len(ids)
	}
	if total != 11 || len(byPosition[GOALKEEPER]) != 1 || byPosition[GOALKEEPER][0] != 1 {
		t.Fatalf("expected a full XI with keeper 1 in goal, got %v", byPosition)
	}
}

// A full lineup is never grown past the formation's size, even when its positions do not match
// the formation's counts.
func TestExcludeUnavailableAndBackfillToFormation_NeverGrowsAFullLineup(t *testing.T) {
	byPosition := map[int][]int64{GOALKEEPER: {1}, DEFENSE: {2, 3, 4, 5, 6}, MIDFIELD: {7, 8, 9}, STRIKER: {10, 11}}

	ExcludeUnavailableAndBackfillToFormation(&byPosition, projectionSquad(), nil, FormationBalanced)

	total := 0
	for _, ids := range byPosition {
		total += len(ids)
	}
	if total != 11 {
		t.Fatalf("expected the XI to stay at 11, got %d: %v", total, byPosition)
	}
}

// Without a valid formation nothing is topped up (the editor's projection relies on this).
func TestExcludeUnavailableAndBackfillToFormation_InvalidFormationKeepsTheOldBehaviour(t *testing.T) {
	byPosition := map[int][]int64{GOALKEEPER: {1}, DEFENSE: {2}}

	ExcludeUnavailableAndBackfillToFormation(&byPosition, projectionSquad(), nil, -1)

	if len(byPosition[GOALKEEPER]) != 1 || len(byPosition[DEFENSE]) != 1 || len(byPosition[MIDFIELD]) != 0 {
		t.Fatalf("expected the lineup unchanged, got %v", byPosition)
	}
}
