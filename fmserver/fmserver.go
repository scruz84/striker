package main

import (
	"errors"
	"fmt"
	"footmanager/fmcommon"
	"footmanager/fmserver/fm/handlers"
	ginjwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/postgres"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"time"
)

func main() {
	log.Info("Starting football manager!")
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)

	databaseDetails := fmcommon.DatabaseDetails{
		Host:     "localhost",
		Port:     5432,
		User:     "sergio",
		Password: "sergio",
		DbName:   "footmanager",
		SslMode:  "disable",
	}
	dbCtx, err := fmcommon.OpenDatabase(databaseDetails)
	err = dbCtx.Db.Ping()
	if err != nil {
		log.Panicln("Error opening database: %s", err)
	}

	router := gin.Default()
	initRouter(router, dbCtx)
	initStatements(dbCtx)

	errorListener := router.Run("localhost:1984")
	if errorListener != nil {
		log.Panicf("ERROR starting listener: %s", errorListener)
	}

	defer dbCtx.Db.Close()
}

// Http router initializer
func initRouter(router *gin.Engine, dbCtx *fmcommon.DatabaseContext) {
	//Init session management
	store, err := postgres.NewStore(dbCtx.Db, []byte("sessions secret"))
	if err != nil {
		log.Panicln("Error accessing database for session management: %s", err)
	}
	router.Use(sessions.Sessions("footballManager", store))

	//V1 API
	v1Group := router.Group("/v1")

	authMiddleware := initJWTMiddleware(dbCtx)

	//Authenticated endpoints
	v1UserGroup := v1Group.Group("/user")
	v1UserGroup.Use(authMiddleware.MiddlewareFunc())
	v1UserGroup.GET("/team", handlers.UserTeam(dbCtx))

	//Authentication endpoints
	v1AuthGroup := v1Group.Group("/auth")
	v1AuthGroup.Use(authMiddleware.MiddlewareFunc())
	v1AuthGroup.GET("/me", handlers.Me(dbCtx, fmcommon.UserId))
	v1AuthGroup.GET("/refresh_token", authMiddleware.RefreshHandler)
	v1AuthGroup.POST("/logout", authMiddleware.LogoutHandler)

	//Others V1
	v1Group.GET("/division/:"+handlers.ID+"/", handlers.DivisionDetailsHandler(dbCtx))
	v1Group.GET("/initData", handlers.Initialize(dbCtx))
	v1Group.POST("/login", authMiddleware.LoginHandler)
	v1Group.POST("/registerUser", handlers.RegisterUser(dbCtx))
	v1Group.GET("/teams", handlers.Teams(dbCtx))
}

// Initialize database statements
func initStatements(dbCtx *fmcommon.DatabaseContext) {
	var preparedStmt, _ = dbCtx.Db.Prepare(fmcommon.RegisterUserStmtSql)
	dbCtx.PreparedStmts[fmcommon.RegisterUserStmt] = preparedStmt
	preparedStmt, _ = dbCtx.Db.Prepare(fmcommon.GetUserStmtSql)
	dbCtx.PreparedStmts[fmcommon.GetUserStmt] = preparedStmt
	preparedStmt, _ = dbCtx.Db.Prepare(fmcommon.GetUserIdByEmailStmtSql)
	dbCtx.PreparedStmts[fmcommon.GetUserIdByEmailStmt] = preparedStmt
	preparedStmt, _ = dbCtx.Db.Prepare(fmcommon.GetPlayersFromTeamStmtSql)
	dbCtx.PreparedStmts[fmcommon.GetPlayersFromTeamStmt] = preparedStmt
	preparedStmt, _ = dbCtx.Db.Prepare(fmcommon.GetTeamDetailsStmtByOwnerSql)
	dbCtx.PreparedStmts[fmcommon.GetTeamDetailsStmtByOwner] = preparedStmt
	preparedStmt, _ = dbCtx.Db.Prepare(fmcommon.GetDivisionDetailsStmtSql)
	dbCtx.PreparedStmts[fmcommon.GetDivisionDetailsStmt] = preparedStmt
}

// Initializes the authentication middleware
func initJWTMiddleware(dbCtx *fmcommon.DatabaseContext) *ginjwt.GinJWTMiddleware {
	authMiddleware, err := ginjwt.New(&ginjwt.GinJWTMiddleware{
		Realm:          "Football manager",
		Key:            []byte("secret key"),
		Timeout:        time.Minute * 10,
		MaxRefresh:     time.Hour,
		IdentityKey:    fmcommon.UserId,
		SendCookie:     true,
		SecureCookie:   false, //non HTTPS dev environments
		TokenLookup:    "cookie:jwt",
		CookieHTTPOnly: true, // JS can't modify
		CookieDomain:   "localhost",
		CookieName:     "jwt",
		CookieSameSite: http.SameSiteDefaultMode, //SameSiteDefaultMode, SameSiteLaxMode, SameSiteStrictMode, SameSiteNoneMode
		PayloadFunc: func(data interface{}) ginjwt.MapClaims {
			if u, ok := data.(*fmcommon.LoggedUser); ok {
				return ginjwt.MapClaims{
					fmcommon.UserId:     u.Id,
					fmcommon.PictureUrl: u.PictureUrl,
				}
			}
			return ginjwt.MapClaims{}
		},

		IdentityHandler: func(c *gin.Context) interface{} {
			claims := ginjwt.ExtractClaims(c)
			return claims
			//return &User{
			//	UserName: claims[identityKey].(string),
			//}
		},

		Authenticator: func(c *gin.Context) (interface{}, error) {
			var loginRequest fmcommon.LoginRequest
			if err := c.ShouldBind(&loginRequest); err != nil {
				return "", ginjwt.ErrMissingLoginValues
			}
			loggedUser, _, err := handlers.DoExternalLogin(dbCtx, &loginRequest)
			if err != nil {
				return "", errors.New(fmt.Sprint(err))
			}
			session := sessions.Default(c)
			session.Set(fmcommon.UserId, loggedUser.Id)
			err = session.Save()
			if err != nil {
				log.Errorf("error storing session: %s", err)
			}
			return loggedUser, nil
		},

		Authorizator: func(data interface{}, c *gin.Context) bool {
			//if v, ok := data.(*User); ok && v.UserName == "admin" {
			//	return true
			//}
			return true
		},

		Unauthorized: func(c *gin.Context, code int, message string) {
			c.JSON(code, gin.H{
				"code":    code,
				"message": message,
			})
		},

		TokenHeadName: "Bearer",
		TimeFunc:      time.Now,
	})

	if err != nil {
		log.Panicf("Error initializing authentication middleware: %s", err)
	}

	return authMiddleware
}
