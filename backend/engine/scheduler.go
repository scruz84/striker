package engine

import (
	"context"
	"log"

	"github.com/reugn/go-quartz/quartz"
)

var ctx = context.Background()
var sched, _ = quartz.NewStdScheduler()

func StartSimulateMatchesScheduler(simulateMatchesJob quartz.Job) {
	sched.Start(ctx)
	cronTrigger, err := quartz.NewCronTrigger("* 1/1 * * * *")
	if err != nil {
		log.Panic(err)
	}
	jobKey := quartz.NewJobKey(simulateMatchesJob.Description())
	jobDetail := quartz.NewJobDetail(simulateMatchesJob, jobKey)
	errJob := sched.ScheduleJob(jobDetail, cronTrigger)
	//errJob := sched.ScheduleJob(ctx, simulateMatchesJob, cronTrigger)
	if errJob != nil {
		log.Panic(errJob)
	}
}

func StartAdvanceSeasonScheduler(advanceSeasonJob quartz.Job) {
	sched.Start(ctx)
	// Checking season completion is much cheaper and lower-stakes than match simulation, so
	// it doesn't need per-minute granularity - every 5 minutes is plenty.
	cronTrigger, err := quartz.NewCronTrigger("0 */5 * * * *")
	if err != nil {
		log.Panic(err)
	}
	jobKey := quartz.NewJobKey(advanceSeasonJob.Description())
	jobDetail := quartz.NewJobDetail(advanceSeasonJob, jobKey)
	errJob := sched.ScheduleJob(jobDetail, cronTrigger)
	if errJob != nil {
		log.Panic(errJob)
	}
}

func StartFreeAgentProgressionScheduler(freeAgentProgressionJob quartz.Job) {
	sched.Start(ctx)
	// Once a day is plenty - there's no season boundary driving this anymore (see
	// init75_season_per_country.sql), just a slow background drift of the unowned player pool.
	cronTrigger, err := quartz.NewCronTrigger("0 0 2 * * ?")
	if err != nil {
		log.Panic(err)
	}
	jobKey := quartz.NewJobKey(freeAgentProgressionJob.Description())
	jobDetail := quartz.NewJobDetail(freeAgentProgressionJob, jobKey)
	errJob := sched.ScheduleJob(jobDetail, cronTrigger)
	if errJob != nil {
		log.Panic(errJob)
	}
}

func StartInternationalCupScheduler(internationalCupJob quartz.Job) {
	sched.Start(ctx)
	// Ticks daily, same as StartFreeAgentProgressionScheduler, but InternationalCupCadence
	// (engine/cups.go) gates how often it actually starts anything - cheap to check this often,
	// since most ticks just read season.last_international_cup_check and return.
	cronTrigger, err := quartz.NewCronTrigger("0 0 3 * * ?")
	if err != nil {
		log.Panic(err)
	}
	jobKey := quartz.NewJobKey(internationalCupJob.Description())
	jobDetail := quartz.NewJobDetail(internationalCupJob, jobKey)
	errJob := sched.ScheduleJob(jobDetail, cronTrigger)
	if errJob != nil {
		log.Panic(errJob)
	}
}

func StartCupScheduler(cupJob quartz.Job) {
	sched.Start(ctx)
	// Same cadence as match simulation - cup ties are scheduled the same way league matches
	// are (a fixed delay after being paired), so they need the same per-minute granularity to
	// fire close to on time.
	cronTrigger, err := quartz.NewCronTrigger("* 1/1 * * * *")
	if err != nil {
		log.Panic(err)
	}
	jobKey := quartz.NewJobKey(cupJob.Description())
	jobDetail := quartz.NewJobDetail(cupJob, jobKey)
	errJob := sched.ScheduleJob(jobDetail, cronTrigger)
	if errJob != nil {
		log.Panic(errJob)
	}
}

