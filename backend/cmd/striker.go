package main

import (
	"github.com/scruz84/striker/engine"
	"github.com/scruz84/striker/service"
	log "github.com/sirupsen/logrus"
	"os"
)

func main() {
	args := os.Args[1:]

	if args[0] == "--svc" {
		service.StartSvc()
	} else if args[0] == "--engine" {
		engine.StartEngine()
	} else {
		log.Fatalf("Usage --svc or --engine")
	}
}
