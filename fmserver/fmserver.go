package main

import (
	"footmanager/fmserver/fm"
	log "github.com/sirupsen/logrus"
	"os"
)

func main() {
	args := os.Args[1:]

	if args[0] == "--svc" {
		fm.StartSvc()
	} else if args[0] == "--engine" {
		fm.StartEngine()
	} else {
		log.Fatalf("Usage --svc or --engine")
	}
}
