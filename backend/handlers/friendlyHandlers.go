package handlers

import (
	"database/sql"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
	"net/http"
	"time"
)

type ChallengeToFriendlyRequest struct {
	OpponentTeamId int64 `json:"opponentTeamId"`
	// ProposedKickoff optionally suggests when the challenger would like to play - left nil to
	// leave the decision entirely to the opponent, same as before this field existed.
	ProposedKickoff *time.Time `json:"proposedKickoff"`
}

// ChallengeToFriendlyHandler proposes a friendly match to another team on behalf of the
// caller's team. If the opponent is unowned, common.ChallengeToFriendly accepts on their
// behalf immediately; otherwise it waits for the opponent's owner to accept/reject.
func ChallengeToFriendlyHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := GetUserIdFromCookie(c)
		if userId == nil {
			c.JSON(http.StatusForbidden, nil)
			return
		}

		var req ChallengeToFriendlyRequest
		if err := c.BindJSON(&req); err != nil || req.OpponentTeamId <= 0 {
			c.JSON(http.StatusBadRequest, GetError(0, "opponentTeamId is required"))
			return
		}

		team, err := common.GetTeamByOwner(*userId, dbCtx)
		if err != nil {
			c.JSON(http.StatusNotFound, GetError(0, "team not found for user"))
			return
		}

		if err := common.ChallengeToFriendly(dbCtx, team.Id, req.OpponentTeamId, req.ProposedKickoff); err != nil {
			switch {
			case errors.Is(err, common.ErrCannotChallengeOwnTeam),
				errors.Is(err, common.ErrFriendlyChallengeAlreadyExists),
				errors.Is(err, common.ErrFriendlyLimitReached),
				errors.Is(err, common.ErrInvalidFriendlyKickoff):
				c.JSON(http.StatusConflict, GetError(0, err.Error()))
				return
			}
			log.Errorf("error challenging team %d to a friendly from team %d: %s", req.OpponentTeamId, team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		c.JSON(http.StatusOK, common.HttpResponse{Message: "friendly challenge sent"})
	}
}

// FriendlyChallengesReceivedHandler lists every pending friendly challenge made to the caller's team.
func FriendlyChallengesReceivedHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := GetUserIdFromCookie(c)
		if userId == nil {
			c.JSON(http.StatusForbidden, nil)
			return
		}
		team, err := common.GetTeamByOwner(*userId, dbCtx)
		if err != nil {
			c.JSON(http.StatusNotFound, GetError(0, "team not found for user"))
			return
		}
		challenges, err := common.GetFriendlyChallengesReceived(dbCtx, team.Id)
		if err != nil {
			log.Errorf("error getting friendly challenges received for team %d: %s", team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		c.JSON(http.StatusOK, challenges)
	}
}

// FriendlyChallengesSentHandler lists every pending friendly challenge the caller's team has made.
func FriendlyChallengesSentHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := GetUserIdFromCookie(c)
		if userId == nil {
			c.JSON(http.StatusForbidden, nil)
			return
		}
		team, err := common.GetTeamByOwner(*userId, dbCtx)
		if err != nil {
			c.JSON(http.StatusNotFound, GetError(0, "team not found for user"))
			return
		}
		challenges, err := common.GetFriendlyChallengesSent(dbCtx, team.Id)
		if err != nil {
			log.Errorf("error getting friendly challenges sent for team %d: %s", team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		c.JSON(http.StatusOK, challenges)
	}
}

type AcceptFriendlyChallengeRequest struct {
	Kickoff time.Time `json:"kickoff" binding:"required"`
}

// AcceptFriendlyChallengeHandler agrees to play a pending challenge made to the caller's team,
// at the kickoff date/time the accepting manager chose (see common.AcceptFriendlyChallenge for
// the constraints it must satisfy - far enough ahead, on an allowed weekday).
func AcceptFriendlyChallengeHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := GetUserIdFromCookie(c)
		if userId == nil {
			c.JSON(http.StatusForbidden, nil)
			return
		}
		challengeId, err := common.StringToInt64(c.Param(ID))
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid challenge id"))
			return
		}
		var req AcceptFriendlyChallengeRequest
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "kickoff is required"))
			return
		}
		team, err := common.GetTeamByOwner(*userId, dbCtx)
		if err != nil {
			c.JSON(http.StatusNotFound, GetError(0, "team not found for user"))
			return
		}

		err = common.AcceptFriendlyChallenge(dbCtx, challengeId, team.Id, req.Kickoff)
		switch {
		case err == nil:
			c.JSON(http.StatusOK, common.HttpResponse{Message: "friendly challenge accepted"})
		case errors.Is(err, common.ErrFriendlyChallengeNotFound):
			c.JSON(http.StatusNotFound, GetError(0, err.Error()))
		case errors.Is(err, common.ErrNotChallengeRecipient):
			c.JSON(http.StatusForbidden, GetError(0, err.Error()))
		case errors.Is(err, common.ErrFriendlyLimitReached), errors.Is(err, common.ErrInvalidFriendlyKickoff):
			c.JSON(http.StatusConflict, GetError(0, err.Error()))
		default:
			log.Errorf("error accepting friendly challenge %d for team %d: %s", challengeId, team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
		}
	}
}

