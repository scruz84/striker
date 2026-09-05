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

func WaitUntilFinish() {
	sched.Wait(ctx)
}
