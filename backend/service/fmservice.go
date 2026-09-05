package service

import (
	"errors"
	"fmt"
	ginjwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/postgres"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/scruz84/striker/common"
	"github.com/scruz84/striker/handlers"
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

	if err := godotenv.Load("app.env"); err != nil {
		log.Warnf("no app.env file loaded, relying on environment variables: %s", err.Error())
	}

	port, _ := strconv.Atoi(os.Getenv("DB_PORT"))
	databaseDetails := common.DatabaseDetails{
		Host:     os.Getenv("DB_HOST"),
		Port:     port,
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		DbName:   os.Getenv("DB_DATABASE"),
		SslMode:  "disable",
	}
	dbCtx, err := common.OpenDatabase(databaseDetails)
	err = dbCtx.Db.Ping()
	if err != nil {
		log.Panicf("Error opening database: %s", err)
	}

	router := gin.Default()
	initRouter(router, dbCtx)

	errorListener := router.Run("0.0.0.0:1984")
	//errorListener := router.Run("localhost:1984")
	if errorListener != nil {
		log.Panicf("ERROR starting listener: %s", errorListener)
	}

	defer dbCtx.Db.Close()
}

// Http router initializer
func initRouter(router *gin.Engine, dbCtx *common.DatabaseContext) {
	//CORS
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = []string{"https://api.striker.example.com",
		"https://striker.example.com",
		"http://localhost:3000"}
	corsConfig.AllowCredentials = true
	corsConfig.AddAllowMethods("OPTIONS")
	corsConfig.AddAllowHeaders("Access-Control-Allow-Origin")
	corsConfig.AddAllowHeaders("Access-Control-Allow-Headers",
		"Content-Type", "Content-Length", "Accept-Encoding", "X-CSRF-Token", "Authorization", "accept", "origin",
		"Cache-Control", "X-Requested-With")
	router.Use(cors.New(corsConfig))

	//Init session management
	store, err := postgres.NewStore(dbCtx.Db, []byte(os.Getenv("SESSION_SECRET")))
	if err != nil {
		log.Panicf("Error accessing database for session management: %s", err)
	}
	router.Use(sessions.Sessions("footballManager", store))

	//V1 API
	v1Group := router.Group("/v1")

	authMiddleware := initJWTMiddleware(dbCtx)

	//Authenticated endpoints
	v1UserGroup := v1Group.Group("/user")
	v1UserGroup.Use(authMiddleware.MiddlewareFunc())
	v1UserGroup.GET("/team", handlers.UserTeam(dbCtx))
	v1UserGroup.GET("/team/lineup", handlers.GetLineupHandler(dbCtx))
	v1UserGroup.PUT("/team/lineup", handlers.SetLineupHandler(dbCtx))
	v1UserGroup.GET("/team/finances", handlers.GetFinancesHandler(dbCtx))
	v1UserGroup.PUT("/team/stadium/ticket-price", handlers.SetTicketPriceHandler(dbCtx))
	v1UserGroup.POST("/team/stadium/upgrade", handlers.UpgradeStadiumHandler(dbCtx))
	v1UserGroup.POST("/team/players/:"+handlers.ID+"/list", handlers.ListPlayerHandler(dbCtx))
	v1UserGroup.DELETE("/team/players/:"+handlers.ID+"/list", handlers.UnlistPlayerHandler(dbCtx))
	v1UserGroup.POST("/team/players/:"+handlers.ID+"/loan-list", handlers.ListPlayerForLoanHandler(dbCtx))
	v1UserGroup.DELETE("/team/players/:"+handlers.ID+"/loan-list", handlers.UnlistPlayerForLoanHandler(dbCtx))
	v1UserGroup.GET("/notifications", handlers.GetNotificationsHandler(dbCtx))
	v1UserGroup.POST("/notifications/:"+handlers.ID+"/read", handlers.MarkNotificationReadHandler(dbCtx))
	v1UserGroup.POST("/notifications/read-all", handlers.MarkAllNotificationsReadHandler(dbCtx))
	v1UserGroup.GET("/notification-preferences", handlers.GetNotificationPreferencesHandler(dbCtx))
	v1UserGroup.PUT("/notification-preferences", handlers.SetNotificationPreferencesHandler(dbCtx))

	v1TransferGroup := v1Group.Group("/transfers")
	v1TransferGroup.Use(authMiddleware.MiddlewareFunc())
	v1TransferGroup.GET("", handlers.BrowseTransfersHandler(dbCtx))
	v1TransferGroup.POST("/:"+handlers.ID+"/buy", handlers.BuyTransferHandler(dbCtx))
	v1TransferGroup.GET("/window", handlers.TransferWindowStatusHandler(dbCtx))

	v1LoanGroup := v1Group.Group("/loans")
	v1LoanGroup.Use(authMiddleware.MiddlewareFunc())
	v1LoanGroup.GET("", handlers.BrowseLoansHandler(dbCtx))
	v1LoanGroup.POST("/:"+handlers.ID+"/accept", handlers.AcceptLoanHandler(dbCtx))

	v1OfferGroup := v1Group.Group("/offers")
	v1OfferGroup.Use(authMiddleware.MiddlewareFunc())
	v1OfferGroup.POST("", handlers.MakeOfferHandler(dbCtx))
	v1OfferGroup.GET("/received", handlers.OffersReceivedHandler(dbCtx))
	v1OfferGroup.GET("/sent", handlers.OffersSentHandler(dbCtx))
	v1OfferGroup.DELETE("/:"+handlers.ID, handlers.WithdrawOfferHandler(dbCtx))
	v1OfferGroup.POST("/:"+handlers.ID+"/accept", handlers.AcceptOfferHandler(dbCtx))
	v1OfferGroup.POST("/:"+handlers.ID+"/reject", handlers.RejectOfferHandler(dbCtx))

	v1FriendGroup := v1Group.Group("/groups")
	v1FriendGroup.Use(authMiddleware.MiddlewareFunc())
	v1FriendGroup.POST("", handlers.CreateGroupHandler(dbCtx))
	v1FriendGroup.POST("/join", handlers.JoinGroupHandler(dbCtx))
	v1FriendGroup.GET("/mine", handlers.MyGroupsHandler(dbCtx))
	v1FriendGroup.GET("/:"+handlers.ID, handlers.GroupDetailsHandler(dbCtx))
	v1FriendGroup.POST("/:"+handlers.ID+"/leave", handlers.LeaveGroupHandler(dbCtx))
	v1FriendGroup.DELETE("/:"+handlers.ID, handlers.DeleteGroupHandler(dbCtx))

	v1TeamGroup := v1Group.Group("/teams")
	v1TeamGroup.Use(authMiddleware.MiddlewareFunc())
	v1TeamGroup.GET("/match/:"+handlers.ID+"/", handlers.MatchDetailsHandler(dbCtx))
	v1TeamGroup.GET("/match/:"+handlers.ID+"/live", handlers.LiveMatchHandler(dbCtx))
	v1TeamGroup.GET("/division/:"+handlers.ID+"/", handlers.DivisionDetailsHandler(dbCtx))
	v1TeamGroup.GET("/division/:"+handlers.ID+"/top-scorers", handlers.TopScorersHandler(dbCtx))
	v1TeamGroup.GET("/team/:"+handlers.ID+"/", handlers.Team(dbCtx))
	v1TeamGroup.GET("/team/:"+handlers.ID+"/history", handlers.TeamHistoryHandler(dbCtx))
	v1TeamGroup.GET("/head-to-head/:teamA/:teamB", handlers.HeadToHeadHandler(dbCtx))

	v1CupGroup := v1Group.Group("/cups")
	v1CupGroup.Use(authMiddleware.MiddlewareFunc())
	v1CupGroup.GET("/latest/:country", handlers.LatestCupHandler(dbCtx))
	v1CupGroup.GET("/:"+handlers.ID, handlers.CupDetailsHandler(dbCtx))
	v1CupGroup.GET("/ties/:"+handlers.ID+"/events", handlers.CupTieEventsHandler(dbCtx))
	v1CupGroup.GET("/ties/:"+handlers.ID+"/live", handlers.LiveCupTieHandler(dbCtx))

	//Authentication endpoints
	v1AuthGroup := v1Group.Group("/auth")
	v1AuthGroup.Use(authMiddleware.MiddlewareFunc())
	v1AuthGroup.GET("/me", handlers.Me(dbCtx, common.UserId))
	v1AuthGroup.GET("/refresh_token", authMiddleware.RefreshHandler)
	v1AuthGroup.POST("/logout", authMiddleware.LogoutHandler)

	//Commissioner/admin tooling - requires is_admin, see handlers.RequireAdmin
	v1AdminGroup := v1Group.Group("/admin")
	v1AdminGroup.Use(authMiddleware.MiddlewareFunc())
	v1AdminGroup.Use(handlers.RequireAdmin(dbCtx))
	v1AdminGroup.GET("/overview", handlers.AdminOverviewHandler(dbCtx))
	v1AdminGroup.GET("/users", handlers.AdminListUsersHandler(dbCtx))
	v1AdminGroup.POST("/simulate", handlers.AdminSimulateNowHandler(dbCtx))
	v1AdminGroup.POST("/advance-season", handlers.AdminAdvanceSeasonHandler(dbCtx))
	v1AdminGroup.POST("/teams/:"+handlers.ID+"/budget", handlers.AdminAdjustBudgetHandler(dbCtx))
	v1AdminGroup.PUT("/transfer-window", handlers.AdminSetTransferWindowHandler(dbCtx))
	v1AdminGroup.PUT("/match-schedule", handlers.AdminSetMatchScheduleHandler(dbCtx))

	//Others V1
	v1Group.GET("/initData", handlers.Initialize(dbCtx))
	v1Group.POST("/login", authMiddleware.LoginHandler)
	v1Group.POST("/registerUser", handlers.RegisterUser(dbCtx))
	v1Group.GET("/teams", handlers.Teams(dbCtx))
}

