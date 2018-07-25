package main

import (
	"time"

	gc "github.com/dgkanatsios/azuregameserversscalingkubernetes/apiserver/gc"
	webserver "github.com/dgkanatsios/azuregameserversscalingkubernetes/apiserver/webserver"
	log "github.com/sirupsen/logrus"
)

func main() {

	// initialize the garbage collector
	go gc.Run(1 * time.Minute)

	err := webserver.Run("dm4ish", "docker.io/dgkanatsios/docker_openarena_k8s:0.0.1", 8000)
	if err != nil {
		log.Fatalf("error creating WebServer: %s", err.Error())
	}
}
