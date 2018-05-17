package main

import (
	"time"

	controller "github.com/dgkanatsios/azuregameserversscalingkubernetes/controller"
	shared "github.com/dgkanatsios/azuregameserversscalingkubernetes/shared"
	dgsinformers "github.com/dgkanatsios/azuregameserversscalingkubernetes/shared/pkg/client/informers/externalversions"
	informers "k8s.io/client-go/informers"
)

var client, dgsclient = shared.GetClientSet()

func main() {

	sharedInformers := informers.NewSharedInformerFactory(client, 24*time.Hour)
	dgsSharedInformers := dgsinformers.NewSharedInformerFactory(dgsclient, 24*time.Hour)
	controller := controller.NewDedicatedGameServerController(clientset, dedicatedgameserverclientset,
		sharedInformers.Core().V1().Pods(), dgsSharedInformers.Azure().V1().DedicatedGameServers())

	sharedInformers.Start(nil)
	dgsSharedInformers.Start(nil)

	controller.Run(nil)
}
