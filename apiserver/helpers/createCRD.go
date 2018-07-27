package helpers

import (
	"log"

	"github.com/dgkanatsios/azuregameserversscalingkubernetes/shared"
)

func CreateDedicatedGameServerCRD(dgsInfo DedicatedGameServerInfo) (string, error) {

	if dgsInfo.Name == "" {
		dgsInfo.Name = "gameserver-" + shared.RandString(6)
	}

	for _, portInfo := range dgsInfo.Ports {
		//get a random port
		hostport, err := shared.GetRandomPort()
		if err != nil {
			return "", err
		}
		portInfo.HostPort = int32(hostport)
	}

	log.Printf("Creating DedicatedGameServer %s", dgsInfo.Name)

	dgs := shared.NewDedicatedGameServer(nil, dgsInfo.Name, dgsInfo.Ports, dgsInfo.StartMap, dgsInfo.Image)

	dgsInstance, err := shared.Dedicatedgameserverclientset.AzuregamingV1alpha1().DedicatedGameServers(shared.GameNamespace).Create(dgs)

	if err != nil {
		return "", err
	}
	return dgsInstance.ObjectMeta.Name, nil

}

func CreateDedicatedGameServerCollectionCRD(dgs DedicatedGameServerCollectionInfo) (string, error) {

	if dgs.Name == "" {
		dgs.Name = "dgscollection-" + shared.RandString(6)
	}

	log.Printf("Creating DedicatedGameServerCollection %s", dgs.Name)

	dgsCol := shared.NewDedicatedGameServerCollection(dgs.Name, dgs.StartMap, dgs.Image, dgs.Replicas, dgs.Ports)

	dgsColInstance, err := shared.Dedicatedgameserverclientset.AzuregamingV1alpha1().DedicatedGameServerCollections(shared.GameNamespace).Create(dgsCol)

	if err != nil {
		return "", err
	}
	return dgsColInstance.ObjectMeta.Name, nil

}
