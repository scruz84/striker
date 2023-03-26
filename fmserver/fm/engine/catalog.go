package engine

import (
	"encoding/json"
	"footmanager/fmcommon"
	log "github.com/sirupsen/logrus"
	"strconv"
)

const TeamKey = "teamDet_key_"

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
	log.Debugf("%s", v)
	teamDetails, _ := fmcommon.GetPlayersFromTeamId(dbCtx, teamId)
	value, _ := json.Marshal(teamDetails)
	dbCtx.ObjectCache.SetValue(key, string(value))
	return teamDetails, nil
}
