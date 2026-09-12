package handlers

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/scruz84/striker/common"
	"github.com/scruz84/striker/engine"
	log "github.com/sirupsen/logrus"
	"google.golang.org/api/oauth2/v2"
	"google.golang.org/api/option"
	"net/http"
	"strconv"
)

func RegisterUser(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Infof("[goroutine - %d] - Starting to register a new user", common.CurGoroutineID())
		var user common.RegisterUser
		var err error
		if err = c.BindJSON(&user); err != nil {
			log.Errorf("[goroutine - %d] - error reading user to register: %s", common.CurGoroutineID(), err)
			c.JSON(http.StatusBadRequest, &err)
			return
		}

		var user_id = int64(-1)
		err = dbCtx.Db.QueryRow(common.RegisterUserStmtSql, user.Name, user.Provider, user.Email, user.Country).Scan(&user_id)
		if err != nil {
			log.Errorf("[goroutine - %d] - error inserting new user: %s", common.CurGoroutineID(), err)
			c.JSON(http.StatusBadRequest, common.HttpResponse{Message: err.Error()})
			return
		}
		team, err2 := common.AssignTeam(dbCtx, user_id, user.Country)
		if err2 != nil {
			log.Errorf("[goroutine - %d] - error assigning new team to user: %s", common.CurGoroutineID(), err2)
			c.JSON(http.StatusOK, common.Team{Id: -1})
			return
		} else {
			c.JSON(http.StatusOK, team)
			return
		}
	}
}

func Me(dbCtx *common.DatabaseContext, idKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Infof("Starting to get user details")

		id := GetUserIdFromCookie(c)
		if id == nil {
			c.JSON(http.StatusForbidden, nil)
			return
		}
		//Get user details
		loggedUser, err := common.GetUserDetails(*id, dbCtx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, nil)
			return
		}
		if pictureUrl := GetPictureUrlFromCookie(c); pictureUrl != nil {
			loggedUser.PictureUrl = *pictureUrl
		}
		c.JSON(http.StatusOK, loggedUser)
		return
	}
}

// DoExternalLogin Authenticates against an external service. Currently, only Google supported
func DoExternalLogin(dbCtx *common.DatabaseContext, loginRequest *common.LoginRequest) (*common.LoggedUser, int, error) {
	log.Infof("Starting to login user")

	oauth2Service, err := oauth2.NewService(context.Background(), option.WithoutAuthentication())
	if err != nil {
		log.Errorf("%s", err)
		return nil, http.StatusInternalServerError, err
	}
	tokenInfoCall := oauth2Service.Tokeninfo()
	tokenInfoCall.IdToken(loginRequest.Jwt)
	_, errToken := tokenInfoCall.Do()
	if errToken != nil {
		log.Errorf("%s", errToken)
		return nil, http.StatusForbidden, errToken
	}

	//Extract info from the token
	jwtTokenClaims, _ := TokenClaimsNoValidation(loginRequest.Jwt)

	//Search for the user
	email := fmt.Sprint((*jwtTokenClaims)["email"])
	userId, errUser := common.GetUserIdByEmail(email, dbCtx)
	var loggedUser *common.LoggedUser
	if errUser != nil {
		//the user does not exist. Register it without a country or team yet - the frontend sends
		//them to the onboarding country picker (see handlers.SelectCountryHandler) right after
		//this login completes, and AssignTeam only runs once they've chosen one.
		err = dbCtx.Db.QueryRow(common.RegisterPendingUserStmtSql, (*jwtTokenClaims)["name"], "google", email).Scan(&userId)
		if err != nil {
			log.Errorf("error creating new user: %s", err)
			return nil, http.StatusInternalServerError, err
		}
		common.UserRegistrationsCounter.Add(context.Background(), 1)
		loggedUser = &common.LoggedUser{
			Id:                      strconv.FormatInt(userId, 10),
			Name:                    fmt.Sprint((*jwtTokenClaims)["name"]),
			Provider:                "google",
			Email:                   email,
			TeamId:                  "0",
			PendingCountrySelection: true,
		}
	} else {
		//Get user details
		loggedUser, err = common.GetUserDetails(userId, dbCtx)
		if err != nil {
			return nil, http.StatusInternalServerError, err
		}
	}

	// Admin access is granted purely by config (ADMIN_EMAILS) - there's no admin-managing-admins
	// UI, so this re-checks and persists the promotion on every login rather than only at
	// registration, letting an operator add someone to the list without a manual DB edit.
	if common.IsConfiguredAdminEmail(email) && !loggedUser.IsAdmin {
		if err := common.SetUserAdmin(dbCtx, userId, true); err != nil {
			log.Errorf("error promoting user %d to admin: %s", userId, err)
		} else {
			loggedUser.IsAdmin = true
		}
	}

	loggedUser.PictureUrl = fmt.Sprint((*jwtTokenClaims)["picture"])
	return loggedUser, http.StatusOK, nil
}

