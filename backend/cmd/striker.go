package main

import (
	"github.com/joho/godotenv"
	"github.com/scruz84/striker/engine"
	"github.com/scruz84/striker/handlers"
	"github.com/scruz84/striker/service"
	log "github.com/sirupsen/logrus"
	"os"
	"slices"
)

func main() {
	args := os.Args[1:]
	if err := godotenv.Load("app.env"); err != nil {
		log.Warnf("no app.env file loaded, relying on environment variables: %s", err.Error())
	}

	runSvc := slices.Contains(args, "--svc")
	runEngine := slices.Contains(args, "--engine")
	runInitialize := slices.Contains(args, "--initialize")

	if !runSvc && !runEngine && !runInitialize {
		log.Fatalf("Usage: --svc, --engine, --initialize (--svc and --engine can be combined to run both in one process)")
	}
	if runInitialize && (runSvc || runEngine) {
		log.Fatalf("--initialize cannot be combined with --svc or --engine")
	}

	if runInitialize {
		handlers.InitializeDatabase()
		return
	}

	// StartSvc and StartEngine both block forever (an HTTP server and a scheduler wait,
	// respectively), so when both are requested one has to run in the background - otherwise
	// main would return, and with it the process would exit, as soon as the first call returned.
	if runSvc && runEngine {
		go service.StartSvc()
		engine.StartEngine()
		return
	}
	if runSvc {
		service.StartSvc()
		return
	}
	engine.StartEngine()
}
