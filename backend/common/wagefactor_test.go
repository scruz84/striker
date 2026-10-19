package common

import "testing"

func TestDivisionWageFactor(t *testing.T) {
	tests := []struct {
		tier int
		want float64
	}{{0, 1.0}, {-1, 1.0}, {1, 4.0}, {2, 2.1}, {3, 1.1}, {4, 1.1}, {9, 1.1}}
	for _, tt := range tests {
		if got := DivisionWageFactor(tt.tier); got != tt.want {
			t.Errorf("DivisionWageFactor(%d) = %v, want %v", tt.tier, got, tt.want)
		}
	}
	if !(DivisionWageFactor(1) > DivisionWageFactor(2) && DivisionWageFactor(2) > DivisionWageFactor(3)) {
		t.Error("a higher division must pay a higher wage factor")
	}
}

func TestApplyWageFactor(t *testing.T) {
	tests := []struct {
		base   int64
		factor float64
		want   int64
	}{{0, 4.0, 0}, {1000, 1.0, 1000}, {1000, 4.0, 4000}, {1000, 2.1, 2100}, {333, 1.1, 366}, {7, 2.1, 15}}
	for _, tt := range tests {
		if got := ApplyWageFactor(tt.base, tt.factor); got != tt.want {
			t.Errorf("ApplyWageFactor(%d, %v) = %d, want %d", tt.base, tt.factor, got, tt.want)
		}
	}
}

// The division factor must reach the CPU's affordability check: the same signing that a third-
// division payroll can carry is out of reach once the first-division factor multiplies both the
// existing bill and the candidate's own wage.
func TestScoreTarget_WageFactorMakesTheSamePayrollLessAffordable(t *testing.T) {
	squad := settledSquad()
	target := midfielderAged(70, 95, 27)
	profile := AiTransferProfileFor(AiLevelDirector)

	ctx := scoreContext(squad, target)
	ctx.Price = 0
	ctx.Target.Wage = 1_000
	ctx.WageBill = 100_000
	// Budget exactly covers WageRunwayRounds of the unfactored new bill, and not of the x4 one.
	ctx.Budget = (ctx.WageBill + int64(ctx.Target.Wage)) * WageRunwayRounds * 2

	ctx.WageFactor = 1.0
	if s := ScoreTarget(profile, ctx, newRng(1)); !s.Affordable {
		t.Fatalf("expected the signing to be affordable without a factor: %+v", s)
	}

	// Same stored figures, but the bill and the new wage are paid x4 in the top flight.
	ctx.WageBill = ApplyWageFactor(ctx.WageBill, 4.0)
	ctx.WageFactor = 4.0
	if s := ScoreTarget(profile, ctx, newRng(1)); s.Affordable {
		t.Fatalf("expected the same signing to be unaffordable at the first-division factor: %+v", s)
	}
}
