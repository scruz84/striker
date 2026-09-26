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

// FreeAgentInjectionJob implements the quartz.Job interface: adds the admin-configured count of
// brand-new players to the free agent pool, on a real-calendar-time schedule independent of
// match simulation - see common.InjectFreeAgents/common.GetFreeAgentInjectionsPerWeek.
type FreeAgentInjectionJob struct {
	desc  string
	dbCtx *common.DatabaseContext
}

func (pj *FreeAgentInjectionJob) Description() string {
	return pj.desc
}

// Execute is called by a Scheduler when the Trigger associated with this job fires.
func (pj *FreeAgentInjectionJob) Execute(context.Context) error {
	count, err := common.GetFreeAgentInjectionsPerWeek(pj.dbCtx)
	if err != nil {
		log.Errorf("error reading free agent injections per week: %s", err)
		return nil
	}
	log.Infof("Free agent injection job (%d players)", count)
	if err := common.InjectFreeAgents(pj.dbCtx, count); err != nil {
		log.Errorf("error injecting free agents: %s", err)
	}
	return nil
}

func NewFreeAgentInjectionJob(dbCtx *common.DatabaseContext) *FreeAgentInjectionJob {
	return &FreeAgentInjectionJob{desc: "free agent injection job", dbCtx: dbCtx}
}

// FriendlyJob implements the quartz.Job interface: each tick, it simulates any friendly matches
// whose kickoff has arrived.
type FriendlyJob struct {
	desc  string
	dbCtx *common.DatabaseContext
}

func (pj *FriendlyJob) Description() string {
	return pj.desc
}

// Execute is called by a Scheduler when the Trigger associated with this job fires.
func (pj *FriendlyJob) Execute(context.Context) error {
	log.Info("Friendly matches job")
	SimulateFriendlies(pj.dbCtx)
	return nil
}

func NewFriendlyJob(dbCtx *common.DatabaseContext) *FriendlyJob {
	return &FriendlyJob{desc: "friendly matches job", dbCtx: dbCtx}
}

// OfferResponseJob implements the quartz.Job interface: sweeps every unowned team's pending
// incoming transfer offers, on a real-calendar-time schedule independent of match simulation -
// see engine.RespondToAllPendingOffers.
type OfferResponseJob struct {
	desc  string
	dbCtx *common.DatabaseContext
}

func (pj *OfferResponseJob) Description() string {
	return pj.desc
}

// Execute is called by a Scheduler when the Trigger associated with this job fires.
func (pj *OfferResponseJob) Execute(context.Context) error {
	log.Info("Offer response sweep job")
	RespondToAllPendingOffers(pj.dbCtx)
	return nil
}

func NewOfferResponseJob(dbCtx *common.DatabaseContext) *OfferResponseJob {
	return &OfferResponseJob{desc: "offer response sweep job", dbCtx: dbCtx}
}

// BirthdayJob implements the quartz.Job interface: ages up every player whose periodic in-game
// birthday has come around - see engine.ProcessBirthdays.
type BirthdayJob struct {
	desc  string
	dbCtx *common.DatabaseContext
}

func (pj *BirthdayJob) Description() string {
	return pj.desc
}

// Execute is called by a Scheduler when the Trigger associated with this job fires.
func (pj *BirthdayJob) Execute(context.Context) error {
	log.Info("Player birthdays job")
	ProcessBirthdays(pj.dbCtx)
	return nil
}

func NewBirthdayJob(dbCtx *common.DatabaseContext) *BirthdayJob {
	return &BirthdayJob{desc: "player birthdays job", dbCtx: dbCtx}
}

// AiActivityJob implements the quartz.Job interface: gives every AI-managed team its
// independent, randomly-timed daily chance at market management and at challenging another
// AI-managed team to a friendly - see engine.RunAiTeamDailyActivity.
type AiActivityJob struct {
	desc  string
	dbCtx *common.DatabaseContext
}

func (pj *AiActivityJob) Description() string {
	return pj.desc
}

// Execute is called by a Scheduler when the Trigger associated with this job fires.
func (pj *AiActivityJob) Execute(context.Context) error {
	log.Info("AI team daily activity job")
	RunAiTeamDailyActivity(pj.dbCtx)
	return nil
}

func NewAiActivityJob(dbCtx *common.DatabaseContext) *AiActivityJob {
	return &AiActivityJob{desc: "AI team daily activity job", dbCtx: dbCtx}
}
