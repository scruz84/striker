package main

import (
	"github.com/joho/godotenv"
	"github.com/scruz84/striker/engine"
	"github.com/scruz84/striker/handlers"
	"github.com/scruz84/striker/service"
	log "github.com/sirupsen/logrus"
	"os"
)

func main() {
	args := os.Args[1:]
	if err := godotenv.Load("app.env"); err != nil {
		log.Warnf("no app.env file loaded, relying on environment variables: %s", err.Error())
	}

	if args[0] == "--svc" {
		service.StartSvc()
	} else if args[0] == "--engine" {
		engine.StartEngine()
	} else if args[0] == "--initialize" {
		handlers.InitializeDatabase()
	} else {
		log.Fatalf("Usage --svc or --engine")
	}
}