// Initializes the authentication middleware
func initJWTMiddleware(dbCtx *common.DatabaseContext) *ginjwt.GinJWTMiddleware {
	var timeout string
	timeout = os.Getenv("SESSION_TIMEOUT")
	if timeout == "" {
		timeout = "30m"
	}
	sessionTimeout, _ := time.ParseDuration(timeout)
	authMiddleware, err := ginjwt.New(&ginjwt.GinJWTMiddleware{
		Realm:          os.Getenv("REALM"), //"Football manager",
		Key:            []byte(os.Getenv("SECRET_JWT")),
		Timeout:        sessionTimeout,
		MaxRefresh:     time.Hour,
		IdentityKey:    common.UserId,
		SendCookie:     true,
		SecureCookie:   true, //non HTTPS dev environments
		TokenLookup:    "cookie:jwt",
		CookieHTTPOnly: true, // JS can't modify
		CookieName:     "jwt",
		CookieSameSite: http.SameSiteNoneMode, //SameSiteDefaultMode, SameSiteLaxMode, SameSiteStrictMode, SameSiteNoneMode
		PayloadFunc: func(data interface{}) ginjwt.MapClaims {
			if u, ok := data.(*common.LoggedUser); ok {
				return ginjwt.MapClaims{
					common.UserId:     u.Id,
					common.PictureUrl: u.PictureUrl,
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
			//log.Infof("Authorizator ... %s", c)
			var loginRequest common.LoginRequest
			if err := c.ShouldBind(&loginRequest); err != nil {
				return "", ginjwt.ErrMissingLoginValues
			}
			loggedUser, _, err := handlers.DoExternalLogin(dbCtx, &loginRequest)
			if err != nil {
				return "", errors.New(fmt.Sprint(err))
			}
			session := sessions.Default(c)
			session.Set(common.UserId, loggedUser.Id)
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
