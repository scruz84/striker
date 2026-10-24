package engine

import (
	"context"
	"time"

	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
)

// claimRun reports whether this replica should run a "once per period" job this tick. Every
// --engine replica fires the same cron trigger at the same instant, so without a claim a weekly
// job would inject its free agents, or a 5-minute job roll its dice, once per replica - see
// common.ClaimEngineJobSlot. period must be no longer than the job's cron interval.
func claimRun(dbCtx *common.DatabaseContext, jobName string, period time.Duration) bool {
	return common.ClaimEngineJobSlot(dbCtx, jobName, time.Now(), period)
}

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
	AdvanceSeasonsIfComplete(pj.dbCtx)
	return nil
}

func NewAdvanceSeasonJob(dbCtx *common.DatabaseContext) *AdvanceSeasonJob {
	return &AdvanceSeasonJob{desc: "season rollover job", dbCtx: dbCtx}
}

// FreeAgentProgressionJob implements the quartz.Job interface: ages the unowned free-agent pool
// (see ProgressFreeAgents) on its own real-calendar-time schedule - split out from the season
// rollover (see AdvanceSeasonJob) because a free agent belongs to no team/league/country, so
// there's no single country's season boundary left to hang it on now that seasons advance per
// country (see init75_season_per_country.sql).
type FreeAgentProgressionJob struct {
	desc  string
	dbCtx *common.DatabaseContext
}

func (pj *FreeAgentProgressionJob) Description() string {
	return pj.desc
}

func (pj *FreeAgentProgressionJob) Execute(context.Context) error {
	if !claimRun(pj.dbCtx, "free_agent_progression", 24*time.Hour) {
		return nil
	}
	log.Info("Free agent progression job")
	ProgressFreeAgents(pj.dbCtx)
	return nil
}

func NewFreeAgentProgressionJob(dbCtx *common.DatabaseContext) *FreeAgentProgressionJob {
	return &FreeAgentProgressionJob{desc: "free agent progression job", dbCtx: dbCtx}
}

// InternationalCupJob implements the quartz.Job interface: starts a new regional/international
// cup round, and (every WorldCupFrequencySeasons rounds) a new World Cup - see
// StartInternationalCupsIfDue, which gates the actual cadence internally
// (InternationalCupCadence) so this can tick far more often than it actually fires.
type InternationalCupJob struct {
	desc  string
	dbCtx *common.DatabaseContext
}

func (pj *InternationalCupJob) Description() string {
	return pj.desc
}

func (pj *InternationalCupJob) Execute(context.Context) error {
	StartInternationalCupsIfDue(pj.dbCtx)
	return nil
}

