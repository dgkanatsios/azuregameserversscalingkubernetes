package main

import (
	"flag"
	"time"

	log "github.com/Sirupsen/logrus"
	controller "github.com/dgkanatsios/azuregameserversscalingkubernetes/controller"
	shared "github.com/dgkanatsios/azuregameserversscalingkubernetes/shared"
	dgsinformers "github.com/dgkanatsios/azuregameserversscalingkubernetes/shared/pkg/client/informers/externalversions"
	signals "github.com/dgkanatsios/azuregameserversscalingkubernetes/shared/pkg/signals"
	informers "k8s.io/client-go/informers"
)

var client, dgsclient = shared.GetClientSet()

func main() {

	flag.Parse()

	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

	sharedInformers := informers.NewSharedInformerFactory(client, 30*time.Second)
	dgsSharedInformers := dgsinformers.NewSharedInformerFactory(dgsclient, 30*time.Second)

	controller := controller.NewDedicatedGameServerController(client, dgsclient,
		sharedInformers.Core().V1().Pods(), dgsSharedInformers.Azure().V1().DedicatedGameServers())

	go sharedInformers.Start(stopCh)
	go dgsSharedInformers.Start(stopCh)

	if err := controller.Run(2, stopCh); err != nil {
		log.Fatalf("Error running controller: %s", err.Error())
	}
}
