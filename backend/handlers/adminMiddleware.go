package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/scruz84/striker/common"
	"net/http"
)

// RequireAdmin blocks any request from a user who isn't flagged is_admin - see
// common.IsConfiguredAdminEmail for the only way that flag gets set. Must run after the JWT
// auth middleware, since it relies on the same cookie that middleware already validated.
func RequireAdmin(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := GetUserIdFromCookie(c)
		if userId == nil {
			c.AbortWithStatusJSON(http.StatusForbidden, nil)
			return
		}

		isAdmin, err := common.IsUserAdmin(dbCtx, *userId)
		if err != nil || !isAdmin {
			c.AbortWithStatusJSON(http.StatusForbidden, GetError(0, "admin access required"))
			return
		}

		c.Next()
	}
}
