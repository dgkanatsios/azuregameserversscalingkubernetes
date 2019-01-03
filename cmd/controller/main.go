package main

import (
	"flag"
	"reflect"
	"time"

	dgsinformers "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/informers/externalversions"
	controllers "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/controller"
	"github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/controller/autoscale"
	"github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/controller/dgs"
	"github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/controller/dgscollection"
	shared "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/shared"
	signals "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/signals"

	"github.com/jonboulle/clockwork"

	log "github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	informers "k8s.io/client-go/informers"
)

func main() {
	podautoscalerenabled := flag.Bool("podautoscaler", false, "Determines whether Pod AutoScaler is enabled. Default: false")
	controllerthreadiness := flag.Int("controllerthreadiness", 1, "Controller Threadiness. Default: 1")

	flag.Parse()

	client, dgsclient, err := shared.GetClientSet()

	if err != nil {
		log.Panicf("Cannot initialize connection to cluster due to: %v", err)
	}

	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

	sharedInformerFactory := informers.NewSharedInformerFactory(client, 30*time.Minute)
	dgsSharedInformerFactory := dgsinformers.NewSharedInformerFactory(dgsclient, 30*time.Minute)

	log.Info("Initializing Port Registry")
	portRegistry, err := controllers.NewPortRegistry(dgsclient, shared.MinPort, shared.MaxPort, metav1.NamespaceAll)
	if err != nil {
		log.Panicf("Cannot initialize Port Registry because of %s", err.Error())
	}

	dgsColController, err := dgscollection.NewDedicatedGameServerCollectionController(client, dgsclient,
		dgsSharedInformerFactory.Azuregaming().V1alpha1().DedicatedGameServerCollections(),
		dgsSharedInformerFactory.Azuregaming().V1alpha1().DedicatedGameServers(),
		portRegistry)

	if err != nil {
		log.Errorf("Cannot initialize DGSCollection controller due to %s", err.Error())
		return
	}

	dgsController := dgs.NewDedicatedGameServerController(client, dgsclient,
		dgsSharedInformerFactory.Azuregaming().V1alpha1().DedicatedGameServers(),
		sharedInformerFactory.Core().V1().Pods(), sharedInformerFactory.Core().V1().Nodes(), portRegistry)

	controllers := []controllerHelper{dgsColController, dgsController}

	if *podautoscalerenabled {
		podAutoscalerController := autoscale.NewActivePlayersAutoScalerController(client, dgsclient,
			dgsSharedInformerFactory.Azuregaming().V1alpha1().DedicatedGameServerCollections(),
			dgsSharedInformerFactory.Azuregaming().V1alpha1().DedicatedGameServers(), clockwork.NewRealClock())
		controllers = append(controllers, podAutoscalerController)
	}

	go sharedInformerFactory.Start(stopCh)
	go dgsSharedInformerFactory.Start(stopCh)

	runAllControllers(controllers, *controllerthreadiness, stopCh)

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
