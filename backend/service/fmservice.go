package service

import (
	"errors"
	"fmt"
	ginjwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/postgres"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	"github.com/scruz84/striker/common"
	"github.com/scruz84/striker/handlers"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
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

	// Fails closed: weak or placeholder secrets and a missing Google client id stop the service
	// unless dev login is on (then they are only logged). See common.CheckDeploymentSecurity.
	if err := common.CheckDeploymentSecurity(); err != nil {
		log.Fatalf("%s", err)
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

	// Applies any migration added since this database was last started - see common.EnsureSchema.
	if err := common.EnsureSchema(dbCtx); err != nil {
		log.Panicf("Error ensuring schema is up to date: %s", err)
	}

	common.ConfigureWebPushNotifier(dbCtx)

	if err := common.RegisterDBPoolMetrics(dbCtx.Db); err != nil {
		log.Errorf("error registering database pool metrics: %s", err)
	}

	// RegisterMarketGauges is deliberately not called here - see its doc comment. Only
	// engine.StartEngine registers it, so a normal svc+engine deployment (see
	// docker-compose.yml) exports each gauge once instead of once per process.

	// Backfills the trainer market for a world that existed before the trainer feature shipped -
	// see common.TopUpTrainerMarket.
	if err := common.TopUpTrainerMarket(dbCtx); err != nil {
		log.Errorf("error topping up trainer market at startup: %s", err)
	}
	// Assigns an AI level to any unowned team that existed before the AI-teams feature shipped -
	// see common.BackfillAiLevels.
	if err := common.BackfillAiLevels(dbCtx); err != nil {
		log.Errorf("error backfilling AI levels at startup: %s", err)
	}
	// Clears a leftover CPU AI level on any team a user already claimed before AssignTeam started
	// resetting it itself - see common.ClearOwnedTeamAiLevels.
	if err := common.ClearOwnedTeamAiLevels(dbCtx); err != nil {
		log.Errorf("error clearing owned team AI levels at startup: %s", err)
	}
	if err := common.BackfillAiStyles(dbCtx); err != nil {
		log.Errorf("error backfilling CPU club styles at startup: %s", err)
	}
	// Creates a national team for any active country that existed before this feature shipped -
	// see common.BackfillNationalTeams.
	if err := common.BackfillNationalTeams(dbCtx); err != nil {
		log.Errorf("error backfilling national teams at startup: %s", err)
	}

	router := gin.Default()
	router.Use(metricsMiddleware())
	router.Use(gzipMiddleware())
	initRouter(router, dbCtx)

	errorListener := router.Run("0.0.0.0:1984")
	//errorListener := router.Run("localhost:1984")
	if errorListener != nil {
		log.Panicf("ERROR starting listener: %s", errorListener)
	}

	defer dbCtx.Db.Close()
}

// metricsMiddleware records every request's count and duration (see
// common.HTTPRequestsCounter/HTTPRequestDuration) - a no-op unless OpenTelemetry metrics export
// is enabled (common.InitMetrics). Uses c.FullPath() rather than c.Request.URL.Path so a
// parameterized route (e.g. /v1/teams/player/:id/) stays one series regardless of which id was
// requested - an unmatched route (404) has no full path, labeled "unmatched" instead of leaving
// the attribute empty. A websocket upgrade (the live match viewer) is counted but its duration is
// not recorded: the connection stays open for the whole match, which would bury the latency of
// real requests in the histogram.
func metricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}
		attrs := metric.WithAttributes(
			attribute.String("method", c.Request.Method),
			attribute.String("route", route),
			attribute.Int("status", c.Writer.Status()),
		)
		ctx := c.Request.Context()
		common.HTTPRequestsCounter.Add(ctx, 1, attrs)
		if websocket.IsWebSocketUpgrade(c.Request) {
			return
		}
		common.HTTPRequestDuration.Record(ctx, float64(time.Since(start).Microseconds())/1000, attrs)
	}
}

