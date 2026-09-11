package engine

import (
	"github.com/joho/godotenv"
	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
	"os"
	"time"
)

func StartEngine() {
	log.Info("Starting football manager engine!")
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)

	if err := godotenv.Load("app.env"); err != nil {
		log.Warnf("no app.env file loaded, relying on environment variables: %s", err.Error())
	}

	dbCtx, err := common.CreateDatabaseContext()

	err = dbCtx.Db.Ping()
	if err != nil {
		log.Panicf("Error opening database: %s", err)
	}

	//check actions
	StartSimulateMatchesScheduler(NewSimulateMatchesJob(dbCtx))
	StartAdvanceSeasonScheduler(NewAdvanceSeasonJob(dbCtx))
	StartCupScheduler(NewCupJob(dbCtx))
	StartWeeklyWagesScheduler(NewWeeklyWagesJob(dbCtx))
	StartFriendlyScheduler(NewFriendlyJob(dbCtx))
	WaitUntilFinish()
	//go longPeriodicTimeTasks(dbCtx)
	//shortPeriodicTimeTasks(dbCtx)
}

func longPeriodicTimeTasks(dbCtx *common.DatabaseContext) {
	for {
		log.Info("Checking long periodic actions...")
		//GenerateMatches(dbCtx)
		time.Sleep(30 * time.Second)
	}
}

func shortPeriodicTimeTasks(dbCtx *common.DatabaseContext) {
	for {
		log.Info("Checking short periodic actions...")
		SimulateMatches(dbCtx)
		time.Sleep(10 * time.Second)
	}
}
