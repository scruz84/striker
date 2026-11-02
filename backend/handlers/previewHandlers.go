package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/scruz84/striker/common"
	"github.com/scruz84/striker/engine"
	log "github.com/sirupsen/logrus"
)

// KindParam is the :kind path parameter of the opponent preview route.
const KindParam = "kind"

// OpponentPreviewResponse is the opponent preview of one upcoming fixture: the opponent's public
// form, likely XI and in-form players, plus the projected zone duel of the caller's saved lineup
// against that XI (nil when the opponent has no history to project against).
type OpponentPreviewResponse struct {
	*common.OpponentPreview
	Projection *engine.ZoneDuelProjection `json:"projection"`
}

// previewFixture is the fixture a preview is requested for, resolved for the calling owner.
type previewFixture struct {
	opponentId int64
	matchDate  time.Time
	played     bool
	friendly   bool
	// homeAdvantage is the multiplier the home side's zone ratings will get at this fixture's
	// ground, from the expected crowd (see engine.ExpectedHomeAdvantage), and ownIsHome whether
	// the owner's club is that home side.
	homeAdvantage float32
	ownIsHome     bool
	// ownLineup is the owner's lineup for this fixture as the lineup editor shows it: his saved
	// override, else his standing default (or, for a friendly, his friendly default or best XI).
	ownLineup func() (*common.Lineup, error)
}

// OpponentPreviewHandler returns the opponent preview of an upcoming league match, cup tie or
// friendly (kind = league, cup or friendly) to an owner whose club plays it: 403 for anyone else,
// 409 once the lineup lock has started (or the fixture is played), when the preview is no longer
// useful. Only public data is used for the opponent (see common.GetOpponentPreview).
func OpponentPreviewHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := GetUserIdFromCookie(c)
		if userId == nil {
			c.JSON(http.StatusForbidden, nil)
			return
		}
		id, err := common.StringToInt64(c.Param(ID))
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid fixture id"))
			return
		}
		kind := c.Param(KindParam)
		team, fixture, err := loadPreviewFixture(dbCtx, *userId, kind, id)
		if err != nil {
			writePreviewLookupError(c, kind, id, err)
			return
		}
		if fixture.played || common.IsLineupLocked(fixture.matchDate) {
			c.JSON(http.StatusConflict, GetError(0, "the lineup is locked: no preview for this fixture any more"))
			return
		}

		preview, err := common.GetOpponentPreview(dbCtx, fixture.opponentId)
		if err != nil {
			log.Errorf("error building the opponent preview for %s %d: %s", kind, id, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		response := OpponentPreviewResponse{OpponentPreview: preview}
		if preview.Likely != nil {
			response.Projection, err = projectOpponentDuel(dbCtx, team.Id, fixture, preview.Likely)
			if err != nil {
				log.Errorf("error projecting the zone duel for %s %d: %s", kind, id, err)
				c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
				return
			}
		}
		c.JSON(http.StatusOK, response)
	}
}

// projectOpponentDuel rates the owner's lineup for the fixture - after the pre-kickoff refill the
// editor already projects - against the opponent's likely XI with engine.ProjectZoneDuel.
func projectOpponentDuel(dbCtx *common.DatabaseContext, teamId int64, fixture *previewFixture, likely *common.LikelyLineup) (*engine.ZoneDuelProjection, error) {
	ownLineup, err := fixture.ownLineup()
	if err != nil {
		return nil, err
	}
	ownSquad, err := common.GetPlayersFromTeamId(dbCtx, teamId)
	if err != nil {
		return nil, err
	}
	var unavailable map[int64]bool
	if fixture.friendly {
		unavailable, err = common.GetUnavailablePlayerIdsForFriendly(dbCtx, teamId)
	} else {
		unavailable, err = common.GetUnavailablePlayerIds(dbCtx, teamId)
	}
	if err != nil {
		return nil, err
	}
	ownLineup = common.ApplyProjectedReplacements(ownLineup, common.ProjectKickoffReplacements(ownLineup, *ownSquad, unavailable))

	opponentSquad, err := common.GetPlayersFromTeamId(dbCtx, fixture.opponentId)
	if err != nil {
		return nil, err
	}
	opponentLineup := &common.Lineup{TeamId: fixture.opponentId, Formation: likely.Formation, Positions: make([]common.LineupPosition, 0, len(likely.Players))}
	for _, p := range likely.Players {
		opponentLineup.Positions = append(opponentLineup.Positions, common.LineupPosition{PlayerId: p.PlayerId, Position: p.Position})
	}

	projection := engine.ProjectZoneDuel(
		engine.ZoneDuelSide{Lineup: ownLineup, Squad: *ownSquad},
		engine.ZoneDuelSide{Lineup: opponentLineup, Squad: *opponentSquad},
		fixture.homeAdvantage, fixture.ownIsHome,
	)
	return &projection, nil
}

var errPreviewBadKind = errors.New("unknown fixture kind")

