package main

import (
	"flag"
	"reflect"
	"time"

	"github.com/dgkanatsios/azuregameserversscalingkubernetes/controller"
	dgsinformers "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/informers/externalversions"
	signals "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/signals"
	shared "github.com/dgkanatsios/azuregameserversscalingkubernetes/shared"
	log "github.com/sirupsen/logrus"
	informers "k8s.io/client-go/informers"
)

func main() {

	flag.Parse()
	client, dgsclient, err := shared.GetClientSet()

	if err != nil {
		log.Panic("Cannot initialize connection to cluster due to: %v", err)
	}

	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

	sharedInformerFactory := informers.NewSharedInformerFactory(client, 30*time.Second)
	dgsSharedInformerFactory := dgsinformers.NewSharedInformerFactory(dgsclient, 30*time.Second)

	dgsColController := controller.NewDedicatedGameServerCollectionController(client, dgsclient,
		dgsSharedInformerFactory.Azuregaming().V1alpha1().DedicatedGameServerCollections(), dgsSharedInformerFactory.Azuregaming().V1alpha1().DedicatedGameServers())

	dgsController := controller.NewDedicatedGameServerController(client, dgsclient,
		dgsSharedInformerFactory.Azuregaming().V1alpha1().DedicatedGameServers(), sharedInformerFactory.Core().V1().Pods(), sharedInformerFactory.Core().V1().Nodes())

	err = controller.InitializePortRegistry(dgsclient)
	if err != nil {
		log.Panicf("Cannot initialize PortRegistry:%v", err)
	}
	log.Info("Initialized Port Registry")

	go sharedInformerFactory.Start(stopCh)
	go dgsSharedInformerFactory.Start(stopCh)

	controllers := []controllerHelper{dgsColController, dgsController}

	runAllControllers(controllers, 1, stopCh)

}

// runAllControllers will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func runAllControllers(controllers []controllerHelper, controllerThreadiness int, stopCh <-chan struct{}) {

	// Start the informer factories to begin populating the informer caches
	log.Info("Starting controllers")

	// for all our controllers
	for _, c := range controllers {
		go func(ch controllerHelper) {
			err := ch.Run(controllerThreadiness, stopCh)
			if err != nil {
				log.Errorf("Cannot run controller %s", reflect.TypeOf(ch))
			}
		}(c)
	}

	<-stopCh
	log.Info("Controllers stopped")
}

type controllerHelper interface {
	Run(controllerThreadiness int, stopCh <-chan struct{}) error
}
