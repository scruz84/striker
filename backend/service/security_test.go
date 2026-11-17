package service

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/scruz84/striker/common"
	"github.com/scruz84/striker/handlers"
)

const testSecret = "0123456789abcdef0123456789abcdef-test"

// securityRouter mounts the real /v1 route table with a nil database: nothing in these tests
// may reach a handler that touches it, which is exactly what is being asserted.
func securityRouter(t *testing.T) *gin.Engine {
	t.Helper()
	t.Setenv("SECRET_JWT", testSecret)
	t.Setenv("REALM", "test")
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequireClientHeader())
	registerV1Routes(router, &common.DatabaseContext{})
	return router
}

// forgedCookie is a session cookie claiming to be user 1, signed with a key the server does not
// use (an attacker's guess at the admin's id).
func forgedCookie(t *testing.T, key string) *http.Cookie {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		common.UserId: "1",
		"exp":         time.Now().Add(time.Hour).Unix(),
	})
	signed, err := token.SignedString([]byte(key))
	if err != nil {
		t.Fatal(err)
	}
	return &http.Cookie{Name: "jwt", Value: signed}
}

func do(router *gin.Engine, method, path string, header bool, cookie *http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader("{}"))
	if header {
		req.Header.Set(ClientHeader, ClientHeaderValue)
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestInitialize_RejectsForgedCookie(t *testing.T) {
	router := securityRouter(t)
	for name, key := range map[string]string{"wrong key": "attacker-key", "empty key": ""} {
		rec := do(router, http.MethodPost, "/v1/admin/initialize", true, forgedCookie(t, key))
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s: POST /v1/admin/initialize with a forged cookie = %d, want 401", name, rec.Code)
		}
	}
	if rec := do(router, http.MethodPost, "/v1/admin/initialize", true, nil); rec.Code != http.StatusUnauthorized {
		t.Errorf("no cookie = %d, want 401", rec.Code)
	}
}

// The instant archive setting is admin-only: without a valid session it is refused before any
// handler runs (openspec/specs/match-data-retention "Not an admin").
func TestInstantArchive_RequiresAdminSession(t *testing.T) {
	router := securityRouter(t)
	for name, cookie := range map[string]*http.Cookie{"no cookie": nil, "forged cookie": forgedCookie(t, "attacker-key")} {
		if rec := do(router, http.MethodPut, "/v1/admin/instant-archive", true, cookie); rec.Code != http.StatusUnauthorized {
			t.Errorf("%s: PUT /v1/admin/instant-archive = %d, want 401", name, rec.Code)
		}
	}
}

func TestOldUnauthenticatedEndpointsAreGone(t *testing.T) {
	router := securityRouter(t)
	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/v1/initData"},
		{http.MethodPost, "/v1/initData"},
		{http.MethodPost, "/v1/registerUser"},
		{http.MethodGet, "/v1/teams"},
	} {
		if rec := do(router, tc.method, tc.path, true, forgedCookie(t, "attacker-key")); rec.Code != http.StatusNotFound {
			t.Errorf("%s %s = %d, want 404", tc.method, tc.path, rec.Code)
		}
	}
}

func TestRequireClientHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequireClientHeader())
	ok := func(c *gin.Context) { c.Status(http.StatusNoContent) }
	for _, p := range []string{"/v1/login", "/v1/user/team/players/1/release"} {
		router.POST(p, ok)
	}
	router.GET("/v1/user/team", ok)
	router.PUT("/v1/user/team/name", ok)
	router.DELETE("/v1/user/notifications", ok)
	router.OPTIONS("/v1/user/team/players/1/release", ok)

	tests := []struct {
		method, path string
		header       bool
		want         int
	}{
		{http.MethodPost, "/v1/user/team/players/1/release", false, http.StatusForbidden},
		{http.MethodPost, "/v1/user/team/players/1/release", true, http.StatusNoContent},
		{http.MethodPut, "/v1/user/team/name", false, http.StatusForbidden},
		{http.MethodPut, "/v1/user/team/name", true, http.StatusNoContent},
		{http.MethodDelete, "/v1/user/notifications", false, http.StatusForbidden},
		{http.MethodDelete, "/v1/user/notifications", true, http.StatusNoContent},
		{http.MethodGet, "/v1/user/team", false, http.StatusNoContent},
		{http.MethodOptions, "/v1/user/team/players/1/release", false, http.StatusNoContent},
		{http.MethodPost, "/v1/login", false, http.StatusNoContent},
	}
	for _, tt := range tests {
		req := httptest.NewRequest(tt.method, tt.path, nil)
		if tt.header {
			req.Header.Set(ClientHeader, ClientHeaderValue)
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != tt.want {
			t.Errorf("%s %s (header=%v) = %d, want %d", tt.method, tt.path, tt.header, rec.Code, tt.want)
		}
	}

	// A wrong value is as bad as none.
	req := httptest.NewRequest(http.MethodPost, "/v1/user/team/players/1/release", nil)
	req.Header.Set(ClientHeader, "mobile")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("wrong header value = %d, want 403", rec.Code)
	}
}

// A still-valid JWT whose user was deleted gets 401 (and an expired cookie) on reads and cannot
// recreate data through the onboarding endpoint - the deleted-id map answers before any query,
// so the nil database is never touched.
func TestDeletedUsersSessionIsRefused(t *testing.T) {
	router := securityRouter(t)
	handlers.Activity.MarkDeleted(7)
	session := signedCookie(t, "7")

	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/v1/auth/me"},
		{http.MethodGet, "/v1/user/team"},
		{http.MethodGet, "/v1/user/account"},
		{http.MethodDelete, "/v1/user/account"},
		{http.MethodPost, "/v1/auth/onboarding/country"},
		{http.MethodPost, "/v1/auth/accept-terms"},
	} {
		rec := do(router, tc.method, tc.path, true, session)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s for a deleted user = %d, want 401", tc.method, tc.path, rec.Code)
		}
		if !strings.Contains(strings.Join(rec.Header().Values("Set-Cookie"), ";"), "jwt=;") {
			t.Errorf("%s %s should expire the session cookie, got %v", tc.method, tc.path, rec.Header().Values("Set-Cookie"))
		}
	}
}

// signedCookie is a valid session cookie for userId, signed with the server's key.
func signedCookie(t *testing.T, userId string) *http.Cookie {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		common.UserId: userId,
		"exp":         time.Now().Add(time.Hour).Unix(),
		"orig_iat":    time.Now().Unix(),
	})
	signed, err := token.SignedString([]byte(testSecret))
	if err != nil {
		t.Fatal(err)
	}
	return &http.Cookie{Name: "jwt", Value: signed}
}

func TestNewAccountEndpointsRequireASession(t *testing.T) {
	router := securityRouter(t)
	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/v1/user/account"},
		{http.MethodDelete, "/v1/user/account"},
		{http.MethodGet, "/v1/user/account/export"},
		{http.MethodPost, "/v1/auth/accept-terms"},
		{http.MethodDelete, "/v1/admin/users/5"},
	} {
		if rec := do(router, tc.method, tc.path, true, nil); rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s without a session = %d, want 401", tc.method, tc.path, rec.Code)
		}
		if rec := do(router, tc.method, tc.path, true, forgedCookie(t, "attacker-key")); rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s with a forged session = %d, want 401", tc.method, tc.path, rec.Code)
		}
	}
	// And without the CSRF header the state-changing ones never reach a handler.
	if rec := do(router, http.MethodDelete, "/v1/user/account", false, signedCookie(t, "7")); rec.Code != http.StatusForbidden {
		t.Errorf("DELETE /v1/user/account without the client header = %d, want 403", rec.Code)
	}
}
