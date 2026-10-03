package common

import (
	"sort"
	"testing"
)

func makePlayer(id int64, skill int) *Player {
	// A flat skill for every stat keeps the derived position values (GetLateralDefenseValue
	// etc.) proportional to it too, so "higher skill" unambiguously means "should be picked
	// first" regardless of which slot is being filled.
	return &Player{
		Id: id, GoalKeeper: skill, Defense: skill, Lateral: skill, Pass: skill,
		Midfield: skill, Dribbling: skill, Shot: skill,
	}
}

func TestSelectBestLineup_FillsEverySlotWhenEnoughPlayers(t *testing.T) {
	players := make([]*Player, 0, 18)
	for i := int64(1); i <= 18; i++ {
		players = append(players, makePlayer(i, int(i)))
	}

	lineup := SelectBestLineup(players)

	total := 0
	seen := make(map[int64]bool)
	for _, ids := range lineup {
		for _, id := range ids {
			if seen[id] {
				t.Fatalf("player %d selected more than once", id)
			}
			seen[id] = true
			total++
		}
	}
	if want := len(formationSlotOrder(FormationBalanced)); total != want {
		t.Fatalf("expected %d players selected, got %d", want, total)
	}
}

func TestSelectBestLineup_PrefersHigherSkill(t *testing.T) {
	// Only two players and one goalkeeper slot in the pool - the better one must win it.
	best := makePlayer(1, 90)
	worst := makePlayer(2, 10)

	lineup := SelectBestLineup([]*Player{worst, best})

	if len(lineup[GOALKEEPER]) != 1 || lineup[GOALKEEPER][0] != best.Id {
		t.Fatalf("expected player %d as goalkeeper, got %v", best.Id, lineup[GOALKEEPER])
	}
}

func TestSelectBestLineup_FewerPlayersThanSlotsDoesNotPanic(t *testing.T) {
	players := []*Player{makePlayer(1, 50), makePlayer(2, 40), makePlayer(3, 30)}

	lineup := SelectBestLineup(players)

	total := 0
	for _, ids := range lineup {
		total += len(ids)
	}
	if total != len(players) {
		t.Fatalf("expected all %d players placed, got %d", len(players), total)
	}
}

func TestSelectBestLineup_EmptyPlayersDoesNotPanic(t *testing.T) {
	lineup := SelectBestLineup(nil)
	if len(lineup) != 0 {
		t.Fatalf("expected an empty lineup, got %v", lineup)
	}
}

func TestFlattenLineup(t *testing.T) {
	lineup := map[int][]int64{
		GOALKEEPER: {1},
		DEFENSE:    {2, 3},
	}

	positions := FlattenLineup(lineup)

	if len(positions) != 3 {
		t.Fatalf("expected 3 flattened positions, got %d", len(positions))
	}
	byPlayer := make(map[int64]int)
	for _, p := range positions {
		byPlayer[p.PlayerId] = p.Position
	}
	if byPlayer[1] != GOALKEEPER || byPlayer[2] != DEFENSE || byPlayer[3] != DEFENSE {
		t.Fatalf("unexpected flattened positions: %v", positions)
	}
}

func TestPositionValue_CoversEveryPosition(t *testing.T) {
	p := Player{GoalKeeper: 10, Defense: 20, Lateral: 30, Pass: 40, Midfield: 50, Dribbling: 60, Shot: 70}

	allPositions := []int{
		GOALKEEPER, DEFENSE, LATERAL_DEFENSE_LEFT, LATERAL_DEFENSE_RIGHT,
		MIDFIELD, LATERAL_MIDFIELD_LEFT, LATERAL_MIDFIELD_RIGHT,
		STRIKER, LATERAL_STRIKER_LEFT, LATERAL_STRIKER_RIGHT,
	}
	for _, position := range allPositions {
		if PositionValue(position, p) <= 0 {
			t.Errorf("expected a positive value for position %d, got %d", position, PositionValue(position, p))
		}
	}
}

