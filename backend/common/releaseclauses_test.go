package common

import "testing"

// release-clauses "Wage discount for a clause": "Low clause" and "High clause".
func TestReleaseClauseDiscount(t *testing.T) {
	amount := func(v int64) *int64 { return &v }
	cases := []struct {
		clause *int64
		want   float64
	}{
		{nil, 0},
		{amount(800_000), 0.10},
		{amount(1_600_000), 0.05},
		{amount(2_000_000), 0},
	}
	for _, c := range cases {
		if got := ReleaseClauseDiscount(c.clause, 400_000); got != c.want {
			t.Errorf("clause %v against 400,000: discount %v, want %v", c.clause, got, c.want)
		}
	}
}

// release-clauses "Setting a clause": "Clause too low"; "Wage discount for a clause": the discounted
// wage rounds up.
func TestValidateReleaseClauseAndDiscountedWage(t *testing.T) {
	p := Player{Age: 25, GoalKeeper: 70, Defense: 70, Lateral: 70, Pass: 70, Midfield: 70, Dribbling: 70, Shot: 70, ContractYearsRemaining: 1}
	base := ClauseBaseValue(p)
	fresh := p
	fresh.ContractYearsRemaining = 3
	if base != MarketValue(fresh) {
		t.Fatalf("base value %d, want the 3-season market value %d", base, MarketValue(fresh))
	}
	low := MinReleaseClause(p) - 1
	if err := ValidateReleaseClause(p, &low); err != ErrReleaseClauseTooLow {
		t.Errorf("clause below 1.5x: %v, want ErrReleaseClauseTooLow", err)
	}
	if err := ValidateReleaseClause(p, nil); err != nil {
		t.Errorf("no clause: %v", err)
	}
	twice := 2 * base
	if got := DiscountedWage(p, 2001, &twice); got != 1801 {
		t.Errorf("wage with a 2x clause: %d, want 1,801 (90%% of 2,001, rounded up)", got)
	}
	if got := DiscountedWage(p, 2000, nil); got != 2000 {
		t.Errorf("wage without a clause: %d, want 2,000", got)
	}
}
