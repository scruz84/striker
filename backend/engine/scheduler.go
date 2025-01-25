package engine

import (
	"context"
	"github.com/reugn/go-quartz/quartz"
	"log"
)

var ctx = context.Background()
var sched = quartz.NewStdScheduler()

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

func WaitUntilFinish() {
	sched.Wait(ctx)
}
