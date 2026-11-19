package common

import (
	"os"
	"strings"
)

// defaultCorsAllowedOrigins are always allowed, regardless of CORS_ALLOWED_ORIGINS - the two
// ports the frontend actually runs on locally: 5173 for `npm run dev` (Vite's default) and 3000
// for the Dockerized/nginx-served build (the frontend service of the root docker-compose.yml). A
// deployment adds its own public frontend origin(s) through CORS_ALLOWED_ORIGINS.
var defaultCorsAllowedOrigins = []string{
	"http://localhost:3000",
	"http://localhost:5173",
}

// CorsAllowedOrigins returns the origins the API accepts cross-origin requests from -
// defaultCorsAllowedOrigins plus any extra origins listed in the comma-separated
// CORS_ALLOWED_ORIGINS environment variable (a custom local port, a staging domain, ...).
func CorsAllowedOrigins() []string {
	origins := append([]string{}, defaultCorsAllowedOrigins...)
	configured := os.Getenv("CORS_ALLOWED_ORIGINS")
	if configured == "" {
		return origins
	}
	for _, candidate := range strings.Split(configured, ",") {
		if candidate = strings.TrimSpace(candidate); candidate != "" {
			origins = append(origins, candidate)
		}
	}
	return origins
}
