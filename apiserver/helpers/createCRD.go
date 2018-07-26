package helpers

import (
	"log"

	"github.com/dgkanatsios/azuregameserversscalingkubernetes/shared"
)

func CreateDedicatedGameServerCRD(startmap string, dockerImage string) (string, error) {
	name := "openarena-" + shared.RandString(6)

	//get a random port
	port, err := shared.GetRandomPort()

	if err != nil {
		return "", err
	}

	log.Printf("Creating DedicatedGameServer %s", name)

	dgs := shared.NewDedicatedGameServer(nil, name, port, startmap, dockerImage)

	dgsInstance, err := shared.Dedicatedgameserverclientset.AzuregamingV1alpha1().DedicatedGameServers(shared.GameNamespace).Create(dgs)

	if err != nil {
		return "", err
	}
	return dgsInstance.ObjectMeta.Name, nil

}

func CreateDedicatedGameServerCollectionCRD(startmap string, dockerImage string) (string, error) {
	name := "openarenacollection-" + shared.RandString(6)

	log.Printf("Creating DedicatedGameServerCollection %s", name)

	dgsCol := shared.NewDedicatedGameServerCollection(name, startmap, dockerImage, 5)

	dgsColInstance, err := shared.Dedicatedgameserverclientset.AzuregamingV1alpha1().DedicatedGameServerCollections(shared.GameNamespace).Create(dgsCol)

	if err != nil {
		return "", err
	}
	return dgsColInstance.ObjectMeta.Name, nil

}
