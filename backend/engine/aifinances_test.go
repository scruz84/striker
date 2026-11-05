package engine

import (
	"testing"
	"time"
)

func TestInFinancialCrisis(t *testing.T) {
	if !inFinancialCrisis(-1) || !inFinancialCrisis(-50000) {
		t.Error("a budget below 0 must be a crisis")
	}
	if inFinancialCrisis(0) || inFinancialCrisis(10000) {
		t.Error("a budget of 0 or more is no crisis")
	}
}

// cpu-team-management "CPU financial crisis": the best-paid sellable player, never a key player, a
// protected one or one whose sale thins his position.
func TestPickCrisisSale(t *testing.T) {
	squad := squadOf(2, 5, 6, 5) // ids 1-2 GK, 3-7 D, 8-13 M, 14-18 S
	for _, p := range squad {
		p.Wage = 1000
	}
	squad[0].Wage = 9000 // first keeper: only 2 keepers, his sale would thin the position
	squad[8].Wage = 5000 // a bench midfielder (id 9), well-paid but not key
	squad[8].Midfield, squad[8].Pass = 40, 40
	squad[9].Wage = 7000  // another midfielder (id 10), protected below
	squad[2].Wage = 6000  // the best defender (id 3) - make him key
	squad[2].Defense = 99 // far above the 85s and below behind him
	protected := map[int64]bool{10: true}

	got := pickCrisisSale(squad, none(), none(), protected)
	if got == nil || got.Id != 9 {
		t.Fatalf("crisis sale %v, want the best-paid sellable player 9", got)
	}
	if got := pickCrisisSale(squad, none(), map[int64]bool{9: true}, protected); got == nil || got.Id == 9 {
		t.Errorf("a loan-listed player was picked: %v", got)
	}
	if got := pickCrisisSale(squad[:16], none(), none(), protected); got != nil {
		t.Errorf("a squad of 16 cannot spare anyone: %v", got)
	}
}

// cpu-team-management "Releasing unsold veterans": "Nobody wanted him", "Squad at its size" and
// "Listed for 10 days".
func TestPickVeteranToRelease(t *testing.T) {
	now := time.Now()
	squad := squadOf(2, 6, 7, 5) // 20 owned players
	for _, p := range squad {
		p.Age, p.ContractYearsRemaining, p.Wage = 25, 3, 1000
	}
	veteran := squad[10]
	veteran.Age, veteran.ContractYearsRemaining, veteran.Wage = 32, 2, 3000
	listed := map[int64]time.Time{veteran.Id: now.Add(-15 * 24 * time.Hour)}

	if got := pickVeteranToRelease(squad, listed, now, 200000); got == nil || got.Id != veteran.Id {
		t.Errorf("released %v, want the unsold veteran %d", got, veteran.Id)
	}
	if got := pickVeteranToRelease(squad[:18], listed, now, 200000); got != nil {
		t.Errorf("a squad of 18 released %v", got)
	}
	if got := pickVeteranToRelease(squad, map[int64]time.Time{veteran.Id: now.Add(-10 * 24 * time.Hour)}, now, 200000); got != nil {
		t.Errorf("a veteran listed for 10 days was released")
	}
	if got := pickVeteranToRelease(squad, listed, now, 89999); got != nil {
		t.Errorf("released with a budget under 3 x the 30,000 severance")
	}
	veteran.ContractYearsRemaining = 1
	if got := pickVeteranToRelease(squad, listed, now, 200000); got != nil {
		t.Errorf("released a veteran whose contract runs out this season")
	}
}
