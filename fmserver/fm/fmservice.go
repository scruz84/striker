package fm

import (
	"errors"
	"fmt"
	"footmanager/fmcommon"
	"footmanager/fmserver/fm/handlers"
	ginjwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/postgres"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"strconv"
	"time"
)

func StartSvc() {
	log.Info("Starting football manager service!")
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)

	err := godotenv.Load("app.env")
	if err != nil {
		log.Fatalf("error reading app.env file: %s", err.Error())
	}

	port, _ := strconv.Atoi(os.Getenv("DB_PORT"))
	databaseDetails := fmcommon.DatabaseDetails{
		Host:     os.Getenv("DB_HOST"),
		Port:     port,
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		DbName:   os.Getenv("DB_DATABASE"),
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

	errorListener := router.Run("0.0.0.0:1984")
	//errorListener := router.Run("localhost:1984")
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
	router.Use(CORSMiddleware())
	router.Use(sessions.Sessions("footballManager", store))

	//V1 API
	v1Group := router.Group("/v1")

	authMiddleware := initJWTMiddleware(dbCtx)

	//Authenticated endpoints
	v1UserGroup := v1Group.Group("/user")
	v1UserGroup.Use(authMiddleware.MiddlewareFunc())
	v1UserGroup.GET("/team", handlers.UserTeam(dbCtx))

	v1TeamGroup := v1Group.Group("/teams")
	v1TeamGroup.Use(authMiddleware.MiddlewareFunc())
	v1TeamGroup.GET("/team/:"+handlers.ID+"/", handlers.Team(dbCtx))

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
	preparedStmt, _ = dbCtx.Db.Prepare(fmcommon.GetTeamStmtByIdSql)
	dbCtx.PreparedStmts[fmcommon.GetTeamStmtById] = preparedStmt
}

// Initializes the authentication middleware
func initJWTMiddleware(dbCtx *fmcommon.DatabaseContext) *ginjwt.GinJWTMiddleware {
	authMiddleware, err := ginjwt.New(&ginjwt.GinJWTMiddleware{
		Realm:          os.Getenv("REALM"), //"Football manager",
		Key:            []byte(os.Getenv("SECRET_JWT")),
		Timeout:        time.Minute * 10,
		MaxRefresh:     time.Hour,
		IdentityKey:    fmcommon.UserId,
		SendCookie:     true,
		SecureCookie:   false, //non HTTPS dev environments
		TokenLookup:    "cookie:jwt",
		CookieHTTPOnly: true,                       // JS can't modify
		CookieDomain:   os.Getenv("COOKIE_DOMAIN"), //"localhost",
		CookieName:     "jwt",
		//CookieSameSite: http.SameSiteDefaultMode, //SameSiteDefaultMode, SameSiteLaxMode, SameSiteStrictMode, SameSiteNoneMode
		CookieSameSite: http.SameSiteNoneMode, //SameSiteDefaultMode, SameSiteLaxMode, SameSiteStrictMode, SameSiteNoneMode
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
			log.Infof("Authorizator ... %s", c)
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

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, "+
			"X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