func StartFreeAgentInjectionScheduler(freeAgentInjectionJob quartz.Job) {
	sched.Start(ctx)
	// A real calendar week, independent of match simulation - Sunday midnight UTC, so a fresh
	// batch of free agents is sitting on the market ahead of Monday's usual transfer activity.
	cronTrigger, err := quartz.NewCronTrigger("0 0 0 ? * SUN")
	if err != nil {
		log.Panic(err)
	}
	jobKey := quartz.NewJobKey(freeAgentInjectionJob.Description())
	jobDetail := quartz.NewJobDetail(freeAgentInjectionJob, jobKey)
	errJob := sched.ScheduleJob(jobDetail, cronTrigger)
	if errJob != nil {
		log.Panic(errJob)
	}
}

func StartFriendlyScheduler(friendlyJob quartz.Job) {
	sched.Start(ctx)
	// Same per-minute cadence as cup ties - friendlies are scheduled a fixed delay after
	// acceptance (see common.friendlyScheduleDelay), so they need the same granularity to fire
	// close to on time.
	cronTrigger, err := quartz.NewCronTrigger("* 1/1 * * * *")
	if err != nil {
		log.Panic(err)
	}
	jobKey := quartz.NewJobKey(friendlyJob.Description())
	jobDetail := quartz.NewJobDetail(friendlyJob, jobKey)
	errJob := sched.ScheduleJob(jobDetail, cronTrigger)
	if errJob != nil {
		log.Panic(errJob)
	}
}

func StartNationalTeamFriendlyScheduler(nationalTeamFriendlyJob quartz.Job) {
	sched.Start(ctx)
	// Same cadence as StartAdvanceSeasonScheduler/StartAiActivityScheduler - cheap per tick (see
	// common.ScheduleNationalTeamFriendlies's own doc comment for why this needs to run often
	// rather than on a single fixed weekly timestamp: a country should be re-paired within
	// minutes of its previous friendly being played, not however long is left until the next
	// rigid weekly slot). Unlike club friendlies there's no human to challenge/accept, so there's
	// no per-team cap to enforce instead - GetCountriesWithPendingNationalTeamFriendly's "already
	// waiting on a fixture" check is what keeps this idempotent across ticks.
	cronTrigger, err := quartz.NewCronTrigger("0 */5 * * * *")
	if err != nil {
		log.Panic(err)
	}
	jobKey := quartz.NewJobKey(nationalTeamFriendlyJob.Description())
	jobDetail := quartz.NewJobDetail(nationalTeamFriendlyJob, jobKey)
	errJob := sched.ScheduleJob(jobDetail, cronTrigger)
	if errJob != nil {
		log.Panic(errJob)
	}
}

func StartNationalTeamCallUpScheduler(nationalTeamCallUpJob quartz.Job) {
	sched.Start(ctx)
	// Same 5-minute cadence as StartNationalTeamFriendlyScheduler - frequent enough to close a
	// national team's call-up (see common.CallUpNationalTeamSquads/common.nationalTeamCallUpWindow)
	// within minutes of crossing the one-day-before-kickoff mark, without needing per-minute
	// granularity the way kickoff-triggered friendly simulation does.
	cronTrigger, err := quartz.NewCronTrigger("0 */5 * * * *")
	if err != nil {
		log.Panic(err)
	}
	jobKey := quartz.NewJobKey(nationalTeamCallUpJob.Description())
	jobDetail := quartz.NewJobDetail(nationalTeamCallUpJob, jobKey)
	errJob := sched.ScheduleJob(jobDetail, cronTrigger)
	if errJob != nil {
		log.Panic(errJob)
	}
}

func StartFinishStuckSimulationsScheduler(finishStuckSimulationsJob quartz.Job) {
	sched.Start(ctx)
	// Deliberately slower than the other periodic jobs' 5-minute cadence: GetStuckMatches/
	// GetStuckFriendlies have no "claimed at" timestamp to tell a genuinely abandoned
	// in_simulation = true row (a crash mid-run) apart from one a normal simulation pass is still
	// actively working through right now, so this can only lean on running rarely enough that a
	// real simulation (in practice well under a second of in-process work) is never still running
	// when a tick lands - 15 minutes keeps that collision essentially impossible while still
	// turning "stuck until an admin happens to notice and hits the manual endpoint" into "self-heals
	// within 15 minutes" (see engine.FinishStuckSimulations's own doc comment for why a single
	// stuck national-team/World Cup match is worse than just that one match never finishing).
	cronTrigger, err := quartz.NewCronTrigger("0 */15 * * * *")
	if err != nil {
		log.Panic(err)
	}
	jobKey := quartz.NewJobKey(finishStuckSimulationsJob.Description())
	jobDetail := quartz.NewJobDetail(finishStuckSimulationsJob, jobKey)
	errJob := sched.ScheduleJob(jobDetail, cronTrigger)
	if errJob != nil {
		log.Panic(errJob)
	}
}

