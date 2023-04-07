package engine

import (
	"encoding/json"
	"footmanager/fmcommon"
	log "github.com/sirupsen/logrus"
	"strconv"
)

const ClassificationKey = "classDet_key_"
const FullTeamDetailsKey = "fullTeamDet_key_"
const PlayersFromTeamKey = "playTeam_key_"
const TeamDetailsKey = "teamDet_key_"
const TeamOwnerKey = "teamOwner_key_"

func GetPlayersFromTeam(dbCtx *fmcommon.DatabaseContext, teamId int64) (*[]*fmcommon.Player, error) {
	key := PlayersFromTeamKey + strconv.FormatInt(teamId, 10)
	v, _ := dbCtx.ObjectCache.GetValue(key)
	if v != "" {
		var players *[]*fmcommon.Player
		err := json.Unmarshal([]byte(v), &players)
		if err == nil {
			return players, nil
		}
	}
	teamDetails, _ := fmcommon.GetPlayersFromTeamId(dbCtx, teamId)
	value, _ := json.Marshal(teamDetails)
	dbCtx.ObjectCache.SetValue(key, string(value))
	return teamDetails, nil
}

func GetDivisionDetails(dbCtx *fmcommon.DatabaseContext, leagueId int64) (*fmcommon.DivisionDetails, error) {
	key := ClassificationKey + strconv.FormatInt(leagueId, 10)
	v, _ := dbCtx.ObjectCache.GetValue(key)
	var divisionDetails *fmcommon.DivisionDetails
	if v != "" {
		err := json.Unmarshal([]byte(v), &divisionDetails)
		if err == nil {
			return divisionDetails, nil
		}
	}
	divisionDetails, err := fmcommon.GetDivisionDetails(dbCtx, leagueId)
	if err != nil {
		log.Errorf("error retrieving division details from: %d. %s", leagueId, err)
		return nil, err
	}
	value, _ := json.Marshal(divisionDetails)
	dbCtx.ObjectCache.SetValue(key, string(value))
	return divisionDetails, nil
}

func GetTeamDetails(dbCtx *fmcommon.DatabaseContext, owner bool, teamId int64) (*fmcommon.TeamDetails, error) {
	var key string
	if owner {
		key = FullTeamDetailsKey + strconv.FormatInt(teamId, 10)
	} else {
		key = TeamDetailsKey + strconv.FormatInt(teamId, 10)
	}
	v, _ := dbCtx.ObjectCache.GetValue(key)
	var teamDetails *fmcommon.TeamDetails
	if v != "" {
		err := json.Unmarshal([]byte(v), &teamDetails)
		if err == nil {
			return teamDetails, nil
		}
	}
	var err error
	teamDetails, err = fmcommon.GetTeamDetailsById(teamId, owner, dbCtx)
	if err != nil {
		return nil, err
	}
	value, _ := json.Marshal(teamDetails)
	dbCtx.ObjectCache.SetValue(key, string(value))
	return teamDetails, nil
}

func GetUserOfTeamId(dbCtx *fmcommon.DatabaseContext, teamId int64) (int64, error) {
	key := TeamOwnerKey + strconv.FormatInt(teamId, 10)
	v, _ := dbCtx.ObjectCache.GetValue(key)
	if v != "" {
		return fmcommon.StringToInt64(v)
	}
	userId, err := fmcommon.GetTeamOwner(dbCtx, teamId)
	if err != nil {
		return -1, err
	}
	dbCtx.ObjectCache.SetValueLongDuration(key, strconv.FormatInt(userId, 10))
	return userId, nil
}