func TestPositionValue_HigherSkillScoresHigher(t *testing.T) {
	weak := Player{GoalKeeper: 10, Defense: 10, Lateral: 10, Pass: 10, Midfield: 10, Dribbling: 10, Shot: 10}
	strong := Player{GoalKeeper: 90, Defense: 90, Lateral: 90, Pass: 90, Midfield: 90, Dribbling: 90, Shot: 90}

	if PositionValue(STRIKER, strong) <= PositionValue(STRIKER, weak) {
		t.Fatal("expected the stronger player to score higher at striker")
	}
}

func TestFormationSlotOrder_MatchesFormationPositionCountsForEveryFormation(t *testing.T) {
	for _, formation := range AllFormations {
		order := formationSlotOrder(formation)
		if len(order) != 11 {
			t.Errorf("formation %d: expected 11 total slots, got %d", formation, len(order))
		}
		got := make(map[int]int)
		for _, position := range order {
			got[position]++
		}
		for position, want := range FormationPositionCounts(formation) {
			if got[position] != want {
				t.Errorf("formation %d: position %d wanted %d slots, formationSlotOrder gave %d", formation, position, want, got[position])
			}
		}
	}
}

func TestSelectBestLineupForFormation_FillsFormationShapeNotJustBalanced(t *testing.T) {
	players := make([]*Player, 0, 18)
	for i := int64(1); i <= 18; i++ {
		players = append(players, makePlayer(i, int(i)))
	}

	lineup := SelectBestLineupForFormation(players, FormationThreeFiveTwo)

	for position, want := range FormationPositionCounts(FormationThreeFiveTwo) {
		if got := len(lineup[position]); got != want {
			t.Errorf("position %d: wanted %d players for a 3-5-2, got %d", position, want, got)
		}
	}
	// FormationThreeFiveTwo has no LATERAL_STRIKER slots at all (see FormationPositionCounts) -
	// unlike FormationBalanced this isn't just "zero because it wasn't requested", it genuinely
	// shouldn't appear.
	if len(lineup[LATERAL_STRIKER_LEFT]) != 0 || len(lineup[LATERAL_STRIKER_RIGHT]) != 0 {
		t.Fatalf("expected no lateral strikers in a 3-5-2, got left=%v right=%v", lineup[LATERAL_STRIKER_LEFT], lineup[LATERAL_STRIKER_RIGHT])
	}
}

// diverseSquadForFormationTest builds a deterministic but non-uniform squad (each of the seven
// skills varies independently per player via a fixed pseudo-random formula, not a flat value like
// makePlayer) - needed so a squad's best-XI strength actually differs across formations (with
// every skill flat, every formation's PositionValue formulas collapse to the same number per
// player, see makePlayer's own doc comment, making every formation tie).
func diverseSquadForFormationTest() []*Player {
	players := make([]*Player, 0, 20)
	skillFor := func(id int64, multiplier int64) int {
		return int((id*multiplier+id*id)%90) + 10
	}
	for i := int64(1); i <= 20; i++ {
		players = append(players, &Player{
			Id:         i,
			GoalKeeper: skillFor(i, 3),
			Defense:    skillFor(i, 5),
			Lateral:    skillFor(i, 7),
			Pass:       skillFor(i, 11),
			Midfield:   skillFor(i, 13),
			Dribbling:  skillFor(i, 17),
			Shot:       skillFor(i, 19),
		})
	}
	return players
}

// rankedFormationStrengths independently recomputes every formation's own best-XI strength (the
// same building blocks SelectBestFormationAndLineup itself uses) and returns them sorted
// strongest-first - the ground truth formationRankForAiLevel's rank picks are checked against.
func rankedFormationStrengths(players []*Player) []int {
	playersById := make(map[int64]*Player, len(players))
	for _, p := range players {
		playersById[p.Id] = p
	}
	strengths := make([]int, 0, len(AllFormations))
	for _, formation := range AllFormations {
		strengths = append(strengths, lineupStrength(SelectBestLineupForFormation(players, formation), playersById))
	}
	sort.Sort(sort.Reverse(sort.IntSlice(strengths)))
	return strengths
}

