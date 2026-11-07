package engine

import (
	"context"
	"log"
	"time"

	"github.com/reugn/go-quartz/quartz"
	"github.com/scruz84/striker/common"
)

var ctx = context.Background()
var sched, _ = quartz.NewStdScheduler()

// instrumentedJob wraps a job so every scheduled run is counted and timed under `name`, a stable
// snake_case label (striker.engine.job.runs / .duration / .last_run - see
// common.RecordEngineJobRun). A run whose job returns early because another replica holds the
// claim (claimRun) still counts: the trigger fired on this replica.
type instrumentedJob struct {
	inner quartz.Job
	name  string
}

func (j instrumentedJob) Description() string { return j.inner.Description() }

func (j instrumentedJob) Execute(jobCtx context.Context) error {
	started := time.Now()
	err := j.inner.Execute(jobCtx)
	common.RecordEngineJobRun(jobCtx, j.name, started, time.Now())
	return err
}

// scheduleJob starts the scheduler (idempotent) and schedules job on a quartz cron expression
// (seconds first), wrapped in instrumentedJob under the metric label `name`.
func scheduleJob(job quartz.Job, cron string, name string) {
	sched.Start(ctx)
	cronTrigger, err := quartz.NewCronTrigger(cron)
	if err != nil {
		log.Panic(err)
	}
	wrapped := instrumentedJob{inner: job, name: name}
	jobDetail := quartz.NewJobDetail(wrapped, quartz.NewJobKey(job.Description()))
	if err := sched.ScheduleJob(jobDetail, cronTrigger); err != nil {
		log.Panic(err)
	}
}

func StartSimulateMatchesScheduler(simulateMatchesJob quartz.Job) {
	scheduleJob(simulateMatchesJob, "* 1/1 * * * *", "simulate_matches")
}

func StartAdvanceSeasonScheduler(advanceSeasonJob quartz.Job) {
	// Checking season completion is much cheaper and lower-stakes than match simulation, so
	// it doesn't need per-minute granularity - every 5 minutes is plenty.
	scheduleJob(advanceSeasonJob, "0 */5 * * * *", "advance_season")
}

func StartFreeAgentProgressionScheduler(freeAgentProgressionJob quartz.Job) {
	// Once a day is plenty - there's no season boundary driving this anymore (see
	// init75_season_per_country.sql), just a slow background drift of the unowned player pool.
	scheduleJob(freeAgentProgressionJob, "0 0 2 * * ?", "free_agent_progression")
}

func StartInternationalCupScheduler(internationalCupJob quartz.Job) {
	// Ticks daily, same as StartFreeAgentProgressionScheduler, but WorldCupCycleCadence
	// (engine/cups.go) gates how often it actually starts anything - cheap to check this often,
	// since most ticks just read season.last_international_cup_check and return.
	scheduleJob(internationalCupJob, "0 0 3 * * ?", "international_cup")
}

func StartRegionalCupScheduler(regionalCupJob quartz.Job) {
	// Every 5 minutes, half a minute after the season-rollover check, so a region whose last
	// country just rolled over starts its next edition within minutes. Most ticks only read a
	// few rows per region and return.
	scheduleJob(regionalCupJob, "30 */5 * * * *", "regional_cup")
}

func StartCupScheduler(cupJob quartz.Job) {
	// Same cadence as match simulation - cup ties are scheduled the same way league matches
	// are (a fixed delay after being paired), so they need the same per-minute granularity to
	// fire close to on time.
	scheduleJob(cupJob, "* 1/1 * * * *", "cups")
}

func StartFreeAgentInjectionScheduler(freeAgentInjectionJob quartz.Job) {
	// A real calendar week, independent of match simulation - Sunday midnight UTC, so a fresh
	// batch of free agents is sitting on the market ahead of Monday's usual transfer activity.
	scheduleJob(freeAgentInjectionJob, "0 0 0 ? * SUN", "free_agent_injection")
}

func StartFriendlyScheduler(friendlyJob quartz.Job) {
	// Same per-minute cadence as cup ties - friendlies are scheduled a fixed delay after
	// acceptance (see common.friendlyScheduleDelay), so they need the same granularity to fire
	// close to on time.
	scheduleJob(friendlyJob, "* 1/1 * * * *", "friendlies")
}

func StartNationalTeamFriendlyScheduler(nationalTeamFriendlyJob quartz.Job) {
	// Same cadence as StartAdvanceSeasonScheduler/StartAiActivityScheduler - cheap per tick (see
	// common.ScheduleNationalTeamFriendlies's own doc comment for why this needs to run often
	// rather than on a single fixed weekly timestamp: a country should be re-paired within
	// minutes of its previous friendly being played, not however long is left until the next
	// rigid weekly slot). Unlike club friendlies there's no human to challenge/accept, so there's
	// no per-team cap to enforce instead - GetCountriesWithPendingNationalTeamFriendly's "already
	// waiting on a fixture" check is what keeps this idempotent across ticks.
	scheduleJob(nationalTeamFriendlyJob, "0 */5 * * * *", "national_team_friendlies")
}

