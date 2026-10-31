package common

import (
	"strings"
	"testing"
)

const strongSecret = "0123456789abcdef0123456789abcdef-strong"

func TestDeploymentSecurityProblems(t *testing.T) {
	tests := []struct {
		name                 string
		jwt, session, client string
		wantMentions         []string
	}{
		{"all good", strongSecret, strongSecret + "2", "client.apps.googleusercontent.com", nil},
		{"empty jwt", "", strongSecret, "c", []string{"SECRET_JWT"}},
		{"short session", strongSecret, "short", "c", []string{"SESSION_SECRET"}},
		{"placeholder jwt", "dev-only-change-me", strongSecret, "c", []string{"SECRET_JWT"}},
		{"placeholder is case-insensitive", "DEV-ONLY-CHANGE-ME", strongSecret, "c", []string{"SECRET_JWT"}},
		{"missing client id", strongSecret, strongSecret, "  ", []string{"GOOGLE_OAUTH_CLIENT_ID"}},
		{"everything wrong", "", "my-session-secret", "", []string{"SECRET_JWT", "SESSION_SECRET", "GOOGLE_OAUTH_CLIENT_ID"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := strings.Join(DeploymentSecurityProblems(tt.jwt, tt.session, tt.client), "|")
			for _, want := range tt.wantMentions {
				if !strings.Contains(got, want) {
					t.Errorf("problems %q should mention %s", got, want)
				}
			}
			if len(tt.wantMentions) == 0 && got != "" {
				t.Errorf("expected no problems, got %q", got)
			}
		})
	}
}

func TestCheckDeploymentSecurity(t *testing.T) {
	set := func(jwt, session, client, dev string) {
		t.Setenv("SECRET_JWT", jwt)
		t.Setenv("SESSION_SECRET", session)
		t.Setenv("GOOGLE_OAUTH_CLIENT_ID", client)
		t.Setenv("DEV_LOGIN_ENABLED", dev)
	}

	set(strongSecret, strongSecret, "client", "")
	if err := CheckDeploymentSecurity(); err != nil {
		t.Errorf("a strong configuration must start, got %v", err)
	}

	set("dev-only-change-me", "dev-only-change-me", "", "")
	err := CheckDeploymentSecurity()
	if err == nil {
		t.Fatal("placeholder secrets with dev login off must refuse to start")
	}
	for _, v := range []string{"SECRET_JWT", "SESSION_SECRET", "GOOGLE_OAUTH_CLIENT_ID"} {
		if !strings.Contains(err.Error(), v) {
			t.Errorf("error %q should name %s", err, v)
		}
	}

	set("dev-only-change-me", "dev-only-change-me", "", "true")
	if err := CheckDeploymentSecurity(); err != nil {
		t.Errorf("dev login on tolerates weak configuration, got %v", err)
	}

	set(strongSecret, strongSecret, "client", "true")
	if err := CheckDeploymentSecurity(); err != nil {
		t.Errorf("dev login with a strong configuration must start, got %v", err)
	}
}
