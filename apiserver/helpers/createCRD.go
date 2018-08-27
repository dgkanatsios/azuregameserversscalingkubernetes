package helpers

import (
	"github.com/dgkanatsios/azuregameserversscalingkubernetes/shared"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
)

func CreateDedicatedGameServerCRD(dgsName string, podSpec corev1.PodSpec) (finalDDGSName string, err error) {

	if dgsName == "" {
		dgsName = shared.GenerateRandomName("gameserver")
	}

	log.Printf("Creating DedicatedGameServer %s", dgsName)

	dgs := shared.NewDedicatedGameServerWithNoParent(shared.GameNamespace, dgsName, podSpec)

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

func CreateDedicatedGameServerCollectionCRD(dgsColName string, replicas int32, podSpec corev1.PodSpec) (finalDGSColName string, err error) {

	if dgsColName == "" {
		dgsColName = shared.GenerateRandomName("dedicatedgameservercollection")
	}

	log.Printf("Creating DedicatedGameServerCollection %s", dgsColName)

	dgsCol := shared.NewDedicatedGameServerCollection(dgsColName, shared.GameNamespace, replicas, podSpec)

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
