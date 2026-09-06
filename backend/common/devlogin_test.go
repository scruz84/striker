package common

import "testing"

func TestDevLoginEnabled_DefaultsFalse(t *testing.T) {
	t.Setenv("DEV_LOGIN_ENABLED", "")
	if DevLoginEnabled() {
		t.Fatal("expected dev login to default to disabled")
	}
}

func TestDevLoginEnabled_TrueWhenSet(t *testing.T) {
	t.Setenv("DEV_LOGIN_ENABLED", "true")
	if !DevLoginEnabled() {
		t.Fatal("expected dev login to be enabled")
	}
}

func TestFindDevUser_CaseInsensitiveMatch(t *testing.T) {
	u := FindDevUser("ADMIN@dev.local")
	if u == nil {
		t.Fatal("expected to find the seeded admin dev user")
	}
	if !u.IsAdmin {
		t.Fatal("expected admin@dev.local to be an admin")
	}
}

func TestFindDevUser_UnknownEmail(t *testing.T) {
	if FindDevUser("nobody@example.com") != nil {
		t.Fatal("expected no match for an email outside the roster")
	}
}
