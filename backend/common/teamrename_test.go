package common

import "testing"

func TestValidateTeamName_TrimsWhitespace(t *testing.T) {
	got, err := ValidateTeamName("  Ashford United  ")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if got != "Ashford United" {
		t.Fatalf("expected trimmed name, got %q", got)
	}
}

func TestValidateTeamName_RejectsTooShort(t *testing.T) {
	if _, err := ValidateTeamName("A"); err != ErrInvalidTeamName {
		t.Fatalf("expected ErrInvalidTeamName, got %v", err)
	}
}

func TestValidateTeamName_RejectsBlank(t *testing.T) {
	if _, err := ValidateTeamName("   "); err != ErrInvalidTeamName {
		t.Fatalf("expected ErrInvalidTeamName, got %v", err)
	}
}

func TestValidateTeamName_RejectsTooLong(t *testing.T) {
	longName := ""
	for i := 0; i < MaxTeamNameLength+1; i++ {
		longName += "a"
	}
	if _, err := ValidateTeamName(longName); err != ErrInvalidTeamName {
		t.Fatalf("expected ErrInvalidTeamName, got %v", err)
	}
}

func TestValidateTeamName_AcceptsBoundaryLengths(t *testing.T) {
	if _, err := ValidateTeamName("AB"); err != nil {
		t.Fatalf("expected min-length name to be valid, got %s", err)
	}
	maxName := ""
	for i := 0; i < MaxTeamNameLength; i++ {
		maxName += "a"
	}
	if _, err := ValidateTeamName(maxName); err != nil {
		t.Fatalf("expected max-length name to be valid, got %s", err)
	}
}
