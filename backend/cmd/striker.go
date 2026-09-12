package main

import (
	"context"
	"github.com/joho/godotenv"
	"github.com/scruz84/striker/common"
	"github.com/scruz84/striker/engine"
	"github.com/scruz84/striker/handlers"
	"github.com/scruz84/striker/service"
	log "github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"slices"
	"syscall"
	"time"
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

	// serviceName distinguishes the two long-running process shapes in exported metrics (see
	// common.InitMetrics) - svc and engine can run as separate containers exporting to the same
	// collector at once (see backend/docker-compose.yml), so their metrics need distinct
	// service.name resource attributes to tell apart.
	serviceName := "striker-engine"
	switch {
	case runSvc && runEngine:
		serviceName = "striker-combined"
	case runSvc:
		serviceName = "striker-svc"
	}
	shutdownMetrics, err := common.InitMetrics(serviceName)
	if err != nil {
		log.Warnf("failed to initialize OpenTelemetry metrics export: %s", err)
	}
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdownMetrics(ctx); err != nil {
			log.Warnf("error flushing OpenTelemetry metrics on shutdown: %s", err)
		}
		os.Exit(0)
	}()

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
