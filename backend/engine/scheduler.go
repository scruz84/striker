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

func StartWeeklyWagesScheduler(weeklyWagesJob quartz.Job) {
	sched.Start(ctx)
	// A real calendar week, not tied to match simulation at all - see engine.PayWeeklyWages.
	// Midnight Monday UTC; a missed firing (e.g. the server was down at that exact minute)
	// simply waits for the next one, same as every other job here.
	cronTrigger, err := quartz.NewCronTrigger("0 0 0 ? * MON")
	if err != nil {
		log.Panic(err)
	}
	jobKey := quartz.NewJobKey(weeklyWagesJob.Description())
	jobDetail := quartz.NewJobDetail(weeklyWagesJob, jobKey)
	errJob := sched.ScheduleJob(jobDetail, cronTrigger)
	if errJob != nil {
		log.Panic(errJob)
	}
}

func StartFreeAgentInjectionScheduler(freeAgentInjectionJob quartz.Job) {
	sched.Start(ctx)
	// A real calendar week, same cadence as StartWeeklyWagesScheduler - offset to Sunday
	// midnight rather than Monday so a fresh batch of free agents is already sitting on the
	// market by the time StartWeeklyWagesScheduler's Monday run charges everyone's wage bill.
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

func WaitUntilFinish() {
	sched.Wait(ctx)
}