// FriendlyScheduleHandler returns the admin-configured allowed-weekdays bitmask for friendly
// kickoffs, so the accept-challenge UI can restrict the date/time picker to days that will
// actually be accepted (see common.GetFriendlyScheduleWeekdays).
func FriendlyScheduleHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"weekdaysMask": common.GetFriendlyScheduleWeekdays(dbCtx)})
	}
}

// RejectFriendlyChallengeHandler declines a pending challenge made to the caller's team.
func RejectFriendlyChallengeHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := GetUserIdFromCookie(c)
		if userId == nil {
			c.JSON(http.StatusForbidden, nil)
			return
		}
		challengeId, err := common.StringToInt64(c.Param(ID))
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid challenge id"))
			return
		}
		team, err := common.GetTeamByOwner(*userId, dbCtx)
		if err != nil {
			c.JSON(http.StatusNotFound, GetError(0, "team not found for user"))
			return
		}

		if err := common.RejectFriendlyChallenge(dbCtx, challengeId, team.Id); err != nil {
			switch {
			case errors.Is(err, common.ErrFriendlyChallengeNotFound):
				c.JSON(http.StatusNotFound, GetError(0, err.Error()))
			case errors.Is(err, common.ErrNotChallengeRecipient):
				c.JSON(http.StatusForbidden, GetError(0, err.Error()))
			default:
				log.Errorf("error rejecting friendly challenge %d: %s", challengeId, err)
				c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			}
			return
		}

		c.JSON(http.StatusOK, common.HttpResponse{Message: "friendly challenge rejected"})
	}
}

// WithdrawFriendlyChallengeHandler cancels one of the caller's own pending friendly challenges.
func WithdrawFriendlyChallengeHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := GetUserIdFromCookie(c)
		if userId == nil {
			c.JSON(http.StatusForbidden, nil)
			return
		}
		challengeId, err := common.StringToInt64(c.Param(ID))
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid challenge id"))
			return
		}
		team, err := common.GetTeamByOwner(*userId, dbCtx)
		if err != nil {
			c.JSON(http.StatusNotFound, GetError(0, "team not found for user"))
			return
		}

		if err := common.WithdrawFriendlyChallenge(dbCtx, challengeId, team.Id); err != nil {
			switch {
			case errors.Is(err, common.ErrFriendlyChallengeNotFound):
				c.JSON(http.StatusNotFound, GetError(0, err.Error()))
			case errors.Is(err, common.ErrNotChallengeSender):
				c.JSON(http.StatusForbidden, GetError(0, err.Error()))
			default:
				log.Errorf("error withdrawing friendly challenge %d: %s", challengeId, err)
				c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			}
			return
		}

		c.JSON(http.StatusOK, common.HttpResponse{Message: "friendly challenge withdrawn"})
	}
}

// UpcomingFriendliesHandler lists the caller's team's accepted-but-not-yet-played friendlies.
func UpcomingFriendliesHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := GetUserIdFromCookie(c)
		if userId == nil {
			c.JSON(http.StatusForbidden, nil)
			return
		}
		team, err := common.GetTeamByOwner(*userId, dbCtx)
		if err != nil {
			c.JSON(http.StatusNotFound, GetError(0, "team not found for user"))
			return
		}
		matches, err := common.GetUpcomingFriendlies(dbCtx, team.Id)
		if err != nil {
			log.Errorf("error getting upcoming friendlies for team %d: %s", team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		c.JSON(http.StatusOK, matches)
	}
}

const recentFriendliesLimit = 10

