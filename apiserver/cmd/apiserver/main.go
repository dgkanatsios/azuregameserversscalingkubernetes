package main

import (
	"time"

	gc "github.com/dgkanatsios/azuregameserversscalingkubernetes/apiserver/gc"
	webserver "github.com/dgkanatsios/azuregameserversscalingkubernetes/apiserver/webserver"
	"github.com/dgkanatsios/azuregameserversscalingkubernetes/shared"
	log "github.com/sirupsen/logrus"
)

func main() {

	// initialize the garbage collector
	go gc.Run(1 * time.Minute)

	err := webserver.Run("dm4ish", shared.GameDockerImage, 8000)
	if err != nil {
		log.Fatalf("error creating WebServer: %s", err.Error())
	}
}
