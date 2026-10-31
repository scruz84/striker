package common

import (
	"fmt"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
)

// MinSecretLength is the shortest SECRET_JWT / SESSION_SECRET a non-dev deployment may start
// with: HS256 signing keys shorter than the hash output are brute-forceable, and 32 bytes is what
// `openssl rand -base64 48` comfortably exceeds.
const MinSecretLength = 32

// placeholderSecrets are the values shipped in compose files, app.env.sample and the docs. A
// deployment that forgets to override one would accept session cookies anyone can sign, admins
// included.
var placeholderSecrets = map[string]bool{
	"dev-only-change-me": true,
	"my-secret":          true,
	"my-session-secret":  true,
}

// GoogleOAuthClientId is the Google OAuth client id this deployment accepts ID tokens for (the
// token's `aud` claim). Same value the frontend is built with (GOOGLE_OAUTH_CLIENT_ID there).
func GoogleOAuthClientId() string {
	return strings.TrimSpace(os.Getenv("GOOGLE_OAUTH_CLIENT_ID"))
}

// weakSecret reports whether v is unfit to sign sessions with: empty, shorter than
// MinSecretLength or a known placeholder.
func weakSecret(v string) bool {
	v = strings.TrimSpace(v)
	return len(v) < MinSecretLength || placeholderSecrets[strings.ToLower(v)]
}

// DeploymentSecurityProblems lists everything wrong with the security-relevant configuration:
// weak or placeholder SECRET_JWT / SESSION_SECRET and a missing GOOGLE_OAUTH_CLIENT_ID. Each entry
// names the variable so the operator knows what to set. Takes the values as arguments so it can
// be tested without touching the environment.
func DeploymentSecurityProblems(secretJwt, sessionSecret, googleClientId string) []string {
	var problems []string
	if weakSecret(secretJwt) {
		problems = append(problems, fmt.Sprintf("SECRET_JWT is empty, shorter than %d characters or a placeholder", MinSecretLength))
	}
	if weakSecret(sessionSecret) {
		problems = append(problems, fmt.Sprintf("SESSION_SECRET is empty, shorter than %d characters or a placeholder", MinSecretLength))
	}
	if strings.TrimSpace(googleClientId) == "" {
		problems = append(problems, "GOOGLE_OAUTH_CLIENT_ID is not set, so Google sign-in cannot validate ID tokens")
	}
	return problems
}

// CheckDeploymentSecurity applies DeploymentSecurityProblems to the environment. With dev login
// off any problem is returned as an error (the caller refuses to start); with it on each problem
// is logged at ERROR and the process may continue, together with a banner that dev login is on.
func CheckDeploymentSecurity() error {
	problems := DeploymentSecurityProblems(os.Getenv("SECRET_JWT"), os.Getenv("SESSION_SECRET"), GoogleOAuthClientId())
	if DevLoginEnabled() {
		log.Error("DEV LOGIN ENABLED - anyone who can reach this API can sign in as any dev user, admin included. Never on a public deployment.")
		for _, p := range problems {
			log.Errorf("insecure configuration (tolerated only because dev login is on): %s", p)
		}
		return nil
	}
	if len(problems) > 0 {
		return fmt.Errorf("refusing to start with an insecure configuration: %s", strings.Join(problems, "; "))
	}
	return nil
}