func StartNationalTeamCallUpScheduler(nationalTeamCallUpJob quartz.Job) {
	// Same 5-minute cadence as StartNationalTeamFriendlyScheduler - frequent enough to close a
	// national team's call-up (see common.CallUpNationalTeamSquads/common.nationalTeamCallUpWindow)
	// within minutes of crossing the one-day-before-kickoff mark, without needing per-minute
	// granularity the way kickoff-triggered friendly simulation does.
	scheduleJob(nationalTeamCallUpJob, "0 */5 * * * *", "national_team_call_ups")
}

func StartFinishStuckSimulationsScheduler(finishStuckSimulationsJob quartz.Job) {
	// GetStuckMatches/GetStuckFriendlies only take rows claimed at least
	// common.StuckSimulationAge ago, so a row a live simulation pass is still working through is
	// never re-simulated, and they re-claim atomically, so two replicas never re-simulate the same
	// row. Every 15 minutes turns "stuck until an admin happens to notice and hits the manual
	// endpoint" into "self-heals within half an hour" (see engine.FinishStuckSimulations's own doc
	// comment for why a single stuck national-team/World Cup match is worse than just that one
	// match never finishing).
	scheduleJob(finishStuckSimulationsJob, "0 */15 * * * *", "finish_stuck_simulations")
}

func StartOfferResponseScheduler(offerResponseJob quartz.Job) {
	// Twice a day (midnight and noon UTC) is enough to keep the "the club must respond within a
	// day or two" promise (issue #61) without needing per-minute granularity - unlike match/cup/
	// friendly simulation, nothing about this is time-critical to the minute.
	scheduleJob(offerResponseJob, "0 0 0,12 * * ?", "offer_response")
}

func StartBirthdayScheduler(birthdayJob quartz.Job) {
	// Once a day is plenty - a player's birthday anchor only ever moves in whole-day steps (see
	// engine.BirthdayCycleDays), so nothing is missed by checking less often than match
	// simulation. Offset to 01:00 UTC, clear of the other daily/weekly jobs' 00:00/12:00 slots.
	scheduleJob(birthdayJob, "0 0 1 * * ?", "birthdays")
}

func StartAiActivityScheduler(aiActivityJob quartz.Job) {
	// Same cadence as StartAdvanceSeasonScheduler - cheap per tick (a handful of teams rolling a
	// low-probability chance each), and localActivityWindowTicksPerDay (engine/aiteams.go) is
	// built from this exact interval, so changing it here means updating that constant too.
	scheduleJob(aiActivityJob, "0 */5 * * * *", "ai_activity")
}

func StartLiveBroadcastRecoveryScheduler(liveBroadcastRecoveryJob quartz.Job) {
	// Every 5 minutes is frequent enough to close a stale broadcast (see
	// engine.RecoverStaleLiveBroadcasts) promptly after it crosses common.LiveMatchStaleTimeout
	// (18 minutes) without needing per-minute granularity - staleness is itself only measured in
	// minutes, so a few minutes' extra delay in detecting it is immaterial. Same cadence as
	// StartAdvanceSeasonScheduler/StartAiActivityScheduler for the same reason: cheap per tick (a
	// small, already-"live" set of matches plus a couple of Redis reads each).
	scheduleJob(liveBroadcastRecoveryJob, "0 */5 * * * *", "live_broadcast_recovery")
}

func WaitUntilFinish() {
	sched.Wait(ctx)
}

func StartAccountLifecycleScheduler(accountLifecycleJob quartz.Job) {
	// Once a day at 03:17 UTC, clear of the other daily jobs' slots: warns idle owners, releases
	// the clubs of owners inactive for common.InactivityReleaseDays and erases accounts idle for
	// common.AccountErasureDays (see engine.RunAccountLifecycle). One replica per day wins the
	// claim in AccountLifecycleJob.Execute.
	scheduleJob(accountLifecycleJob, "0 17 3 * * ?", "account_lifecycle")
}

func StartResetReleasedTeamsScheduler(resetJob quartz.Job) {
	// Every 15 minutes: resets the clubs released since the last pass (squad swap, budget, names).
	// ResetReleasedTeam skips a club with a match in or near its live window and retries on the
	// next pass, so the cadence is also the longest a club waits behind a busy fixture.
	scheduleJob(resetJob, "0 */15 * * * *", "reset_released_teams")
}
