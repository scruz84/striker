package engine

import (
	"errors"
	"fmt"
	"runtime/debug"

	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
)

// runClaimedSimulation runs simulate for a fixture this replica has just claimed, and turns any
// failure into a recorded one instead of a claimed row nobody will ever release: an error or a
// panic counts as a failed attempt (the claim already counted it), the claim is released and the
// reason stored (see common.RecordSimulationFailure), so the fixture is retried after the backoff
// and, once it has used all its attempts, settled 0-0 (see SettleDefaultResults). Losing the race
// to another simulation of the same fixture (common.ErrAlreadyPlayed) is not a failure.
func runClaimedSimulation(dbCtx *common.DatabaseContext, kind string, id int64, simulate func() error) {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("panic simulating %s %d: %v\n%s", kind, id, r, debug.Stack())
			recordSimulationFailure(dbCtx, kind, id, fmt.Errorf("panic: %v", r))
		}
	}()
	if err := simulate(); err != nil {
		if errors.Is(err, common.ErrAlreadyPlayed) {
			log.Infof("%s %d was already played by another simulation, skipping", kind, id)
			return
		}
		log.Errorf("error simulating %s %d: %s", kind, id, err)
		recordSimulationFailure(dbCtx, kind, id, err)
	}
}

func recordSimulationFailure(dbCtx *common.DatabaseContext, kind string, id int64, cause error) {
	if err := common.RecordSimulationFailure(dbCtx, kind, id, cause); err != nil {
		log.Errorf("error recording the simulation failure of %s %d: %s", kind, id, err)
	}
}
