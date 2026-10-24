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

	// Applies any migration added since this database was last started - see common.EnsureSchema.
	if err := common.EnsureSchema(dbCtx); err != nil {
		log.Panicf("Error ensuring schema is up to date: %s", err)
	}

	common.ConfigureWebPushNotifier(dbCtx)

	if err := common.RegisterDBPoolMetrics(dbCtx.Db); err != nil {
		log.Errorf("error registering database pool metrics: %s", err)
	}

	if err := common.RegisterMarketGauges(dbCtx); err != nil {
		log.Errorf("error registering transfer/loan market OpenTelemetry gauges: %s", err)
	}

	//check actions
	StartSimulateMatchesScheduler(NewSimulateMatchesJob(dbCtx))
	StartAdvanceSeasonScheduler(NewAdvanceSeasonJob(dbCtx))
	StartFreeAgentProgressionScheduler(NewFreeAgentProgressionJob(dbCtx))
	StartInternationalCupScheduler(NewInternationalCupJob(dbCtx))
	StartCupScheduler(NewCupJob(dbCtx))
	StartFreeAgentInjectionScheduler(NewFreeAgentInjectionJob(dbCtx))
	StartFriendlyScheduler(NewFriendlyJob(dbCtx))
	StartNationalTeamFriendlyScheduler(NewNationalTeamFriendlyJob(dbCtx))
	StartNationalTeamCallUpScheduler(NewNationalTeamCallUpJob(dbCtx))
	StartFinishStuckSimulationsScheduler(NewFinishStuckSimulationsJob(dbCtx))
	StartLiveBroadcastRecoveryScheduler(NewLiveBroadcastRecoveryJob(dbCtx))
	StartOfferResponseScheduler(NewOfferResponseJob(dbCtx))
	StartBirthdayScheduler(NewBirthdayJob(dbCtx))
	StartAiActivityScheduler(NewAiActivityJob(dbCtx))
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