func StartOfferResponseScheduler(offerResponseJob quartz.Job) {
	sched.Start(ctx)
	// Twice a day (midnight and noon UTC) is enough to keep the "the club must respond within a
	// day or two" promise (issue #61) without needing per-minute granularity - unlike match/cup/
	// friendly simulation, nothing about this is time-critical to the minute.
	cronTrigger, err := quartz.NewCronTrigger("0 0 0,12 * * ?")
	if err != nil {
		log.Panic(err)
	}
	jobKey := quartz.NewJobKey(offerResponseJob.Description())
	jobDetail := quartz.NewJobDetail(offerResponseJob, jobKey)
	errJob := sched.ScheduleJob(jobDetail, cronTrigger)
	if errJob != nil {
		log.Panic(errJob)
	}
}

func StartBirthdayScheduler(birthdayJob quartz.Job) {
	sched.Start(ctx)
	// Once a day is plenty - a player's birthday anchor only ever moves in whole-day steps (see
	// engine.BirthdayCycleDays), so nothing is missed by checking less often than match
	// simulation. Offset to 01:00 UTC, clear of the other daily/weekly jobs' 00:00/12:00 slots.
	cronTrigger, err := quartz.NewCronTrigger("0 0 1 * * ?")
	if err != nil {
		log.Panic(err)
	}
	jobKey := quartz.NewJobKey(birthdayJob.Description())
	jobDetail := quartz.NewJobDetail(birthdayJob, jobKey)
	errJob := sched.ScheduleJob(jobDetail, cronTrigger)
	if errJob != nil {
		log.Panic(errJob)
	}
}

func StartAiActivityScheduler(aiActivityJob quartz.Job) {
	sched.Start(ctx)
	// Same cadence as StartAdvanceSeasonScheduler - cheap per tick (a handful of teams rolling a
	// low-probability chance each), and localActivityWindowTicksPerDay (engine/aiteams.go) is
	// built from this exact interval, so changing it here means updating that constant too.
	cronTrigger, err := quartz.NewCronTrigger("0 */5 * * * *")
	if err != nil {
		log.Panic(err)
	}
	jobKey := quartz.NewJobKey(aiActivityJob.Description())
	jobDetail := quartz.NewJobDetail(aiActivityJob, jobKey)
	errJob := sched.ScheduleJob(jobDetail, cronTrigger)
	if errJob != nil {
		log.Panic(errJob)
	}
}

func StartLiveBroadcastRecoveryScheduler(liveBroadcastRecoveryJob quartz.Job) {
	sched.Start(ctx)
	// Every 5 minutes is frequent enough to close a stale broadcast (see
	// engine.RecoverStaleLiveBroadcasts) promptly after it crosses common.LiveMatchStaleTimeout
	// (18 minutes) without needing per-minute granularity - staleness is itself only measured in
	// minutes, so a few minutes' extra delay in detecting it is immaterial. Same cadence as
	// StartAdvanceSeasonScheduler/StartAiActivityScheduler for the same reason: cheap per tick (a
	// small, already-"live" set of matches plus a couple of Redis reads each).
	cronTrigger, err := quartz.NewCronTrigger("0 */5 * * * *")
	if err != nil {
		log.Panic(err)
	}
	jobKey := quartz.NewJobKey(liveBroadcastRecoveryJob.Description())
	jobDetail := quartz.NewJobDetail(liveBroadcastRecoveryJob, jobKey)
	errJob := sched.ScheduleJob(jobDetail, cronTrigger)
	if errJob != nil {
		log.Panic(errJob)
	}
}

func WaitUntilFinish() {
	sched.Wait(ctx)
}