func TestSelectBestFormationAndLineup_DirectorFindsTheGenuinelyStrongestFormation(t *testing.T) {
	players := diverseSquadForFormationTest()
	playersById := make(map[int64]*Player, len(players))
	for _, p := range players {
		playersById[p.Id] = p
	}
	ranked := rankedFormationStrengths(players)

	_, lineup := SelectBestFormationAndLineup(players, AiLevelDirector)
	if got := lineupStrength(lineup, playersById); got != ranked[0] {
		t.Fatalf("expected Director's lineup strength to equal the best possible (%d), got %d", ranked[0], got)
	}
}

func TestSelectBestFormationAndLineup_ManagerSettlesForTheSecondBest(t *testing.T) {
	players := diverseSquadForFormationTest()
	playersById := make(map[int64]*Player, len(players))
	for _, p := range players {
		playersById[p.Id] = p
	}
	ranked := rankedFormationStrengths(players)

	_, lineup := SelectBestFormationAndLineup(players, AiLevelManager)
	if got := lineupStrength(lineup, playersById); got != ranked[1] {
		t.Fatalf("expected Manager's lineup strength to equal the second best (%d), got %d", ranked[1], got)
	}
}

func TestSelectBestFormationAndLineup_CaretakerSettlesForAWeakerFormationThanManager(t *testing.T) {
	players := diverseSquadForFormationTest()
	playersById := make(map[int64]*Player, len(players))
	for _, p := range players {
		playersById[p.Id] = p
	}
	ranked := rankedFormationStrengths(players)

	for i := 0; i < 50; i++ {
		_, lineup := SelectBestFormationAndLineup(players, AiLevelCaretaker)
		if got := lineupStrength(lineup, playersById); got != ranked[2] && got != ranked[3] {
			t.Fatalf("expected Caretaker's lineup strength to be the 3rd or 4th best (%d or %d), got %d", ranked[2], ranked[3], got)
		}
	}
}

func TestFormationRankForAiLevel_DirectorAlwaysRankZero(t *testing.T) {
	for i := 0; i < 100; i++ {
		if r := formationRankForAiLevel(AiLevelDirector, 6); r != 0 {
			t.Fatalf("expected Director to always pick rank 0, got %d", r)
		}
	}
}

func TestFormationRankForAiLevel_ManagerAlwaysRankOne(t *testing.T) {
	for i := 0; i < 100; i++ {
		if r := formationRankForAiLevel(AiLevelManager, 6); r != 1 {
			t.Fatalf("expected Manager to always pick rank 1, got %d", r)
		}
	}
}

func TestFormationRankForAiLevel_CaretakerVariesBetweenThirdAndFourthBest(t *testing.T) {
	seen := map[int]bool{}
	for i := 0; i < 200; i++ {
		r := formationRankForAiLevel(AiLevelCaretaker, 6)
		if r != 2 && r != 3 {
			t.Fatalf("expected Caretaker's rank to be 2 or 3, got %d", r)
		}
		seen[r] = true
	}
	if len(seen) != 2 {
		t.Fatalf("expected Caretaker to use both ranks 2 and 3 across many trials, only saw %v", seen)
	}
}

func TestFormationRankForAiLevel_ClampsToShorterCandidateLists(t *testing.T) {
	for i := 0; i < 50; i++ {
		if r := formationRankForAiLevel(AiLevelCaretaker, 2); r != 1 {
			t.Fatalf("expected rank to clamp to the last valid index (1) for only 2 candidates, got %d", r)
		}
	}
}

func TestFormationRankForAiLevel_UnassignedLevelBehavesLikeCaretaker(t *testing.T) {
	if r := formationRankForAiLevel(0, 6); r != 2 && r != 3 {
		t.Fatalf("expected an unassigned AI level (0) to behave like Caretaker, got rank %d", r)
	}
}
