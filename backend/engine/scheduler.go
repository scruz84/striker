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

func WaitUntilFinish() {
	sched.Wait(ctx)
}
