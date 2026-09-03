package common

import "testing"

func TestIsConfiguredAdminEmail(t *testing.T) {
	t.Setenv("ADMIN_EMAILS", "Alice@Example.com, bob@example.com")

	cases := []struct {
		email string
		want  bool
	}{
		{"alice@example.com", true},   // case-insensitive match
		{"bob@example.com", true},     // exact match, second in the list
		{"  bob@example.com  ", true}, // both sides are trimmed before comparing
		{"carol@example.com", false},  // not listed
		{"", false},
	}
	for _, tc := range cases {
		if got := IsConfiguredAdminEmail(tc.email); got != tc.want {
			t.Errorf("IsConfiguredAdminEmail(%q) = %v, want %v", tc.email, got, tc.want)
		}
	}
}

func TestIsConfiguredAdminEmail_UnsetEnv(t *testing.T) {
	t.Setenv("ADMIN_EMAILS", "")

	if IsConfiguredAdminEmail("anyone@example.com") {
		t.Fatal("expected no admin emails to be configured")
	}
}
