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

	err := godotenv.Load("app.env")
	if err != nil {
		log.Fatalf("error reading app.env file: %s", err.Error())
	}

	dbCtx, err := common.CreateDatabaseContext()

	err = dbCtx.Db.Ping()
	if err != nil {
		log.Panicln("Error opening database: %s", err)
	}

	//check actions
	StartSimulateMatchesScheduler(NewSimulateMatchesJob(dbCtx))
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