func NewInternationalCupJob(dbCtx *common.DatabaseContext) *InternationalCupJob {
	return &InternationalCupJob{desc: "international cup job", dbCtx: dbCtx}
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
	// A World Cup's knockout stage is itself an ordinary cup, advanced by the call above; this
	// additionally moves a World Cup from its group stage into that knockout stage once every
	// group match is played, and marks one completed once its knockout cup is - see
	// AdvanceWorldCups.
	AdvanceWorldCups(pj.dbCtx)
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
	if !claimRun(pj.dbCtx, "free_agent_injection", 24*time.Hour) {
		return nil
	}
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

// NationalTeamFriendlyJob implements the quartz.Job interface: once a week, schedules a fresh
// round of national-team friendlies (see common.ScheduleNationalTeamFriendlies) - the fixtures it
// creates are then picked up and simulated by the ordinary FriendlyJob above like any other
// friendly, so this job's only responsibility is creating that week's pairings and lineups.
type NationalTeamFriendlyJob struct {
	desc  string
	dbCtx *common.DatabaseContext
}

func (nj *NationalTeamFriendlyJob) Description() string {
	return nj.desc
}

func (nj *NationalTeamFriendlyJob) Execute(context.Context) error {
	log.Info("National team friendlies job")
	if err := common.ScheduleNationalTeamFriendlies(nj.dbCtx); err != nil {
		log.Errorf("error scheduling national team friendlies: %s", err)
	}
	return nil
}

func NewNationalTeamFriendlyJob(dbCtx *common.DatabaseContext) *NationalTeamFriendlyJob {
	return &NationalTeamFriendlyJob{desc: "national team friendlies job", dbCtx: dbCtx}
}

// NationalTeamCallUpJob implements the quartz.Job interface: closes the call-up (starting XI
// selection) for every national-team friendly within a day of kickoff that doesn't have one yet
// (see common.CallUpNationalTeamSquads) - split out from NationalTeamFriendlyJob above so a
// fixture can be created (and shown in the match viewer) well before its squad is actually picked.
type NationalTeamCallUpJob struct {
	desc  string
	dbCtx *common.DatabaseContext
}

func (nj *NationalTeamCallUpJob) Description() string {
	return nj.desc
}

func (nj *NationalTeamCallUpJob) Execute(context.Context) error {
	log.Info("National team call-up job")
	if err := common.CallUpNationalTeamSquads(nj.dbCtx); err != nil {
		log.Errorf("error calling up national team squads: %s", err)
	}
	return nil
}

func NewNationalTeamCallUpJob(dbCtx *common.DatabaseContext) *NationalTeamCallUpJob {
	return &NationalTeamCallUpJob{desc: "national team call-up job", dbCtx: dbCtx}
}

// FinishStuckSimulationsJob implements the quartz.Job interface: automatically re-simulates every
// league match and friendly left stuck at in_simulation = true (see FinishStuckSimulations) -
// previously only ever run on demand via AdminFinishStuckMatchesHandler, which meant a match a
// simulation pass claimed but never finished (a server crash/restart mid-run) stayed stuck until
// an admin happened to notice and trigger that endpoint by hand. Left unnoticed on a national-team
// friendly or a World Cup group match, that single stuck match also permanently blocks every
// other national-team fixture: GetCountriesWithPendingNationalTeamFriendly treats it as still
// pending forever, and, for a group match, IsWorldCupGroupStageComplete never sees the group
// finish, so HasActiveWorldCup never clears and ScheduleNationalTeamFriendlies skips every
// country indefinitely - not just the one actually stuck. Running this automatically closes that
// gap instead of depending on manual intervention.
type FinishStuckSimulationsJob struct {
	desc  string
	dbCtx *common.DatabaseContext
}

func (sj *FinishStuckSimulationsJob) Description() string {
	return sj.desc
}

func (sj *FinishStuckSimulationsJob) Execute(context.Context) error {
	result := FinishStuckSimulations(sj.dbCtx)
	if result.LeagueMatches > 0 || result.Friendlies > 0 {
		log.Infof("Finished stuck simulations job: %d league matches, %d friendlies", result.LeagueMatches, result.Friendlies)
	}
	return nil
}

func NewFinishStuckSimulationsJob(dbCtx *common.DatabaseContext) *FinishStuckSimulationsJob {
	return &FinishStuckSimulationsJob{desc: "finish stuck simulations job", dbCtx: dbCtx}
}

// LiveBroadcastRecoveryJob implements the quartz.Job interface: closes out any played match/cup
// tie/friendly whose live broadcast has gone stale - see RecoverStaleLiveBroadcasts.
type LiveBroadcastRecoveryJob struct {
	desc  string
	dbCtx *common.DatabaseContext
}

func (lj *LiveBroadcastRecoveryJob) Description() string {
	return lj.desc
}

func (lj *LiveBroadcastRecoveryJob) Execute(context.Context) error {
	RecoverStaleLiveBroadcasts(lj.dbCtx)
	return nil
}

func NewLiveBroadcastRecoveryJob(dbCtx *common.DatabaseContext) *LiveBroadcastRecoveryJob {
	return &LiveBroadcastRecoveryJob{desc: "live broadcast recovery job", dbCtx: dbCtx}
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
	if !claimRun(pj.dbCtx, "offer_response", 12*time.Hour) {
		return nil
	}
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
	if !claimRun(pj.dbCtx, "birthdays", 24*time.Hour) {
		return nil
	}
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
	// One replica per 5-minute tick: ProbAiLocalActivityPerTick is calibrated for one roll per
	// team per tick, and a second replica rolling too would double every AI team's activity.
	if !claimRun(pj.dbCtx, "ai_activity", 5*time.Minute) {
		return nil
	}
	log.Info("AI team daily activity job")
	RunAiTeamDailyActivity(pj.dbCtx)
	return nil
}

func NewAiActivityJob(dbCtx *common.DatabaseContext) *AiActivityJob {
	return &AiActivityJob{desc: "AI team daily activity job", dbCtx: dbCtx}
}
