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
	err := godotenv.Load("app.env")
	if err != nil {
		log.Panicf("error reading app.env file: %s", err.Error())
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
