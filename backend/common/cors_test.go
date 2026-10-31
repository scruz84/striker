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

func TestCorsAllowedOrigins_IncludesDeployedDomains(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "")

	origins := CorsAllowedOrigins()

	for _, want := range []string{"https://striker.example.com", "https://striker.example.com"} {
		if !contains(origins, want) {
			t.Fatalf("expected %q among default origins, got %v", want, origins)
		}
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