// DoDevLogin signs in as one of common.DevUsers with no credential check at all - callers must
// check common.DevLoginEnabled() before calling this (see the Authenticator wiring in
// service.initJWTMiddleware). Auto-registers the account on first use, the same way a
// first-time Google sign-in is registered in DoExternalLogin.
func DoDevLogin(dbCtx *common.DatabaseContext, email string) (*common.LoggedUser, int, error) {
	devUser := common.FindDevUser(email)
	if devUser == nil {
		return nil, http.StatusForbidden, fmt.Errorf("unknown dev user: %s", email)
	}

	userId, errUser := common.GetUserIdByEmail(devUser.Email, dbCtx)
	var loggedUser *common.LoggedUser
	if errUser != nil {
		// Registered without a country/team yet, same as a first-time Google sign-in - the
		// frontend routes them to the onboarding country picker next (see
		// handlers.SelectCountryHandler).
		if err := dbCtx.Db.QueryRow(common.RegisterPendingUserStmtSql, devUser.Name, "dev", devUser.Email).
			Scan(&userId); err != nil {
			log.Errorf("error creating dev user: %s", err)
			return nil, http.StatusInternalServerError, err
		}
		loggedUser = &common.LoggedUser{
			Id:                      strconv.FormatInt(userId, 10),
			Name:                    devUser.Name,
			Provider:                "dev",
			Email:                   devUser.Email,
			TeamId:                  "0",
			PendingCountrySelection: true,
		}
	} else {
		var err error
		loggedUser, err = common.GetUserDetails(userId, dbCtx)
		if err != nil {
			return nil, http.StatusInternalServerError, err
		}
	}

	if (devUser.IsAdmin || common.IsConfiguredAdminEmail(devUser.Email)) && !loggedUser.IsAdmin {
		if err := common.SetUserAdmin(dbCtx, userId, true); err != nil {
			log.Errorf("error promoting dev user %d to admin: %s", userId, err)
		} else {
			loggedUser.IsAdmin = true
		}
	}

	return loggedUser, http.StatusOK, nil
}

// DevLoginUsersHandler lists the dev-login roster for the frontend to offer as sign-in
// shortcuts - an empty list (rather than an error) when common.DevLoginEnabled() is false, so
// the frontend can simply hide the section without a special case.
func DevLoginUsersHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !common.DevLoginEnabled() {
			c.JSON(http.StatusOK, []common.DevUser{})
			return
		}
		c.JSON(http.StatusOK, common.DevUsers)
	}
}

// OnboardingCountriesHandler lists the countries a newly signed-in, not-yet-assigned user can
// pick from - only the ones a world has actually been generated for (common.GetActiveCountries),
// not every country the engine knows how to generate (common.KnownCountries also includes
// admin-activatable ones with no teams yet).
func OnboardingCountriesHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		active, err := common.GetActiveCountries(dbCtx)
		if err != nil {
			log.Errorf("error listing active countries: %s", err)
			c.JSON(http.StatusInternalServerError, common.HttpResponse{Message: err.Error()})
			return
		}
		activeSet := make(map[string]bool, len(active))
		for _, code := range active {
			activeSet[code] = true
		}
		countries := make([]common.CountryInfo, 0, len(active))
		for _, info := range common.KnownCountries {
			if activeSet[info.Code] {
				countries = append(countries, info)
			}
		}
		c.JSON(http.StatusOK, countries)
	}
}

