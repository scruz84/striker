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

// AdvanceSeasonJob implements the quartz.Job interface.
type AdvanceSeasonJob struct {
	desc  string
	dbCtx *common.DatabaseContext
}

func (pj *AdvanceSeasonJob) Description() string {
	return pj.desc
}

// Execute is called by a Scheduler when the Trigger associated with this job fires.
func (pj *AdvanceSeasonJob) Execute(context.Context) error {
	log.Info("Checking season completion job")
	AdvanceSeasonIfComplete(pj.dbCtx)
	return nil
}

func NewAdvanceSeasonJob(dbCtx *common.DatabaseContext) *AdvanceSeasonJob {
	return &AdvanceSeasonJob{desc: "season rollover job", dbCtx: dbCtx}
}

// CupJob implements the quartz.Job interface: each tick, it simulates any due cup ties and
// advances any cup whose current round has fully resolved.
type CupJob struct {
	desc  string
	dbCtx *common.DatabaseContext
}

func (pj *CupJob) Description() string {
	return pj.desc
}

// Execute is called by a Scheduler when the Trigger associated with this job fires.
func (pj *CupJob) Execute(context.Context) error {
	log.Info("Cup progression job")
	SimulateCupTies(pj.dbCtx)
	AdvanceCups(pj.dbCtx)
	return nil
}

func NewCupJob(dbCtx *common.DatabaseContext) *CupJob {
	return &CupJob{desc: "cup progression job", dbCtx: dbCtx}
}
