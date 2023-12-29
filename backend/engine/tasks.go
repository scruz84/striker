package engine

import (
	"context"
	"github.com/reugn/go-quartz/quartz"
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

// Key returns the unique PrintJob key.
func (pj *SimulateMatchesJob) Key() int {
	return quartz.HashCode(pj.Description())
}

// Execute is called by a Scheduler when the Trigger associated with this job fires.
func (pj *SimulateMatchesJob) Execute(_ context.Context) {
	log.Info("Simulating matches job")
	SimulateMatches(pj.dbCtx)
}

func NewSimulateMatchesJob(dbCtx *common.DatabaseContext) *SimulateMatchesJob {
	return &SimulateMatchesJob{desc: "matches simulation job", dbCtx: dbCtx}
}
