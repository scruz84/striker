package common

import "testing"

// skillsForWage returns a player whose seven core skills give exactly wage as his skill-based
// wage (sum × 5 ÷ 7), so the expectations below read in round numbers.
func playerWithSkillWage(wage int) Player {
	sum := wage * WagePerSkillPointDenominator / WagePerSkillPointNumerator
	p := Player{Age: 25}
	p.Defense = sum / 7
	p.GoalKeeper = sum - 6*p.Defense
	p.Lateral, p.Pass, p.Midfield, p.Dribbling, p.Shot = p.Defense, p.Defense, p.Defense, p.Defense, p.Defense
	return p
}

// team-finances "Contract renewal and release": "No free wage cut", "Last-season veteran" and
// "Improving player"; "Personal terms of a transfer offer": "Pay cut refused" and "Free agent".
func TestPersonalTermsExpectation(t *testing.T) {
	p := playerWithSkillWage(1500)
	if got := PlayerWageFromSkills(p.GoalKeeper, p.Defense, p.Lateral, p.Pass, p.Midfield, p.Dribbling, p.Shot); got != 1500 {
		t.Fatalf("fixture skill wage %d, want 1500", got)
	}
	p.Wage = 2000

	p.ContractYearsRemaining = 3
	if wage, years := PersonalTermsExpectation(p, IntelligenceNaive, 0, TermsRenewal); wage != 2000 || years != 3 {
		t.Errorf("3 seasons left: %d for %d years, want 2000 for 3", wage, years)
	}
	p.ContractYearsRemaining = 1
	if wage, _ := PersonalTermsExpectation(p, IntelligenceAware, 0, TermsRenewal); wage != 1700 {
		t.Errorf("last season, Aware: %d, want 1700 (85%% of 2000)", wage)
	}
	// player-negotiation tiers: "Shrewd last-season veteran" keeps 90%, a Naive one 75%.
	if wage, _ := PersonalTermsExpectation(p, IntelligenceShrewd, 0, TermsRenewal); wage != 1800 {
		t.Errorf("last season, Shrewd: %d, want 1800", wage)
	}
	if wage, _ := PersonalTermsExpectation(p, IntelligenceNaive, 0, TermsRenewal); wage != 1500 {
		t.Errorf("last season, Naive: %d, want his market wage 1500 (75%% of 2000 is lower)", wage)
	}
	if wage, _ := PersonalTermsExpectation(p, IntelligenceNaive, 0, TermsTransfer); wage != 2000 {
		t.Errorf("transfer: %d, want 2000", wage)
	}
	if wage, _ := PersonalTermsExpectation(p, IntelligenceNaive, 0, TermsFreeAgent); wage != 1500 {
		t.Errorf("free agent: %d, want his market wage 1500", wage)
	}

	// An improving Mastermind asks a raise above his current wage: the raise wins over the floor.
	improving := playerWithSkillWage(1000)
	improving.Wage, improving.ContractYearsRemaining = 1000, 3
	raise, _ := PersonalTermsExpectation(improving, IntelligenceMastermind, 50, TermsRenewal)
	if want := ExpectedWageWithIntelligence(improving.GoalKeeper, improving.Defense, improving.Lateral, improving.Pass,
		improving.Midfield, improving.Dribbling, improving.Shot, IntelligenceMastermind, 50); raise != want || raise <= 1000 {
		t.Errorf("improving player: %d, want his raised market wage %d", raise, want)
	}
	if transfer, _ := PersonalTermsExpectation(improving, IntelligenceMastermind, 50, TermsTransfer); transfer != raise {
		t.Errorf("an improving target asks %d from a buyer, want the same %d as from his club", transfer, raise)
	}
}

// player-negotiation "Skill trend in negotiations": "One good match" and "Fewer runs".
func TestAverageSkillTrend(t *testing.T) {
	cases := []struct {
		trends []int
		want   int
	}{{[]int{10, 0, 0, 0, 0}, 2}, {[]int{3, 4}, 4}, {[]int{-3, -4}, -4}, {nil, 0}}
	for _, c := range cases {
		if got := AverageSkillTrend(c.trends); got != c.want {
			t.Errorf("trend of %v: %d, want %d", c.trends, got, c.want)
		}
	}
}