// SelectCountryHandler completes onboarding for a user who registered without a country (see
// RegisterPendingUserStmtSql) - records the country they picked and hands them their first team
// in it (common.AssignTeam), which is what actually decides which division they land in (see
// GetFreeTeamStmtSql). A no-op returning the user's current state if they've already been
// assigned a team, so a retried/duplicate request can't hand someone a second team.
func SelectCountryHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := GetUserIdFromCookie(c)
		if userId == nil {
			c.JSON(http.StatusForbidden, nil)
			return
		}

		var body struct {
			Country string `json:"country"`
		}
		if err := c.BindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, common.HttpResponse{Message: err.Error()})
			return
		}

		existing, err := common.GetUserDetails(*userId, dbCtx)
		if err != nil {
			log.Errorf("error loading user %d for onboarding: %s", *userId, err)
			c.JSON(http.StatusInternalServerError, common.HttpResponse{Message: err.Error()})
			return
		}
		if !existing.PendingCountrySelection {
			c.JSON(http.StatusOK, existing)
			return
		}

		active, err := common.GetActiveCountries(dbCtx)
		if err != nil {
			log.Errorf("error listing active countries: %s", err)
			c.JSON(http.StatusInternalServerError, common.HttpResponse{Message: err.Error()})
			return
		}
		valid := false
		for _, code := range active {
			if code == body.Country {
				valid = true
				break
			}
		}
		if !valid {
			c.JSON(http.StatusBadRequest, common.HttpResponse{Message: "unknown or inactive country"})
			return
		}

		if err := common.SetUserCountry(dbCtx, *userId, body.Country); err != nil {
			log.Errorf("error setting country for user %d: %s", *userId, err)
			c.JSON(http.StatusInternalServerError, common.HttpResponse{Message: err.Error()})
			return
		}
		if _, err := common.AssignTeam(dbCtx, *userId, body.Country); err != nil {
			log.Errorf("error assigning team to user %d: %s", *userId, err)
			c.JSON(http.StatusInternalServerError, common.HttpResponse{Message: err.Error()})
			return
		}

		loggedUser, err := common.GetUserDetails(*userId, dbCtx)
		if err != nil {
			log.Errorf("error reloading user %d after onboarding: %s", *userId, err)
			c.JSON(http.StatusInternalServerError, common.HttpResponse{Message: err.Error()})
			return
		}
		c.JSON(http.StatusOK, loggedUser)
	}
}

func Teams(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		teams := make([]common.Team, 0)
		teams = append(teams, common.Team{Id: 12, TeamName: "Irizar", OwnerId: 1111, Country: "es", DivisionId: 222})
		c.JSON(http.StatusOK, teams)
	}
}

func GetUserIdFromCookie(c *gin.Context) *int64 {
	var userId *int64
	value, _ := c.Cookie(common.COOKIE)
	jwtTokenClaims, _ := TokenClaimsNoValidation(value)
	if jwtTokenClaims != nil {
		id, err := common.StringToInt64((*jwtTokenClaims)[common.UserId].(string))
		if err == nil {
			userId = &id
		}
	}
	return userId
}

func GetPictureUrlFromCookie(c *gin.Context) *string {
	value, _ := c.Cookie(common.COOKIE)
	jwtTokenClaims, _ := TokenClaimsNoValidation(value)
	if jwtTokenClaims != nil {
		p, ok := (*jwtTokenClaims)[common.PictureUrl].(string)
		if ok {
			return &p
		}
	}
	return nil
}

func DivisionDetailsHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Infof("Starting to get division details")
		divIdStr := c.Param(ID)
		log.Infof("Starting to get division details from %s", divIdStr)
		divId, err := common.StringToInt64(divIdStr)
		if err != nil {
			c.JSON(http.StatusInternalServerError, nil)
			return
		}
		divisionDetails, err := engine.GetDivisionDetails(dbCtx, divId)
		if err != nil {
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		c.JSON(http.StatusOK, divisionDetails)
		return
	}
}

func MatchDetailsHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Infof("Starting to get match details")
		matchIdStr := c.Param(ID)
		log.Infof("Starting to get match from %s", matchIdStr)
		matchId, err := common.StringToInt64(matchIdStr)
		if err != nil {
			c.JSON(http.StatusInternalServerError, nil)
			return
		}
		matchDetails, err := engine.GetMatchDetails(dbCtx, matchId)
		if err != nil {
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		c.JSON(http.StatusOK, matchDetails)
		return
	}
}
