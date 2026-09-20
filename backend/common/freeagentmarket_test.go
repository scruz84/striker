package common

import "testing"

func TestSetFreeAgentInjectionsPerWeek_RejectsNegativeWithoutTouchingDatabase(t *testing.T) {
	// The count is checked before any database access, so this is safe to exercise with a nil
	// DatabaseContext - same trick as TestSetTicketPrice_RejectsOutOfRange.
	if err := SetFreeAgentInjectionsPerWeek(nil, -1); err != ErrInvalidFreeAgentInjectionCount {
		t.Fatalf("expected ErrInvalidFreeAgentInjectionCount for a negative count, got %v", err)
	}
}

func TestInjectFreeAgents_ZeroCountIsNoop(t *testing.T) {
	// count <= 0 returns before touching the database, so this is safe with a nil DatabaseContext.
	if err := InjectFreeAgents(nil, 0); err != nil {
		t.Fatalf("expected no error for a zero count, got %v", err)
	}
	if err := InjectFreeAgents(nil, -5); err != nil {
		t.Fatalf("expected no error for a negative count, got %v", err)
	}
}
