package main

import (
	"footmanager/fmcommon"
	"footmanager/fmserver/fm/engine"
	log "github.com/sirupsen/logrus"
	"os"
	"time"
)

func main() {
	log.Info("Starting football manager engine!")
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)

	databaseDetails := fmcommon.DatabaseDetails{
		Host:     "localhost",
		Port:     5432,
		User:     "sergio",
		Password: "sergio",
		DbName:   "footmanager",
		SslMode:  "disable",
	}

	dbCtx, err := fmcommon.OpenDatabase(databaseDetails)

	err = dbCtx.Db.Ping()
	if err != nil {
		log.Panicln("Error opening database: %s", err)
	}

	//initialize cache of prepared statements
	initEngineStatements(dbCtx)

	//check actions
	go longPeriodicTimeTasks(dbCtx)
	shortPeriodicTimeTasks(dbCtx)
}

// Initialize database statements
func initEngineStatements(dbCtx *fmcommon.DatabaseContext) {
	var preparedStmt, _ = dbCtx.Db.Prepare(fmcommon.GetSimulableMatchesStmtSql)
	dbCtx.PreparedStmts[fmcommon.GetSimulableMatchesStmt] = preparedStmt
	preparedStmt, _ = dbCtx.Db.Prepare(fmcommon.GetLineupsForMatchStmtSql)
	dbCtx.PreparedStmts[fmcommon.GetLineupsForMatchStmt] = preparedStmt
	preparedStmt, _ = dbCtx.Db.Prepare(fmcommon.GetPlayersFromTeamStmtSql)
	dbCtx.PreparedStmts[fmcommon.GetPlayersFromTeamStmt] = preparedStmt
	preparedStmt, _ = dbCtx.Db.Prepare(fmcommon.InsertMatchEventStmtSql)
	dbCtx.PreparedStmts[fmcommon.InsertMatchEventStmt] = preparedStmt
	preparedStmt, _ = dbCtx.Db.Prepare(fmcommon.UpdateMatchStmtSql)
	dbCtx.PreparedStmts[fmcommon.UpdateMatchStmt] = preparedStmt
}

func longPeriodicTimeTasks(dbCtx *fmcommon.DatabaseContext) {
	for {
		log.Info("Checking long periodic actions...")
		engine.GenerateMatches(dbCtx)
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