// Http router initializer
func initRouter(router *gin.Engine, dbCtx *common.DatabaseContext) {
	//CORS
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = common.CorsAllowedOrigins()
	corsConfig.AllowCredentials = true
	corsConfig.AddAllowMethods("OPTIONS")
	corsConfig.AddAllowHeaders("Access-Control-Allow-Origin")
	corsConfig.AddAllowHeaders("Access-Control-Allow-Headers",
		"Content-Type", "Content-Length", "Accept-Encoding", "X-CSRF-Token", "Authorization", "accept", "origin",
		"Cache-Control", "X-Requested-With", ClientHeader)
	router.Use(cors.New(corsConfig))
	// After CORS, so a preflight is answered by it first. See RequireClientHeader.
	router.Use(RequireClientHeader())

	//Init session management
	store, err := postgres.NewStore(dbCtx.Db, []byte(os.Getenv("SESSION_SECRET")))
	if err != nil {
		log.Panicf("Error accessing database for session management: %s", err)
	}
	router.Use(sessions.Sessions("footballManager", store))

	registerV1Routes(router, dbCtx)
}

// registerV1Routes mounts the whole /v1 API. Split from initRouter (CORS, session store) so tests
// can exercise the real route table - which routes exist and which middleware guards them -
// without a database.
func registerV1Routes(router *gin.Engine, dbCtx *common.DatabaseContext) {
	//V1 API
	v1Group := router.Group("/v1")

	authMiddleware := initJWTMiddleware(dbCtx)
	// Runs after the JWT check on every authenticated group: records activity and refuses the
	// still-valid token of a deleted account.
	activity := handlers.ActivityMiddleware(dbCtx)

	//Authenticated endpoints
	v1UserGroup := v1Group.Group("/user")
	v1UserGroup.Use(authMiddleware.MiddlewareFunc())
	v1UserGroup.Use(activity)
	v1UserGroup.GET("/team", handlers.UserTeam(dbCtx))
	v1UserGroup.PUT("/team/name", handlers.RenameTeamHandler(dbCtx))
	v1UserGroup.GET("/team/lineup", handlers.GetLineupHandler(dbCtx))
	v1UserGroup.PUT("/team/lineup", handlers.SetLineupHandler(dbCtx))
	v1UserGroup.GET("/team/lineup/friendly", handlers.GetFriendlyDefaultLineupHandler(dbCtx))
	v1UserGroup.PUT("/team/lineup/friendly", handlers.SetFriendlyDefaultLineupHandler(dbCtx))
	v1UserGroup.GET("/team/finances", handlers.GetFinancesHandler(dbCtx))
	v1UserGroup.PUT("/team/stadium/ticket-price", handlers.SetTicketPriceHandler(dbCtx))
	v1UserGroup.PUT("/team/stadium/name", handlers.RenameStadiumHandler(dbCtx))
	v1UserGroup.PUT("/team/sponsor-deal", handlers.SetSponsorDealHandler(dbCtx))
	v1UserGroup.POST("/team/stadium/upgrade", handlers.UpgradeStadiumHandler(dbCtx))
	v1UserGroup.POST("/team/players/:"+handlers.ID+"/list", handlers.ListPlayerHandler(dbCtx))
	v1UserGroup.DELETE("/team/players/:"+handlers.ID+"/list", handlers.UnlistPlayerHandler(dbCtx))
	v1UserGroup.POST("/team/players/:"+handlers.ID+"/loan-list", handlers.ListPlayerForLoanHandler(dbCtx))
	v1UserGroup.DELETE("/team/players/:"+handlers.ID+"/loan-list", handlers.UnlistPlayerForLoanHandler(dbCtx))
	v1UserGroup.POST("/team/players/:"+handlers.ID+"/exercise-loan-buy-option", handlers.ExerciseLoanBuyOptionHandler(dbCtx))
	v1UserGroup.POST("/team/players/:"+handlers.ID+"/renew-contract", handlers.RenewContractHandler(dbCtx))
	v1UserGroup.POST("/team/players/:"+handlers.ID+"/release", handlers.ReleasePlayerHandler(dbCtx))
	v1UserGroup.GET("/watchlist", handlers.GetWatchlistHandler(dbCtx))
	v1UserGroup.PUT("/watchlist/:"+handlers.ID, handlers.WatchPlayerHandler(dbCtx))
	v1UserGroup.DELETE("/watchlist/:"+handlers.ID, handlers.UnwatchPlayerHandler(dbCtx))
	v1UserGroup.GET("/team/academy", handlers.GetAcademyHandler(dbCtx))
	v1UserGroup.POST("/team/academy/:"+handlers.ID+"/promote", handlers.PromoteFromAcademyHandler(dbCtx))
	v1UserGroup.GET("/notifications", handlers.GetNotificationsHandler(dbCtx))
	v1UserGroup.POST("/notifications/:"+handlers.ID+"/read", handlers.MarkNotificationReadHandler(dbCtx))
	v1UserGroup.POST("/notifications/read-all", handlers.MarkAllNotificationsReadHandler(dbCtx))
	v1UserGroup.DELETE("/notifications/:"+handlers.ID, handlers.DeleteNotificationHandler(dbCtx))
	v1UserGroup.DELETE("/notifications", handlers.DeleteAllNotificationsHandler(dbCtx))
	v1UserGroup.GET("/notification-preferences", handlers.GetNotificationPreferencesHandler(dbCtx))
	v1UserGroup.PUT("/notification-preferences", handlers.SetNotificationPreferencesHandler(dbCtx))
	v1UserGroup.POST("/push-subscriptions", handlers.SubscribeToPushHandler(dbCtx))
	v1UserGroup.DELETE("/push-subscriptions", handlers.UnsubscribeFromPushHandler(dbCtx))
	v1UserGroup.PUT("/language", handlers.SetUserLanguageHandler(dbCtx))
	v1UserGroup.GET("/account", handlers.GetAccountHandler(dbCtx))
	v1UserGroup.DELETE("/account", handlers.DeleteAccountHandler(dbCtx))
	v1UserGroup.GET("/account/export", handlers.ExportAccountHandler(dbCtx))

	v1TransferGroup := v1Group.Group("/transfers")
	v1TransferGroup.Use(authMiddleware.MiddlewareFunc())
	v1TransferGroup.Use(activity)
	v1TransferGroup.GET("", handlers.BrowseTransfersHandler(dbCtx))
	v1TransferGroup.POST("/:"+handlers.ID+"/buy", handlers.BuyTransferHandler(dbCtx))
	v1TransferGroup.GET("/window", handlers.TransferWindowStatusHandler(dbCtx))
	v1TransferGroup.GET("/stats", handlers.TransferMarketStatsHandler(dbCtx))

	v1LoanGroup := v1Group.Group("/loans")
	v1LoanGroup.Use(authMiddleware.MiddlewareFunc())
	v1LoanGroup.Use(activity)
	v1LoanGroup.GET("", handlers.BrowseLoansHandler(dbCtx))
	v1LoanGroup.POST("/:"+handlers.ID+"/accept", handlers.AcceptLoanHandler(dbCtx))

	v1FreeAgentGroup := v1Group.Group("/free-agents")
	v1FreeAgentGroup.Use(authMiddleware.MiddlewareFunc())
	v1FreeAgentGroup.Use(activity)
	v1FreeAgentGroup.GET("", handlers.BrowseFreeAgentsHandler(dbCtx))
	v1FreeAgentGroup.POST("/:"+handlers.ID+"/sign", handlers.SignFreeAgentHandler(dbCtx))

	v1TrainerGroup := v1Group.Group("/trainers")
	v1TrainerGroup.Use(authMiddleware.MiddlewareFunc())
	v1TrainerGroup.Use(activity)
	v1TrainerGroup.GET("", handlers.BrowseTrainersHandler(dbCtx))
	v1TrainerGroup.GET("/mine", handlers.TeamTrainerHandler(dbCtx))
	v1TrainerGroup.POST("/:"+handlers.ID+"/hire", handlers.HireTrainerHandler(dbCtx))
	v1TrainerGroup.POST("/:"+handlers.ID+"/fire", handlers.FireTrainerHandler(dbCtx))

	v1OfferGroup := v1Group.Group("/offers")
	v1OfferGroup.Use(authMiddleware.MiddlewareFunc())
	v1OfferGroup.Use(activity)
	v1OfferGroup.POST("", handlers.MakeOfferHandler(dbCtx))
	v1OfferGroup.GET("/received", handlers.OffersReceivedHandler(dbCtx))
	v1OfferGroup.GET("/sent", handlers.OffersSentHandler(dbCtx))
	v1OfferGroup.DELETE("/:"+handlers.ID, handlers.WithdrawOfferHandler(dbCtx))
	v1OfferGroup.POST("/:"+handlers.ID+"/accept", handlers.AcceptOfferHandler(dbCtx))
	v1OfferGroup.POST("/:"+handlers.ID+"/reject", handlers.RejectOfferHandler(dbCtx))
	v1OfferGroup.POST("/:"+handlers.ID+"/counter", handlers.CounterOfferHandler(dbCtx))
	v1OfferGroup.POST("/:"+handlers.ID+"/accept-counter", handlers.AcceptCounterOfferHandler(dbCtx))
	v1OfferGroup.POST("/:"+handlers.ID+"/decline-counter", handlers.DeclineCounterOfferHandler(dbCtx))

	v1FriendGroup := v1Group.Group("/groups")
	v1FriendGroup.Use(authMiddleware.MiddlewareFunc())
	v1FriendGroup.Use(activity)
	v1FriendGroup.POST("", handlers.CreateGroupHandler(dbCtx))
	v1FriendGroup.POST("/join", handlers.JoinGroupHandler(dbCtx))
	v1FriendGroup.GET("/mine", handlers.MyGroupsHandler(dbCtx))
	v1FriendGroup.GET("/:"+handlers.ID, handlers.GroupDetailsHandler(dbCtx))
	v1FriendGroup.POST("/:"+handlers.ID+"/leave", handlers.LeaveGroupHandler(dbCtx))
	v1FriendGroup.DELETE("/:"+handlers.ID, handlers.DeleteGroupHandler(dbCtx))

	v1TeamGroup := v1Group.Group("/teams")
	v1TeamGroup.Use(authMiddleware.MiddlewareFunc())
	v1TeamGroup.Use(activity)
	v1TeamGroup.GET("/match/:"+handlers.ID+"/", handlers.MatchDetailsHandler(dbCtx))
	v1TeamGroup.GET("/match/:"+handlers.ID+"/live", handlers.LiveMatchHandler(dbCtx))
	v1TeamGroup.GET("/division/:"+handlers.ID+"/", handlers.DivisionDetailsHandler(dbCtx))
	v1TeamGroup.GET("/division/:"+handlers.ID+"/top-scorers", handlers.TopScorersHandler(dbCtx))
	v1TeamGroup.GET("/division/:"+handlers.ID+"/weeks/:week", handlers.LeagueWeekMatchesHandler(dbCtx))
	v1TeamGroup.GET("/division/:"+handlers.ID+"/live", handlers.LeagueLiveScoresHandler(dbCtx))
	v1TeamGroup.GET("/division/:"+handlers.ID+"/seasons", handlers.LeagueSeasonsHandler(dbCtx))
	v1TeamGroup.GET("/division/:"+handlers.ID+"/seasons/:season", handlers.LeagueSeasonTableHandler(dbCtx))
	v1TeamGroup.GET("/division/:"+handlers.ID+"/recent-transfers", handlers.RecentLeagueTransfersHandler(dbCtx))
	v1TeamGroup.GET("/leagues", handlers.LeagueDirectoryHandler(dbCtx))
	v1TeamGroup.GET("/search", handlers.SearchHandler(dbCtx))
	v1TeamGroup.GET("/team/:"+handlers.ID+"/", handlers.Team(dbCtx))
	v1TeamGroup.GET("/team/:"+handlers.ID+"/history", handlers.TeamHistoryHandler(dbCtx))
	v1TeamGroup.GET("/team/:"+handlers.ID+"/matches", handlers.TeamMatchesHandler(dbCtx))
	v1TeamGroup.GET("/team/:"+handlers.ID+"/rivalries", handlers.TeamRivalriesHandler(dbCtx))
	v1TeamGroup.GET("/player/:"+handlers.ID+"/", handlers.PlayerDetailsHandler(dbCtx))
	v1TeamGroup.GET("/player/:"+handlers.ID+"/transfer-comparables", handlers.TransferComparablesHandler(dbCtx))
	v1TeamGroup.GET("/team/evolutions", handlers.TeamLatestEvolutionsHandler(dbCtx))
	v1TeamGroup.GET("/head-to-head/:teamA/:teamB", handlers.HeadToHeadHandler(dbCtx))
	v1TeamGroup.GET("/matches/live", handlers.LiveMatchesOverviewHandler(dbCtx))
	v1TeamGroup.GET("/preview/:"+handlers.KindParam+"/:"+handlers.ID, handlers.OpponentPreviewHandler(dbCtx))
	v1TeamGroup.PUT("/match/:"+handlers.ID+"/lineup", handlers.SetMatchLineupHandler(dbCtx))
	v1TeamGroup.GET("/match/:"+handlers.ID+"/lineup", handlers.GetMatchLineupHandler(dbCtx))

	v1FriendlyGroup := v1Group.Group("/friendlies")
	v1FriendlyGroup.Use(authMiddleware.MiddlewareFunc())
	v1FriendlyGroup.Use(activity)
	v1FriendlyGroup.POST("/challenges", handlers.ChallengeToFriendlyHandler(dbCtx))
	v1FriendlyGroup.GET("/challenges/received", handlers.FriendlyChallengesReceivedHandler(dbCtx))
	v1FriendlyGroup.GET("/challenges/sent", handlers.FriendlyChallengesSentHandler(dbCtx))
	v1FriendlyGroup.POST("/challenges/:"+handlers.ID+"/accept", handlers.AcceptFriendlyChallengeHandler(dbCtx))
	v1FriendlyGroup.POST("/challenges/:"+handlers.ID+"/reject", handlers.RejectFriendlyChallengeHandler(dbCtx))
	v1FriendlyGroup.DELETE("/challenges/:"+handlers.ID, handlers.WithdrawFriendlyChallengeHandler(dbCtx))
	v1FriendlyGroup.GET("/upcoming", handlers.UpcomingFriendliesHandler(dbCtx))
	v1FriendlyGroup.GET("/recent", handlers.RecentFriendliesHandler(dbCtx))
	v1FriendlyGroup.GET("/schedule", handlers.FriendlyScheduleHandler(dbCtx))
	v1FriendlyGroup.GET("/:"+handlers.ID, handlers.FriendlyMatchHandler(dbCtx))
	v1FriendlyGroup.GET("/:"+handlers.ID+"/live", handlers.LiveFriendlyHandler(dbCtx))
	v1FriendlyGroup.PUT("/:"+handlers.ID+"/lineup", handlers.SetFriendlyLineupHandler(dbCtx))
	v1FriendlyGroup.GET("/:"+handlers.ID+"/lineup", handlers.GetFriendlyLineupHandler(dbCtx))

	v1NationalTeamGroup := v1Group.Group("/national-teams")
	v1NationalTeamGroup.Use(authMiddleware.MiddlewareFunc())
	v1NationalTeamGroup.Use(activity)
	v1NationalTeamGroup.GET("/round", handlers.NationalTeamRoundHandler(dbCtx))
	v1NationalTeamGroup.GET("/results", handlers.NationalTeamResultsHandler(dbCtx))
	v1NationalTeamGroup.GET("/world-cup/current", handlers.CurrentWorldCupHandler(dbCtx))
	v1NationalTeamGroup.GET("/world-cups", handlers.WorldCupHistoryHandler(dbCtx))

	v1CupGroup := v1Group.Group("/cups")
	v1CupGroup.Use(authMiddleware.MiddlewareFunc())
	v1CupGroup.Use(activity)
	v1CupGroup.GET("/latest/:country", handlers.LatestCupHandler(dbCtx))
	v1CupGroup.GET("/latest-international/:region", handlers.LatestInternationalCupHandler(dbCtx))
	v1CupGroup.GET("/:"+handlers.ID, handlers.CupDetailsHandler(dbCtx))
	v1CupGroup.GET("/ties/:"+handlers.ID+"/events", handlers.CupTieEventsHandler(dbCtx))
	v1CupGroup.GET("/ties/:"+handlers.ID+"/live", handlers.LiveCupTieHandler(dbCtx))
	v1CupGroup.GET("/ties/:"+handlers.ID, handlers.CupTieDetailsHandler(dbCtx))
	v1CupGroup.PUT("/ties/:"+handlers.ID+"/lineup", handlers.SetCupTieLineupHandler(dbCtx))
	v1CupGroup.GET("/ties/:"+handlers.ID+"/lineup", handlers.GetCupTieLineupHandler(dbCtx))

	//Authentication endpoints
	v1AuthGroup := v1Group.Group("/auth")
	v1AuthGroup.Use(authMiddleware.MiddlewareFunc())
	v1AuthGroup.Use(activity)
	v1AuthGroup.GET("/me", handlers.Me(dbCtx, common.UserId))
	v1AuthGroup.GET("/refresh_token", authMiddleware.RefreshHandler)
	v1AuthGroup.POST("/logout", authMiddleware.LogoutHandler)
	v1AuthGroup.GET("/onboarding/countries", handlers.OnboardingCountriesHandler(dbCtx))
	v1AuthGroup.POST("/onboarding/country", handlers.SelectCountryHandler(dbCtx))
	v1AuthGroup.POST("/accept-terms", handlers.AcceptTermsHandler(dbCtx))

	//Commissioner/admin tooling - requires is_admin, see handlers.RequireAdmin
	v1AdminGroup := v1Group.Group("/admin")
	v1AdminGroup.Use(authMiddleware.MiddlewareFunc())
	v1AdminGroup.Use(activity)
	v1AdminGroup.Use(handlers.RequireAdmin(dbCtx))
	v1AdminGroup.GET("/overview", handlers.AdminOverviewHandler(dbCtx))
	v1AdminGroup.GET("/users", handlers.AdminListUsersHandler(dbCtx))
	v1AdminGroup.POST("/simulate", handlers.AdminSimulateNowHandler(dbCtx))
	v1AdminGroup.POST("/finish-stuck-matches", handlers.AdminFinishStuckMatchesHandler(dbCtx))
	v1AdminGroup.POST("/pay-wages", handlers.AdminPayWagesNowHandler(dbCtx))
	v1AdminGroup.POST("/advance-season", handlers.AdminAdvanceSeasonHandler(dbCtx))
	v1AdminGroup.POST("/inject-free-agents", handlers.AdminInjectFreeAgentsNowHandler(dbCtx))
	v1AdminGroup.PUT("/free-agent-injections-per-week", handlers.AdminSetFreeAgentInjectionsPerWeekHandler(dbCtx))
	v1AdminGroup.PUT("/offer-negotiation-window", handlers.AdminSetOfferNegotiationWindowHandler(dbCtx))
	v1AdminGroup.POST("/teams/:"+handlers.ID+"/budget", handlers.AdminAdjustBudgetHandler(dbCtx))
	v1AdminGroup.POST("/teams/inject-all", handlers.AdminInjectAllTeamsBudgetHandler(dbCtx))
	v1AdminGroup.PUT("/teams/:"+handlers.ID+"/ai-level", handlers.AdminSetTeamAiLevelHandler(dbCtx))
	v1AdminGroup.PUT("/transfer-window", handlers.AdminSetTransferWindowHandler(dbCtx))
	v1AdminGroup.PUT("/match-schedule", handlers.AdminSetMatchScheduleHandler(dbCtx))
	v1AdminGroup.PUT("/promotion-relegation", handlers.AdminSetPromotionRelegationHandler(dbCtx))
	v1AdminGroup.PUT("/max-friendlies-per-week", handlers.AdminSetMaxFriendliesPerWeekHandler(dbCtx))
	v1AdminGroup.PUT("/friendly-schedule", handlers.AdminSetFriendlyScheduleHandler(dbCtx))
	v1AdminGroup.PUT("/users/:"+handlers.ID+"/admin", handlers.AdminSetUserAdminHandler(dbCtx))
	v1AdminGroup.DELETE("/users/:"+handlers.ID, handlers.AdminDeleteUserHandler(dbCtx))
	v1AdminGroup.GET("/world-config", handlers.AdminGetWorldConfigHandler(dbCtx))
	v1AdminGroup.PUT("/world-config", handlers.AdminSetWorldConfigHandler(dbCtx))
	v1AdminGroup.POST("/recreate-leagues", handlers.AdminRecreateLeaguesHandler(dbCtx))
	// Wipes and reseeds the whole game - admin only, POST so a cross-site GET (an <img> tag)
	// can never trigger it. Replaces the old unauthenticated GET /v1/initData.
	v1AdminGroup.POST("/initialize", handlers.Initialize(dbCtx))
	v1AdminGroup.GET("/countries", handlers.AdminListCountriesHandler(dbCtx))
	v1AdminGroup.POST("/countries", handlers.AdminAddCountryHandler(dbCtx))
	v1AdminGroup.POST("/countries/:code/divisions", handlers.AdminAddDivisionHandler(dbCtx))
	v1AdminGroup.POST("/countries/:code/reschedule", handlers.AdminRescheduleCountryHandler(dbCtx))
	v1AdminGroup.GET("/leagues", handlers.AdminListLeaguesHandler(dbCtx))
	v1AdminGroup.PUT("/leagues/:"+handlers.ID+"/schedule", handlers.AdminSetLeagueScheduleHandler(dbCtx))
	v1AdminGroup.DELETE("/leagues/:"+handlers.ID+"/schedule", handlers.AdminClearLeagueScheduleHandler(dbCtx))

	//Others V1
	v1Group.POST("/login", authMiddleware.LoginHandler)
	v1Group.GET("/dev-login/users", handlers.DevLoginUsersHandler())
}

// Initializes the authentication middleware
func initJWTMiddleware(dbCtx *common.DatabaseContext) *ginjwt.GinJWTMiddleware {
	var timeout string
	timeout = os.Getenv("SESSION_TIMEOUT")
	if timeout == "" {
		timeout = "24h"
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

			var loggedUser *common.LoggedUser
			var err error
			if loginRequest.DevEmail != "" && common.DevLoginEnabled() {
				loggedUser, _, err = handlers.DoDevLogin(dbCtx, loginRequest.DevEmail)
			} else {
				loggedUser, _, err = handlers.DoExternalLogin(dbCtx, &loginRequest)
			}
			if err != nil {
				return "", errors.New(fmt.Sprint(err))
			}
			common.LoginsCounter.Add(c.Request.Context(), 1, metric.WithAttributes(common.LoginProviderAttribute(loggedUser.Provider)))
			// A sign-in counts as activity (it also clears any inactivity warning).
			if loggedId, parseErr := strconv.ParseInt(loggedUser.Id, 10, 64); parseErr == nil {
				if _, touchErr := common.TouchUserActivity(dbCtx, loggedId); touchErr != nil {
					log.Errorf("error recording login activity of user %d: %s", loggedId, touchErr)
				}
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
