package engine

import "testing"

func TestAsTeamIdPtr(t *testing.T) {
	if got := asTeamIdPtr(ByeTeamId); got != nil {
		t.Fatalf("expected nil for a bye slot, got %v", *got)
	}
	if got := asTeamIdPtr(42); got == nil || *got != 42 {
		t.Fatalf("expected a pointer to 42, got %v", got)
	}
}
