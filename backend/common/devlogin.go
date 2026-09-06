package common

import (
	"os"
	"strconv"
	"strings"
)

// DevLoginEnabled reports whether the password-less dev login shortcut (see DevUsers,
// handlers.DoDevLogin) is turned on. Off by default - a real deployment should never set this,
// since anyone who can reach the API could sign in as any of DevUsers with no credential check
// at all. It exists purely so local development and automated testing can exercise the app
// without a working Google OAuth setup.
func DevLoginEnabled() bool {
	enabled, _ := strconv.ParseBool(strings.TrimSpace(os.Getenv("DEV_LOGIN_ENABLED")))
	return enabled
}

// DevUser is one entry of the fixed dev-login roster - see DevUsers.
type DevUser struct {
	Email   string `json:"email"`
	Name    string `json:"name"`
	IsAdmin bool   `json:"isAdmin"`
}

// DevUsers is the fixed set of accounts the dev login shortcut can sign in as: one admin plus a
// few plain users, so multi-user flows (transfers, head-to-head, groups) can be exercised
// locally too. Each is auto-registered the first time it's used, the same way a first-time
// Google sign-in is in DoExternalLogin - see handlers.DoDevLogin.
var DevUsers = []DevUser{
	{Email: "admin@dev.local", Name: "Dev Admin", IsAdmin: true},
	{Email: "user1@dev.local", Name: "Dev User One"},
	{Email: "user2@dev.local", Name: "Dev User Two"},
	{Email: "user3@dev.local", Name: "Dev User Three"},
}

// FindDevUser looks up a DevUsers entry by email (case-insensitive), or nil if email isn't on
// the roster.
func FindDevUser(email string) *DevUser {
	for _, u := range DevUsers {
		if strings.EqualFold(u.Email, email) {
			return &u
		}
	}
	return nil
}
