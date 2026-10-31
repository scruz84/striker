package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	ginjwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/scruz84/striker/common"
	"google.golang.org/api/idtoken"
)

// A forged cookie on a route the JWT middleware does not guard yields no identity: identity comes
// only from claims the middleware verified, never from the raw cookie.
func TestGetUserIdFromCookie_IgnoresUnverifiedCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	token := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{common.UserId: "1", "exp": time.Now().Add(time.Hour).Unix()})
	signed, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatal(err)
	}
	for name, value := range map[string]string{"unsigned": signed, "garbage": "not-a-jwt", "empty": ""} {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		c.Request.AddCookie(&http.Cookie{Name: common.COOKIE, Value: value})
		if id := GetUserIdFromCookie(c); id != nil {
			t.Errorf("%s cookie: got user id %d, want nil", name, *id)
		}
		if p := GetPictureUrlFromCookie(c); p != nil {
			t.Errorf("%s cookie: got picture %q, want nil", name, *p)
		}
	}
}

func TestGetUserIdFromCookie_ReadsVerifiedClaims(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Set("JWT_PAYLOAD", ginjwt.MapClaims{common.UserId: "42", common.PictureUrl: "http://pic"})
	id := GetUserIdFromCookie(c)
	if id == nil || *id != 42 {
		t.Fatalf("user id = %v, want 42", id)
	}
	if p := GetPictureUrlFromCookie(c); p == nil || *p != "http://pic" {
		t.Fatalf("picture = %v, want http://pic", p)
	}
}

func fakeGoogleValidator(payload *idtoken.Payload, err error) func(context.Context, string, string) (*idtoken.Payload, error) {
	return func(_ context.Context, _ string, _ string) (*idtoken.Payload, error) { return payload, err }
}

func TestVerifyGoogleIdToken(t *testing.T) {
	original := validateGoogleIdToken
	defer func() { validateGoogleIdToken = original }()
	t.Setenv("GOOGLE_OAUTH_CLIENT_ID", "my-client.apps.googleusercontent.com")

	good := func() *idtoken.Payload {
		return &idtoken.Payload{
			Issuer:   "https://accounts.google.com",
			Audience: "my-client.apps.googleusercontent.com",
			Claims: map[string]interface{}{
				"email": "a@b.c", "email_verified": true, "name": "Ann", "picture": "http://p",
			},
		}
	}
	withClaim := func(k string, v interface{}) *idtoken.Payload { p := good(); p.Claims[k] = v; return p }
	withIssuer := func(i string) *idtoken.Payload { p := good(); p.Issuer = i; return p }

	tests := []struct {
		name    string
		payload *idtoken.Payload
		err     error
		want    int
	}{
		{"valid payload", good(), nil, http.StatusOK},
		{"bare issuer spelling", withIssuer("accounts.google.com"), nil, http.StatusOK},
		{"wrong audience or bad signature (validator error)", nil, errors.New("idtoken: audience provided does not match aud claim in the JWT"), http.StatusForbidden},
		{"invalid signature", nil, errors.New("idtoken: invalid token, token must have three segments"), http.StatusForbidden},
		{"unverified email", withClaim("email_verified", false), nil, http.StatusForbidden},
		{"missing email_verified", withClaim("email_verified", nil), nil, http.StatusForbidden},
		{"foreign issuer", withIssuer("https://evil.example"), nil, http.StatusForbidden},
		{"no email", withClaim("email", ""), nil, http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validateGoogleIdToken = fakeGoogleValidator(tt.payload, tt.err)
			identity, status, err := verifyGoogleIdToken(context.Background(), "token")
			if status != tt.want {
				t.Fatalf("status = %d (err %v), want %d", status, err, tt.want)
			}
			if tt.want == http.StatusOK && (err != nil || identity.Email != "a@b.c" || identity.Name != "Ann" || identity.Picture != "http://p") {
				t.Fatalf("identity = %+v, err %v", identity, err)
			}
			if tt.want != http.StatusOK && identity != nil {
				t.Fatalf("a rejected token must yield no identity, got %+v", identity)
			}
		})
	}
}

func TestVerifyGoogleIdToken_PassesConfiguredAudience(t *testing.T) {
	original := validateGoogleIdToken
	defer func() { validateGoogleIdToken = original }()
	t.Setenv("GOOGLE_OAUTH_CLIENT_ID", "configured-client")
	var gotAudience string
	validateGoogleIdToken = func(_ context.Context, _ string, audience string) (*idtoken.Payload, error) {
		gotAudience = audience
		return nil, errors.New("stop")
	}
	_, _, _ = verifyGoogleIdToken(context.Background(), "token")
	if gotAudience != "configured-client" {
		t.Errorf("validated against audience %q, want the configured client id", gotAudience)
	}
}

func TestVerifyGoogleIdToken_FailsClosedWithoutClientId(t *testing.T) {
	original := validateGoogleIdToken
	defer func() { validateGoogleIdToken = original }()
	t.Setenv("GOOGLE_OAUTH_CLIENT_ID", "")
	called := false
	validateGoogleIdToken = func(context.Context, string, string) (*idtoken.Payload, error) {
		called = true
		return &idtoken.Payload{}, nil
	}
	if _, status, _ := verifyGoogleIdToken(context.Background(), "token"); status != http.StatusForbidden || called {
		t.Errorf("without a client id: status %d, validator called %v; want 403 and not called", status, called)
	}
}
