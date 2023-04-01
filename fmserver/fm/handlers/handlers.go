package handlers

import (
	"context"
	"fmt"
	"footmanager/fmcommon"
	"footmanager/fmserver/fm/engine"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"google.golang.org/api/oauth2/v2"
	"google.golang.org/api/option"
	"net/http"
	"strconv"
)

func RegisterUser(dbCtx *fmcommon.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Infof("[goroutine - %d] - Starting to register a new user", fmcommon.CurGoroutineID())
		var user fmcommon.RegisterUser
		var err error
		if err = c.BindJSON(&user); err != nil {
			log.Errorf("[goroutine - %d] - error reading user to register %s: %s", fmcommon.CurGoroutineID(),
				c, err)
			c.JSON(http.StatusBadRequest, &err)
			return
		}

		var user_id = int64(-1)
		err = dbCtx.PreparedStmts[fmcommon.RegisterUserStmt].QueryRow(user.Name, user.Provider, user.Email, user.Country).Scan(&user_id)
		if err != nil {
			log.Errorf("[goroutine - %d] - error inserting new user: %s", fmcommon.CurGoroutineID(), err)
			c.JSON(http.StatusBadRequest, fmcommon.HttpResponse{Message: err.Error()})
			return
		}
		team, err2 := fmcommon.AssignTeam(dbCtx, user_id, user.Country)
		if err2 != nil {
			log.Errorf("[goroutine - %d] - error assigning new team to user: %s", fmcommon.CurGoroutineID(), err2)
			c.JSON(http.StatusOK, fmcommon.Team{Id: -1})
			return
		} else {
			c.JSON(http.StatusOK, team)
			return
		}
	}
}

func Me(dbCtx *fmcommon.DatabaseContext, idKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Infof("Starting to get user details")

		GetUserIdFromCookie(c)
		id := GetUserIdFromCookie(c)
		if id == nil {
			c.JSON(http.StatusForbidden, nil)
			return
		}
		//Get user details
		loggedUser, err := fmcommon.GetUserDetails(*id, dbCtx)
		loggedUser.PictureUrl = *GetPictureUrlFromCookie(c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, nil)
			return
		}
		c.JSON(http.StatusOK, loggedUser)
		return
	}
}

// DoExternalLogin Authenticates against an external service. Currently, only Google supported
func DoExternalLogin(dbCtx *fmcommon.DatabaseContext, loginRequest *fmcommon.LoginRequest) (*fmcommon.LoggedUser, int, error) {
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
	userId, errUser := fmcommon.GetUserIdByEmail(email, dbCtx)
	var loggedUser *fmcommon.LoggedUser
	if errUser != nil {
		//the user does not exist. Register it
		err = dbCtx.PreparedStmts[fmcommon.RegisterUserStmt].QueryRow((*jwtTokenClaims)["name"], "google", email, "es").Scan(&userId)
		if err != nil {
			log.Errorf("error creating new user: %s", err)
			return nil, http.StatusInternalServerError, err
		}
		team, err := fmcommon.AssignTeam(dbCtx, userId, "es")
		if err != nil {
			log.Errorf("error assigning new team to user: %s", err)
			return nil, http.StatusInternalServerError, err
		}
		loggedUser = &fmcommon.LoggedUser{
			Id:         strconv.FormatInt(userId, 10),
			Name:       fmt.Sprint((*jwtTokenClaims)["name"]),
			Provider:   "google",
			Email:      email,
			Country:    "es",
			TeamId:     strconv.FormatInt(team.Id, 10),
			TeamName:   team.TeamName,
			TeamLeague: strconv.FormatInt(team.DivisionId, 10),
		}
	} else {
		//Get user details
		loggedUser, err = fmcommon.GetUserDetails(userId, dbCtx)
		if err != nil {
			return nil, http.StatusInternalServerError, err
		}
	}
	loggedUser.PictureUrl = fmt.Sprint((*jwtTokenClaims)["picture"])
	return loggedUser, http.StatusOK, nil
}

func Teams(dbCtx *fmcommon.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		teams := make([]fmcommon.Team, 1)
		teams = append(teams, fmcommon.Team{Id: 12, TeamName: "Irizar", OwnerId: 1111, Country: "es", DivisionId: 222})
		c.JSON(http.StatusOK, teams)
	}
}

func GetUserIdFromCookie(c *gin.Context) *int64 {
	var userId *int64
	value, _ := c.Cookie(fmcommon.COOKIE)
	jwtTokenClaims, _ := TokenClaimsNoValidation(value)
	if jwtTokenClaims != nil {
		id, err := fmcommon.StringToInt64((*jwtTokenClaims)[fmcommon.UserId].(string))
		if err == nil {
			userId = &id
		}
	}
	return userId
}

func GetPictureUrlFromCookie(c *gin.Context) *string {
	value, _ := c.Cookie(fmcommon.COOKIE)
	jwtTokenClaims, _ := TokenClaimsNoValidation(value)
	if jwtTokenClaims != nil {
		p, ok := (*jwtTokenClaims)[fmcommon.PictureUrl].(string)
		if ok {
			return &p
		}
	}
	return nil
}

func DivisionDetailsHandler(dbCtx *fmcommon.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Infof("Starting to get division details")
		divIdStr := c.Param(ID)
		log.Infof("Starting to get division details from %s", divIdStr)
		divId, err := fmcommon.StringToInt64(divIdStr)
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