// RecentFriendliesHandler lists the caller's team's most recently played friendlies.
func RecentFriendliesHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := GetUserIdFromCookie(c)
		if userId == nil {
			c.JSON(http.StatusForbidden, nil)
			return
		}
		team, err := common.GetTeamByOwner(*userId, dbCtx)
		if err != nil {
			c.JSON(http.StatusNotFound, GetError(0, "team not found for user"))
			return
		}
		matches, err := common.GetRecentFriendlies(dbCtx, team.Id, recentFriendliesLimit)
		if err != nil {
			log.Errorf("error getting recent friendlies for team %d: %s", team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		c.JSON(http.StatusOK, matches)
	}
}

type setFriendlyScheduleRequest struct {
	WeekdaysMask int `json:"weekdaysMask"`
}

// AdminSetFriendlyScheduleHandler updates which weekdays a friendly may kick off on (see
// common.SetFriendlyScheduleWeekdays) - e.g. weekends only.
func AdminSetFriendlyScheduleHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req setFriendlyScheduleRequest
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "weekdaysMask is required"))
			return
		}

		if err := common.SetFriendlyScheduleWeekdays(dbCtx, req.WeekdaysMask); err != nil {
			if errors.Is(err, common.ErrInvalidFriendlySchedule) {
				c.JSON(http.StatusBadRequest, GetError(0, err.Error()))
				return
			}
			log.Errorf("error setting friendly schedule: %s", err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		c.JSON(http.StatusOK, common.HttpResponse{Message: "friendly schedule updated"})
	}
}

type setMaxFriendliesPerWeekRequest struct {
	MaxFriendliesPerWeek int `json:"maxFriendliesPerWeek"`
}

// AdminSetMaxFriendliesPerWeekHandler updates the admin-configured cap on friendly matches a
// team may accept per real calendar week (see common.SetMaxFriendliesPerWeek).
func AdminSetMaxFriendliesPerWeekHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req setMaxFriendliesPerWeekRequest
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "maxFriendliesPerWeek is required"))
			return
		}

		if err := common.SetMaxFriendliesPerWeek(dbCtx, req.MaxFriendliesPerWeek); err != nil {
			if errors.Is(err, common.ErrInvalidFriendlyLimit) {
				c.JSON(http.StatusBadRequest, GetError(0, err.Error()))
				return
			}
			log.Errorf("error setting max friendlies per week: %s", err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		c.JSON(http.StatusOK, common.HttpResponse{Message: "max friendlies per week updated"})
	}
}

// errFriendlyTeamNotFound distinguishes "caller has no team" from a real lookup failure below -
// see writeFriendlyLookupError.
var errFriendlyTeamNotFound = errors.New("team not found for user")

// loadFriendlyForParticipant fetches one friendly and confirms the caller's team is one of its
// two teams - shared by the lineup handlers, which need this check before letting the caller see
// or change either team's lineup for it (match details themselves are open to any viewer - see
// FriendlyMatchHandler).
func loadFriendlyForParticipant(dbCtx *common.DatabaseContext, userId int64, matchId int64) (*common.FriendlyMatch, *common.Team, error) {
	team, err := common.GetTeamByOwner(userId, dbCtx)
	if err != nil {
		return nil, nil, errFriendlyTeamNotFound
	}
	match, err := common.GetFriendlyMatchById(dbCtx, matchId)
	if err != nil {
		return nil, nil, err
	}
	if team.Id != match.HomeTeamId && team.Id != match.AwayTeamId {
		return nil, nil, common.ErrNotFriendlyParticipant
	}
	return match, team, nil
}

// writeFriendlyLookupError renders the errors loadFriendlyForParticipant can return - shared so
// the two lineup handlers that call it respond the same way to the same failures.
func writeFriendlyLookupError(c *gin.Context, matchId int64, err error) {
	switch {
	case errors.Is(err, errFriendlyTeamNotFound):
		c.JSON(http.StatusNotFound, GetError(0, err.Error()))
	case errors.Is(err, sql.ErrNoRows):
		c.JSON(http.StatusNotFound, GetError(0, "friendly not found"))
	case errors.Is(err, common.ErrNotFriendlyParticipant):
		c.JSON(http.StatusForbidden, GetError(0, err.Error()))
	default:
		log.Errorf("error loading friendly %d: %s", matchId, err)
		c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
	}
}

