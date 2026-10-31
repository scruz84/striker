package handlers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	ginjwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"
	"github.com/scruz84/striker/common"
	"github.com/scruz84/striker/engine"
	log "github.com/sirupsen/logrus"
	"google.golang.org/api/idtoken"
	"net/http"
	"strconv"
)

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

// validateGoogleIdToken checks a Google ID token's signature (against Google's cached public
// certificates), expiry and audience. A variable so tests can return a crafted payload or an
// error without network access.
var validateGoogleIdToken = idtoken.Validate

// googleIssuers are the two spellings Google uses for the `iss` claim of its ID tokens.
var googleIssuers = map[string]bool{
	"accounts.google.com":         true,
	"https://accounts.google.com": true,
}

// googleIdentity is what a validated Google ID token tells us about the person signing in.
type googleIdentity struct {
	Email   string
	Name    string
	Picture string
}

// verifyGoogleIdToken validates an ID token for this deployment (audience = GOOGLE_OAUTH_CLIENT_ID),
// requires a Google issuer and a verified email, and returns the identity from the validated
// payload only - never from claims read without verification. Any failure is 403.
func verifyGoogleIdToken(ctx context.Context, rawToken string) (*googleIdentity, int, error) {
	clientId := common.GoogleOAuthClientId()
	if clientId == "" {
		log.Error("Google sign-in attempted but GOOGLE_OAUTH_CLIENT_ID is not set")
		return nil, http.StatusForbidden, errors.New("google sign-in is not configured")
	}
	payload, err := validateGoogleIdToken(ctx, rawToken, clientId)
	if err != nil {
		log.Warnf("rejected Google ID token: %s", err)
		return nil, http.StatusForbidden, errors.New("invalid Google token")
	}
	if !googleIssuers[payload.Issuer] {
		log.Warnf("rejected Google ID token with issuer %q", payload.Issuer)
		return nil, http.StatusForbidden, errors.New("invalid Google token")
	}
	if verified, _ := payload.Claims["email_verified"].(bool); !verified {
		log.Warnf("rejected Google ID token with an unverified email")
		return nil, http.StatusForbidden, errors.New("Google email is not verified")
	}
	email, _ := payload.Claims["email"].(string)
	if email == "" {
		return nil, http.StatusForbidden, errors.New("Google token has no email")
	}
	name, _ := payload.Claims["name"].(string)
	picture, _ := payload.Claims["picture"].(string)
	return &googleIdentity{Email: email, Name: name, Picture: picture}, http.StatusOK, nil
}

// DoExternalLogin Authenticates against an external service. Currently, only Google supported
func DoExternalLogin(dbCtx *common.DatabaseContext, loginRequest *common.LoginRequest) (*common.LoggedUser, int, error) {
	log.Infof("Starting to login user")

	identity, status, err := verifyGoogleIdToken(context.Background(), loginRequest.Jwt)
	if err != nil {
		return nil, status, err
	}

	//Search for the user
	email := identity.Email
	userId, errUser := common.GetUserIdByEmail(email, dbCtx)
	var loggedUser *common.LoggedUser
	if errUser != nil {
		//the user does not exist. Register it without a country or team yet - the frontend sends
		//them to the onboarding country picker (see handlers.SelectCountryHandler) right after
		//this login completes, and AssignTeam only runs once they've chosen one.
		err = dbCtx.Db.QueryRow(common.RegisterPendingUserStmtSql, identity.Name, "google", email).Scan(&userId)
		if err != nil {
			log.Errorf("error creating new user: %s", err)
			return nil, http.StatusInternalServerError, err
		}
		common.UserRegistrationsCounter.Add(context.Background(), 1)
		loggedUser = &common.LoggedUser{
			Id:                      strconv.FormatInt(userId, 10),
			Name:                    identity.Name,
			Provider:                "google",
			Email:                   email,
			TeamId:                  "0",
			PendingCountrySelection: true,
			TermsVersion:            common.CurrentTermsVersion,
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

	loggedUser.PictureUrl = identity.Picture
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
			TermsVersion:            common.CurrentTermsVersion,
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
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusUnauthorized, common.HttpResponse{Message: "account no longer exists"})
			return
		}
		if err != nil {
			log.Errorf("error loading user %d for onboarding: %s", *userId, err)
			c.JSON(http.StatusInternalServerError, common.HttpResponse{Message: err.Error()})
			return
		}
		if !existing.PendingCountrySelection {
			c.JSON(http.StatusOK, existing)
			return
		}
		// Nobody gets a club without having accepted the Privacy Policy and Terms (and confirmed
		// the minimum age) first - see POST /v1/auth/accept-terms.
		if !existing.TermsAccepted {
			c.JSON(http.StatusBadRequest, common.HttpResponse{Message: "terms must be accepted first"})
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
			if errors.Is(err, common.ErrUserNotFound) {
				c.JSON(http.StatusUnauthorized, common.HttpResponse{Message: "account no longer exists"})
				return
			}
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

// GetUserIdFromCookie returns the signed-in user's id from the claims the JWT middleware already
// verified for this request (signature, expiry). On a route without that middleware - or with a
// forged or missing cookie - there are no verified claims and it returns nil: identity is never
// read from the raw cookie. The name is historical.
func GetUserIdFromCookie(c *gin.Context) *int64 {
	id, ok := ginjwt.ExtractClaims(c)[common.UserId].(string)
	if !ok {
		return nil
	}
	userId, err := common.StringToInt64(id)
	if err != nil {
		return nil
	}
	return &userId
}

// GetPictureUrlFromCookie returns the picture URL carried in the verified claims, or nil (see
// GetUserIdFromCookie).
func GetPictureUrlFromCookie(c *gin.Context) *string {
	if p, ok := ginjwt.ExtractClaims(c)[common.PictureUrl].(string); ok {
		return &p
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
