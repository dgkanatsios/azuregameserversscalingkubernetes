package helpers

import (
	log "github.com/sirupsen/logrus"

	"github.com/dgkanatsios/azuregameserversscalingkubernetes/shared"
)

func CreateDedicatedGameServerCRD(dgsInfo DedicatedGameServerInfo) (dgsName string, err error) {

	if dgsInfo.Name == "" {
		dgsInfo.Name = shared.GenerateRandomName("gamesever")
	}

	//TODO: we used to pass a random port here. Maybe we should later the controller to create one if it's 0?

	log.Printf("Creating DedicatedGameServer %s", dgsInfo.Name)

	dgs := shared.NewDedicatedGameServer(nil, dgsInfo.Name, dgsInfo.Ports, dgsInfo.StartMap, dgsInfo.Image)

	_, dgsClient, err := shared.GetClientSet()
	if err != nil {
		return "", err
	}

	dgsInstance, err := dgsClient.AzuregamingV1alpha1().DedicatedGameServers(shared.GameNamespace).Create(dgs)

	if err != nil {
		return "", err
	}
	return dgsInstance.ObjectMeta.Name, nil

}

func CreateDedicatedGameServerCollectionCRD(dgs DedicatedGameServerCollectionInfo) (dgsColName string, err error) {

	if dgs.Name == "" {
		dgs.Name = shared.GenerateRandomName("dedicatedgameservercollection")
	}

	log.Printf("Creating DedicatedGameServerCollection %s", dgs.Name)

	dgsCol := shared.NewDedicatedGameServerCollection(dgs.Name, dgs.StartMap, dgs.Image, dgs.Replicas, dgs.Ports)

	_, dgsClient, err := shared.GetClientSet()
	if err != nil {
		return "", err
	}

	dgsColInstance, err := dgsClient.AzuregamingV1alpha1().DedicatedGameServerCollections(shared.GameNamespace).Create(dgsCol)

	if err != nil {
		return "", err
	}
	return dgsColInstance.ObjectMeta.Name, nil

}
