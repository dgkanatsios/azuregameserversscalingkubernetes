package main

import (
	"flag"
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/dgkanatsios/azuregameserversscalingkubernetes/controller"
	shared "github.com/dgkanatsios/azuregameserversscalingkubernetes/shared"
	dgsinformers "github.com/dgkanatsios/azuregameserversscalingkubernetes/shared/pkg/client/informers/externalversions"
	signals "github.com/dgkanatsios/azuregameserversscalingkubernetes/shared/pkg/signals"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	informers "k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

var client, dgsclient = shared.GetClientSet()

func main() {

	flag.Parse()

	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

	sharedInformers := informers.NewSharedInformerFactory(client, 30*time.Minute)
	dgsSharedInformers := dgsinformers.NewSharedInformerFactory(dgsclient, 30*time.Minute)

	// controller := controller.NewDedicatedGameServerController(client, dgsclient,
	// 	sharedInformers.Core().V1().Pods(), sharedInformers.Core().V1().Nodes(),
	// 	dgsSharedInformers.Azure().V1().DedicatedGameServers(),
	// 	dgsSharedInformers.Azure().V1().DedicatedGameServerCollections())

	dgsColController := controller.NewDedicatedGameServerCollectionController(client, dgsclient,
		dgsSharedInformers.Azuregaming().V1alpha1().DedicatedGameServerCollections(), dgsSharedInformers.Azuregaming().V1alpha1().DedicatedGameServers())

	dgsController := controller.NewDedicatedGameServerController(client, dgsclient, dgsSharedInformers.Azuregaming().V1alpha1().DedicatedGameServers(),
		sharedInformers.Core().V1().Pods())

	podController := controller.NewPodController(client, sharedInformers.Core().V1().Pods())

	go sharedInformers.Start(stopCh)
	go dgsSharedInformers.Start(stopCh)

	controllers := []controllerHelper{dgsColController, dgsController, podController}

	if err := runAllControllers(controllers, 1, stopCh); err != nil {
		log.Fatalf("Error running controllers: %s", err.Error())
	}

}

// runAllControllers will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func runAllControllers(controllers []controllerHelper, controllerThreadiness int, stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()
	for _, c := range controllers {
		defer c.Workqueue().ShutDown()
	}

	// Start the informer factories to begin populating the informer caches
	log.Info("Starting controllers")

	// Wait for the caches for all controllers to be synced before starting workers
	log.Info("Waiting for informer caches to sync")
	for _, c := range controllers {
		if ok := cache.WaitForCacheSync(stopCh, c.ListersSynced()...); !ok {
			return fmt.Errorf("failed to wait for caches to sync")
		}
	}

	log.Info("Starting workers")
	// for all our controllers
	for _, c := range controllers {
		// Launch a number of workers to process resources
		for i := 0; i < controllerThreadiness; i++ {
			// runWorker will loop until "something bad" happens.  The .Until will
			// then rekick the worker after one second
			go wait.Until(c.RunWorker, time.Second, stopCh)
		}
	}

	log.Info("Started workers")
	<-stopCh
	log.Info("Shutting down workers")

	return nil
}

type controllerHelper interface {
	Workqueue() workqueue.RateLimitingInterface
	RunWorker()
	ListersSynced() []cache.InformerSynced
}
