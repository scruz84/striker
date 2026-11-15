package common

import "testing"

func contains(list []string, want string) bool {
	for _, item := range list {
		if item == want {
			return true
		}
	}
	return false
}

func TestCorsAllowedOrigins_IncludesLocalDevDefaults(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "")

	origins := CorsAllowedOrigins()

	for _, want := range []string{"http://localhost:5173", "http://localhost:3000"} {
		if !contains(origins, want) {
			t.Fatalf("expected %q among default origins, got %v", want, origins)
		}
	}
}

func TestCorsAllowedOrigins_DefaultsAreLocalOnly(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "")

	origins := CorsAllowedOrigins()

	if len(origins) != 2 {
		t.Fatalf("expected only the two local dev origins by default, got %v", origins)
	}
}

func TestCorsAllowedOrigins_DeploymentOriginComesFromEnv(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://striker.example.com")

	if !contains(CorsAllowedOrigins(), "https://striker.example.com") {
		t.Fatalf("expected the configured deployment origin to be allowed")
	}
}

func TestCorsAllowedOrigins_AppendsConfiguredExtras(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://localhost:4000, https://staging.example.com ,")

	origins := CorsAllowedOrigins()

	for _, want := range []string{"http://localhost:4000", "https://staging.example.com"} {
		if !contains(origins, want) {
			t.Fatalf("expected %q among origins, got %v", want, origins)
		}
	}
	// Defaults must still be present alongside the configured extras.
	if !contains(origins, "http://localhost:5173") {
		t.Fatalf("expected defaults to still be present, got %v", origins)
	}
}
