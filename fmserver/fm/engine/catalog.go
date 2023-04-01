package engine

import (
	"encoding/json"
	"footmanager/fmcommon"
	log "github.com/sirupsen/logrus"
	"strconv"
)

const TeamKey = "teamDet_key_"
const ClassificationKey = "classDet_key_"

func GetPlayersFromTeam(dbCtx *fmcommon.DatabaseContext, teamId int64) (*[]*fmcommon.Player, error) {
	key := TeamKey + strconv.FormatInt(teamId, 10)
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
