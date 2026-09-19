package common

import "testing"

func TestIsValidFormation(t *testing.T) {
	valid := []int{FormationBalanced, FormationDefensive, FormationAttacking, FormationFourThreeThree, FormationFourFiveOne, FormationThreeFiveTwo}
	for _, f := range valid {
		if !IsValidFormation(f) {
			t.Fatalf("expected formation %d to be valid", f)
		}
	}
	if IsValidFormation(99) {
		t.Fatal("expected an unknown formation id to be invalid")
	}
}

func TestFormationPositionCounts_EverySupportedFormationHasElevenPlayers(t *testing.T) {
	formations := []int{FormationBalanced, FormationDefensive, FormationAttacking, FormationFourThreeThree, FormationFourFiveOne, FormationThreeFiveTwo}
	for _, f := range formations {
		total := 0
		for _, count := range FormationPositionCounts(f) {
			total += count
		}
		if total != 11 {
			t.Fatalf("formation %d: expected 11 total players, got %d", f, total)
		}
	}
}