// FriendlyMatchHandler returns one friendly's details, open to any authenticated user the same
// way a league match's or cup tie's details are (see MatchDetailsHandler/CupTieDetailsHandler) -
// not just its two participants. A friendly can be the dashboard's random "live right now"
// suggestion (see common.GetLiveMatches) for any two teams in the world, so restricting this to
// participants - as it used to - made that link 403 for every viewer except the two teams
// actually playing it. Only the lineup endpoints below stay participant-only, since a lineup is
// each team's own private call.
func FriendlyMatchHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		matchId, err := common.StringToInt64(c.Param(ID))
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid match id"))
			return
		}
		match, err := common.GetFriendlyMatchById(dbCtx, matchId)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				c.JSON(http.StatusNotFound, GetError(0, "friendly not found"))
				return
			}
			log.Errorf("error getting friendly match %d: %s", matchId, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		if err := common.EnrichPlayedFriendly(dbCtx, match); err != nil {
			log.Errorf("error enriching played friendly %d: %s", matchId, err)
		}
		c.JSON(http.StatusOK, match)
	}
}

// GetFriendlyLineupHandler returns the caller's team's saved lineup override for one friendly
// (see common.GetFriendlyLineup), or its current default lineup if it hasn't saved one yet - the
// same starting point the editor would show either way.
func GetFriendlyLineupHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := GetUserIdFromCookie(c)
		if userId == nil {
			c.JSON(http.StatusForbidden, nil)
			return
		}
		matchId, err := common.StringToInt64(c.Param(ID))
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid match id"))
			return
		}
		_, team, err := loadFriendlyForParticipant(dbCtx, *userId, matchId)
		if err != nil {
			writeFriendlyLookupError(c, matchId, err)
			return
		}

		lineup, hasOverride, err := common.GetFriendlyLineup(dbCtx, matchId, team.Id)
		if err != nil {
			log.Errorf("error getting friendly lineup for match %d team %d: %s", matchId, team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		if hasOverride {
			c.JSON(http.StatusOK, lineup)
			return
		}

		// No override saved yet - hand back the team's current best available XI as the editor's
		// starting point (see common.BestLineupForTeam), same shape as an override so the
		// frontend doesn't need to know which one it got. Deliberately not the team's standing
		// default lineup (unlike the league/cup equivalents): a friendly should be fulfilled by
		// default with the strongest squad currently available, not whatever was last saved for
		// competitive play.
		bestLineup, err := common.BestLineupForTeam(dbCtx, team.Id)
		if err != nil {
			log.Errorf("error getting best lineup for team %d: %s", team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		bestLineup.MatchId = matchId
		c.JSON(http.StatusOK, bestLineup)
	}
}

// SetFriendlyLineupHandler saves the caller's team's lineup override for one friendly, along with
// any substitutions planned specifically for it, after the same formation/roster/availability and
// substitution validation SetLineupHandler applies to the default lineup - unlike the league/cup
// equivalents, a friendly gets its own scheduled substitutions (see common.SetFriendlyLineup),
// since its starting XI can itself differ from the team's standing default lineup.
func SetFriendlyLineupHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := GetUserIdFromCookie(c)
		if userId == nil {
			c.JSON(http.StatusForbidden, nil)
			return
		}
		matchId, err := common.StringToInt64(c.Param(ID))
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid match id"))
			return
		}
		var req SetLineupRequest
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid lineup payload"))
			return
		}

		match, team, err := loadFriendlyForParticipant(dbCtx, *userId, matchId)
		if err != nil {
			writeFriendlyLookupError(c, matchId, err)
			return
		}
		if match.Played {
			c.JSON(http.StatusConflict, GetError(0, common.ErrFriendlyAlreadyPlayed.Error()))
			return
		}
		if common.IsLineupLocked(match.MatchDate) {
			c.JSON(http.StatusConflict, GetError(0, common.ErrLineupLocked.Error()))
			return
		}

		players, err := common.GetPlayersFromTeamId(dbCtx, team.Id)
		if err != nil {
			log.Errorf("error getting players for team %d: %s", team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		unavailable, err := common.GetUnavailablePlayerIdsForFriendly(dbCtx, team.Id)
		if err != nil {
			log.Errorf("error getting unavailable players for team %d: %s", team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		if err := validateLineup(req.Formation, req.Positions, players, unavailable, true); err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, err.Error()))
			return
		}
		if err := validateSubstitutions(req.Positions, req.Substitutions, players, unavailable); err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, err.Error()))
			return
		}

		if err := common.SetFriendlyLineup(dbCtx, matchId, team.Id, req.Formation, req.Positions, req.Substitutions); err != nil {
			log.Errorf("error setting friendly lineup for match %d team %d: %s", matchId, team.Id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		c.JSON(http.StatusOK, common.HttpResponse{Message: "friendly lineup updated"})
	}
}
