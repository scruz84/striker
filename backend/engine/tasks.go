package engine

import (
	"context"
	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
)

// SimulateMatchesJob implements the quartz.Job interface.
type SimulateMatchesJob struct {
	desc  string
	dbCtx *common.DatabaseContext
}

func (pj *SimulateMatchesJob) Description() string {
	return pj.desc
}

// Execute is called by a Scheduler when the Trigger associated with this job fires.
func (pj *SimulateMatchesJob) Execute(context.Context) error {
	log.Info("Simulating matches job")
	SimulateMatches(pj.dbCtx)
	return nil
}

func NewSimulateMatchesJob(dbCtx *common.DatabaseContext) *SimulateMatchesJob {
	return &SimulateMatchesJob{desc: "matches simulation job", dbCtx: dbCtx}
}
