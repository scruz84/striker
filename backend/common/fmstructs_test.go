package common

import "testing"

func TestRedactFinancials_ZeroesRosteredPlayer(t *testing.T) {
	p := Player{TeamId: 42, Wage: 500, ContractYearsRemaining: 2}
	p.RedactFinancials()
	if p.Wage != 0 || p.ContractYearsRemaining != 0 {
		t.Fatalf("expected wage and contract to be zeroed, got wage=%d contractYearsRemaining=%d", p.Wage, p.ContractYearsRemaining)
	}
}

func TestRedactFinancials_LeavesFreeAgentUntouched(t *testing.T) {
	p := Player{TeamId: 0, Wage: 500, ContractYearsRemaining: 0}
	p.RedactFinancials()
	if p.Wage != 500 {
		t.Fatalf("expected a free agent's wage to stay visible, got %d", p.Wage)
	}
}
