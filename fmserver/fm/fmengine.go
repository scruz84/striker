package fm

import (
	"footmanager/fmcommon"
	"footmanager/fmserver/fm/engine"
	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
	"os"
	"strconv"
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

	port, _ := strconv.Atoi(os.Getenv("DB_PORT"))
	databaseDetails := fmcommon.DatabaseDetails{
		Host:     os.Getenv("DB_HOST"),
		Port:     port,
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		DbName:   os.Getenv("DB_DATABASE"),
		SslMode:  "disable",
	}

	dbCtx, err := fmcommon.OpenDatabase(databaseDetails)

	err = dbCtx.Db.Ping()
	if err != nil {
		log.Panicln("Error opening database: %s", err)
	}

	//check actions
	engine.StartSimulateMatchesScheduler(engine.NewSimulateMatchesJob(dbCtx))
	engine.WaitUntilFinish()
	//go longPeriodicTimeTasks(dbCtx)
	//shortPeriodicTimeTasks(dbCtx)
}

func longPeriodicTimeTasks(dbCtx *fmcommon.DatabaseContext) {
	for {
		log.Info("Checking long periodic actions...")
		//engine.GenerateMatches(dbCtx)
		time.Sleep(30 * time.Second)
	}
}

func shortPeriodicTimeTasks(dbCtx *fmcommon.DatabaseContext) {
	for {
		log.Info("Checking short periodic actions...")
		engine.SimulateMatches(dbCtx)
		time.Sleep(10 * time.Second)
	}
}
