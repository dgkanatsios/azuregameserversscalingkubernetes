package helpers

import (
	"github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/shared"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
)

func CreateDedicatedGameServerCRD(dgsName string, podSpec corev1.PodSpec) (finalDDGSName string, err error) {
	namegenerator := shared.NewRealRandomNameGenerator()
	if dgsName == "" {

		dgsName = namegenerator.GenerateName("gameserver")
	}

	log.Printf("Creating DedicatedGameServer %s", dgsName)

	dgs := shared.NewDedicatedGameServerWithNoParent(shared.GameNamespace, dgsName, podSpec, namegenerator)

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
	namegenerator := shared.NewRealRandomNameGenerator()
	if dgsColName == "" {
		dgsColName = namegenerator.GenerateName("dedicatedgameservercollection")
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
