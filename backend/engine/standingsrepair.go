package engine

import (
	"time"

	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
)

// standingsRepairSlot names the job slot that makes the repair run once per deployment of the
// change that introduced it. Bump the suffix (standings_repair_v2) to run it again.
const standingsRepairSlot = "standings_repair_v1"

// RepairStandingsOnce rebuilds every classification row whose matches count differs from the
// team's played league matches that season (see common.CountStandingsMismatches) - the drift left
// by a result that an earlier failed save booked twice. It runs at engine start, guarded by a
// fixed job slot only one replica ever wins, so several replicas starting together run it once;
// a row that already agrees with its matches is never touched. Returns how many rows it rebuilt.
func RepairStandingsOnce(dbCtx *common.DatabaseContext) int64 {
	if !common.ClaimEngineJobSlot(dbCtx, standingsRepairSlot, time.Unix(0, 0), 24*time.Hour) {
		return 0
	}
	return repairStandings(dbCtx)
}

// repairStandings rebuilds every mismatched classification row, league by league, each under that
// league's standings lock. Returns how many rows it rebuilt.
func repairStandings(dbCtx *common.DatabaseContext) int64 {
	leagueIds, err := common.GetLeaguesWithStandingsMismatch(dbCtx)
	if err != nil {
		log.Errorf("error finding leagues with mismatched standings: %s", err)
		return 0
	}
	var rebuilt int64
	for _, leagueId := range leagueIds {
		n, err := common.RebuildLeagueStandings(dbCtx, leagueId)
		if err != nil {
			log.Errorf("error rebuilding the standings of league %d: %s", leagueId, err)
			continue
		}
		rebuilt += n
	}
	if rebuilt > 0 {
		log.Infof("Standings repair rebuilt %d classification rows in %d leagues", rebuilt, len(leagueIds))
	}
	return rebuilt
}
