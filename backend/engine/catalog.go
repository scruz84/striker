package engine

import (
	"encoding/json"
	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
	"strconv"
)

const ClassificationKey = "classDet_key"
const CurrentSeasonKey = "currentSeason_key_"
const FullTeamDetailsKey = "fullTeamDet_key_"
const MatchDetailsKey = "matchDet_key_"
const PlayersFromTeamKey = "playTeam_key_"
const PlayerNameKey = "playerName_key_"
const TeamDetailsKey = "teamDet_key_"
const TeamOwnerKey = "teamOwner_key_"

// InvalidateTeamDetailsCache drops both the owner and non-owner cached views of a team (see
// GetTeamDetails) so a direct DB write - e.g. an admin override - is reflected on the next read
// instead of waiting out the cache's TTL.
func InvalidateTeamDetailsCache(dbCtx *common.DatabaseContext, teamId int64) {
	idStr := strconv.FormatInt(teamId, 10)
	dbCtx.ObjectCache.DeleteValue(FullTeamDetailsKey + idStr)
	dbCtx.ObjectCache.DeleteValue(TeamDetailsKey + idStr)
}

func GetCurrentSeason(dbCtx *common.DatabaseContext) (int, error) {
	key := CurrentSeasonKey
	v, _ := dbCtx.ObjectCache.GetValue(key)
	if v != "" {
		vint, err := strconv.Atoi(v)
		if err == nil {
			return vint, nil
		}
	}
	vint := common.GetCurrentSeason(dbCtx)
	dbCtx.ObjectCache.SetValueLongDuration(key, strconv.Itoa(vint))
	return vint, nil
}

// ResetCurrentSeasonCache overwrites the cached "current season" (see GetCurrentSeason above)
// with the value a caller just wrote straight to the database - the same "refresh the cache
// immediately after a direct write" pattern AdvanceSeasonIfComplete already uses after a normal
// season rollover. handlers.InitializeGame and handlers.RecreateLeaguesKeepingUsers both reset
// the season to 1 directly in Postgres; without this, a season number cached from a previous run
// (most plausible with an external Redis instance kept across a database reset - see issue #8)
// would silently outlive the reset for up to the cache's own TTL. Worse than a transient
// mismatch: GenerateMatches (called once, synchronously, right after) would tag the freshly
// generated fixtures with that stale season number, while every other read gets the correct one
// straight from the database - and nothing ever revisits "zero matches for the season the
// database says it is" as a bug needing recovery, so the fixtures stay permanently orphaned
// rather than the mismatch resolving itself once the cache entry expires.
func ResetCurrentSeasonCache(dbCtx *common.DatabaseContext, season int) {
	dbCtx.ObjectCache.SetValueLongDuration(CurrentSeasonKey, strconv.Itoa(season))
}

func GetPlayersFromTeam(dbCtx *common.DatabaseContext, teamId int64) (*[]*common.Player, error) {
	key := PlayersFromTeamKey + strconv.FormatInt(teamId, 10)
	v, _ := dbCtx.ObjectCache.GetValue(key)
	if v != "" {
		var players *[]*common.Player
		err := json.Unmarshal([]byte(v), &players)
		if err == nil {
			return players, nil
		}
	}
	teamDetails, _ := common.GetPlayersFromTeamId(dbCtx, teamId)
	value, _ := json.Marshal(teamDetails)
	dbCtx.ObjectCache.SetValue(key, string(value))
	return teamDetails, nil
}

func GetPlayerFullName(dbCtx *common.DatabaseContext, playerId int64) (string, error) {
	key := PlayerNameKey + strconv.FormatInt(playerId, 10)
	v, _ := dbCtx.ObjectCache.GetValue(key)
	if v != "" {
		return v, nil
	}
	name, e := common.GetPlayerFullName(dbCtx, playerId)
	if e != nil {
		return "", e
	}
	dbCtx.ObjectCache.SetValue(key, name)
	return name, nil
}

func GetDivisionDetails(dbCtx *common.DatabaseContext, leagueId int64) (*common.DivisionDetails, error) {
	key := ClassificationKey + strconv.FormatInt(leagueId, 10)
	v, _ := dbCtx.ObjectCache.GetValue(key)
	var divisionDetails *common.DivisionDetails
	if v != "" {
		err := json.Unmarshal([]byte(v), &divisionDetails)
		if err == nil {
			return divisionDetails, nil
		}
	}
	divisionDetails, err := common.GetDivisionDetails(dbCtx, leagueId)
	if err != nil {
		log.Errorf("error retrieving division details from: %d. %s", leagueId, err)
		return nil, err
	}
	value, _ := json.Marshal(divisionDetails)
	dbCtx.ObjectCache.SetValue(key, string(value))
	return divisionDetails, nil
}

func GetTeamDetails(dbCtx *common.DatabaseContext, owner bool, teamId int64) (*common.TeamDetails, error) {
	var key string
	if owner {
		key = FullTeamDetailsKey + strconv.FormatInt(teamId, 10)
	} else {
		key = TeamDetailsKey + strconv.FormatInt(teamId, 10)
	}
	v, _ := dbCtx.ObjectCache.GetValue(key)
	var teamDetails *common.TeamDetails
	if v != "" {
		err := json.Unmarshal([]byte(v), &teamDetails)
		if err == nil {
			return teamDetails, nil
		}
	}
	var err error
	teamDetails, err = common.GetTeamDetailsById(teamId, owner, dbCtx)
	if err != nil {
		return nil, err
	}
	value, _ := json.Marshal(teamDetails)
	dbCtx.ObjectCache.SetValue(key, string(value))
	return teamDetails, nil
}

func GetUserOfTeamId(dbCtx *common.DatabaseContext, teamId int64) (int64, error) {
	key := TeamOwnerKey + strconv.FormatInt(teamId, 10)
	v, _ := dbCtx.ObjectCache.GetValue(key)
	if v != "" {
		return common.StringToInt64(v)
	}
	userId, err := common.GetTeamOwner(dbCtx, teamId)
	if err != nil {
		return -1, err
	}
	dbCtx.ObjectCache.SetValueLongDuration(key, strconv.FormatInt(userId, 10))
	return userId, nil
}

func GetMatchDetails(dbCtx *common.DatabaseContext, matchId int64) (*common.WeekMatch, error) {
	key := MatchDetailsKey + strconv.FormatInt(matchId, 10)
	v, _ := dbCtx.ObjectCache.GetValue(key)
	var matchDetails *common.WeekMatch
	if v != "" {
		err := json.Unmarshal([]byte(v), &matchDetails)
		if err == nil {
			return matchDetails, nil
		}
	}
	var err error
	matchDetails, err = common.WeekMatchById(dbCtx, matchId)
	if err != nil {
		return nil, err
	}
	if matchDetails == nil {
		return nil, nil
	}
	value, _ := json.Marshal(matchDetails)
	dbCtx.ObjectCache.SetValue(key, string(value))
	return matchDetails, nil
}
