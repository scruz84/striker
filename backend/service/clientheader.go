package service

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/scruz84/striker/handlers"
)

// ClientHeader / ClientHeaderValue is the custom header every state-changing request must carry.
// A cross-site page (form post, <img>, fetch without credentials tricks) cannot add a custom
// header without a CORS preflight, which the origin allow-list rejects - so the header is proof
// the request came from the bundled frontend, which is what makes the SameSite=None session
// cookie safe against CSRF.
const (
	ClientHeader      = "X-Striker-Client"
	ClientHeaderValue = "web"
)

// RequireClientHeader rejects POST/PUT/PATCH/DELETE requests that lack ClientHeader: web with
// 403. Safe methods (GET, HEAD, OPTIONS - including CORS preflights) pass, as does the login
// endpoint, which carries its own credential and has no session to ride on.
func RequireClientHeader() gin.HandlerFunc {
	return func(c *gin.Context) {
		switch c.Request.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
			if c.Request.URL.Path != "/v1/login" && c.GetHeader(ClientHeader) != ClientHeaderValue {
				c.AbortWithStatusJSON(http.StatusForbidden, handlers.GetError(0, "missing "+ClientHeader+" header"))
				return
			}
		}
		c.Next()
	}
}