// loadPreviewFixture resolves a fixture of the given kind for the calling owner, reusing the
// participant checks of the lineup handlers.
func loadPreviewFixture(dbCtx *common.DatabaseContext, userId int64, kind string, id int64) (*common.Team, *previewFixture, error) {
	switch kind {
	case common.MatchKindLeague:
		match, team, err := loadMatchForParticipant(dbCtx, userId, id)
		if err != nil {
			return nil, nil, err
		}
		return team, &previewFixture{
			opponentId:    otherTeam(team.Id, match.Home, match.Away),
			matchDate:     match.MatchDate,
			played:        match.Played,
			homeAdvantage: engine.ExpectedHomeAdvantage(dbCtx, match.Home, match.Away, 0, false),
			ownIsHome:     team.Id == match.Home,
			ownLineup: func() (*common.Lineup, error) {
				return ownCompetitiveLineup(dbCtx, team.Id, id, common.GetMatchLineupOverride)
			},
		}, nil
	case common.MatchKindCup:
		tie, team, err := loadCupTieForParticipant(dbCtx, userId, id)
		if err != nil {
			return nil, nil, err
		}
		worldCup, err := common.IsWorldCupKnockoutCup(dbCtx, tie.CupId)
		if err != nil {
			return nil, nil, err
		}
		fixture := &previewFixture{
			opponentId:    otherTeam(team.Id, *tie.HomeTeamId, *tie.AwayTeamId),
			played:        tie.Played,
			homeAdvantage: engine.ExpectedHomeAdvantage(dbCtx, *tie.HomeTeamId, *tie.AwayTeamId, tie.Round, worldCup),
			ownIsHome:     team.Id == *tie.HomeTeamId,
			ownLineup: func() (*common.Lineup, error) {
				return ownCompetitiveLineup(dbCtx, team.Id, id, common.GetCupTieLineup)
			},
		}
		if tie.MatchDate != nil {
			fixture.matchDate = *tie.MatchDate
		}
		return team, fixture, nil
	case common.MatchKindFriendly:
		match, team, err := loadFriendlyForParticipant(dbCtx, userId, id)
		if err != nil {
			return nil, nil, err
		}
		worldCup, err := common.IsWorldCupGroupMatch(dbCtx, id)
		if err != nil {
			return nil, nil, err
		}
		return team, &previewFixture{
			opponentId:    otherTeam(team.Id, match.HomeTeamId, match.AwayTeamId),
			matchDate:     match.MatchDate,
			played:        match.Played,
			friendly:      true,
			homeAdvantage: engine.ExpectedHomeAdvantage(dbCtx, match.HomeTeamId, match.AwayTeamId, 0, worldCup),
			ownIsHome:     team.Id == match.HomeTeamId,
			ownLineup:     func() (*common.Lineup, error) { return ownFriendlyLineup(dbCtx, team.Id, id) },
		}, nil
	}
	return nil, nil, errPreviewBadKind
}

func otherTeam(own, home, away int64) int64 {
	if own == home {
		return away
	}
	return home
}

// ownCompetitiveLineup is the owner's lineup for a league match or cup tie as the lineup editor
// shows it: his saved override (read with load), else his standing default lineup and formation.
func ownCompetitiveLineup(dbCtx *common.DatabaseContext, teamId int64, fixtureId int64,
	load func(*common.DatabaseContext, int64, int64) (common.Lineup, bool, error)) (*common.Lineup, error) {
	lineup, hasOverride, err := load(dbCtx, fixtureId, teamId)
	if err != nil {
		return nil, err
	}
	if hasOverride {
		return &lineup, nil
	}
	formation, err := common.GetTeamFormation(dbCtx, teamId)
	if err != nil {
		return nil, err
	}
	defaultLineup, err := common.GetDefaultLineup(dbCtx, teamId)
	if err != nil {
		return nil, err
	}
	defaultLineup.MatchId = fixtureId
	defaultLineup.Formation = formation
	return withBestLineupIfEmpty(dbCtx, teamId, defaultLineup)
}

// ownFriendlyLineup is the owner's lineup for a friendly as the editor shows it: his saved
// override, else his standing friendly lineup, else his best available XI.
func ownFriendlyLineup(dbCtx *common.DatabaseContext, teamId int64, matchId int64) (*common.Lineup, error) {
	lineup, hasOverride, err := common.GetFriendlyLineup(dbCtx, matchId, teamId)
	if err != nil {
		return nil, err
	}
	if hasOverride {
		return &lineup, nil
	}
	friendlyDefault, err := common.GetDefaultFriendlyLineup(dbCtx, teamId)
	if err != nil {
		return nil, err
	}
	friendlyDefault.MatchId = matchId
	return withBestLineupIfEmpty(dbCtx, teamId, friendlyDefault)
}

// withBestLineupIfEmpty falls back to the club's best available XI when it has saved no lineup at
// all, the same starting point the lineup editor offers.
func withBestLineupIfEmpty(dbCtx *common.DatabaseContext, teamId int64, lineup *common.Lineup) (*common.Lineup, error) {
	if len(lineup.Positions) > 0 {
		return lineup, nil
	}
	return common.BestLineupForTeam(dbCtx, teamId)
}

// writePreviewLookupError renders the errors loadPreviewFixture can return.
func writePreviewLookupError(c *gin.Context, kind string, id int64, err error) {
	switch {
	case errors.Is(err, errPreviewBadKind):
		c.JSON(http.StatusBadRequest, GetError(0, err.Error()))
	case errors.Is(err, errMatchTeamNotFound), errors.Is(err, errCupTieTeamNotFound), errors.Is(err, errFriendlyTeamNotFound):
		c.JSON(http.StatusNotFound, GetError(0, "team not found for user"))
	case errors.Is(err, sql.ErrNoRows):
		c.JSON(http.StatusNotFound, GetError(0, "fixture not found"))
	case errors.Is(err, ErrNotMatchParticipant), errors.Is(err, ErrNotCupTieParticipant), errors.Is(err, common.ErrNotFriendlyParticipant):
		c.JSON(http.StatusForbidden, GetError(0, "your team is not part of this fixture"))
	default:
		log.Errorf("error loading %s fixture %d for the opponent preview: %s", kind, id, err)
		c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
	}
}
