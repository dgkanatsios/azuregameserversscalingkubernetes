package controller

import (
	"fmt"
	"log"
	"strings"

	listercorev1 "k8s.io/client-go/listers/core/v1"

	shared "github.com/dgkanatsios/azuregameserversscalingkubernetes/shared"
	dgs_v1 "github.com/dgkanatsios/azuregameserversscalingkubernetes/shared/pkg/apis/dedicatedgameserver/v1"
	apiv1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	dgsclientset "github.com/dgkanatsios/azuregameserversscalingkubernetes/shared/pkg/client/clientset/versioned"
	"k8s.io/client-go/kubernetes"
)

func handleDedicatedGameServerAdd(obj interface{}, podLister listercorev1.PodLister, client *kubernetes.Clientset) {
	log.Print("DedicatedGameServer added " + getDGS(obj).Name)

	dgs := obj.(*dgs_v1.DedicatedGameServer)
	name := dgs.ObjectMeta.Name
	if !isOpenArena(name) {
		return
	}

	podFound, err := podLister.Pods(namespace).Get(name)
	if err != nil || podFound == nil {
		log.Printf("Pod with name %s already exists. Will not create a new one", name)
		return
	}

	pod := shared.NewPod(name, *dgs.Spec.Port, "sessionsurl", dgs.Spec.StartMap, dgs.Spec.Image)
	podsClient := client.Core().Pods(namespace)
	result, err := podsClient.Create(pod)
	if err != nil {
		log.Fatal("Oops. Error in creating pod: " + err.Error())
	}
	log.Print("Asked K8s to create a pod for the new DGS with name " + result.GetObjectMeta().GetName())
}

func handleDedicatedGameServerUpdate(oldObj interface{}, newObj interface{}) {
	log.Print("DedicatedGameServer updated " + getDGS(newObj).Name)
}

func handleDedicatedGameServerDelete(obj interface{}, client *kubernetes.Clientset) {
	log.Print("DedicatedGameServer deleted " + getDGS(obj).Name)
	dgs := obj.(*dgs_v1.DedicatedGameServer)
	name := dgs.ObjectMeta.Name
	if !isOpenArena(name) {
		return
	}

	podsClient := client.Core().Pods(namespace)
	err := podsClient.Delete(name, nil)
	if err != nil {
		log.Fatal("Oops. Error in deleting pod: " + err.Error())
	}
	log.Print("Asked K8s to delete a pod for the deleted DGS with name " + name)
}

func handlePodAdd(obj interface{}, client *kubernetes.Clientset, dgsclient *dgsclientset.Clientset) {

	pod := getPod(obj)
	name := pod.Name

	if !isOpenArena(name) {
		return
	}

	log.Print("pod added " + name)

	//add the node public IP to the DGS details
	dgs, err := dgsclient.Azure().DedicatedGameServers(namespace).Get(name, meta_v1.GetOptions{})

	if err != nil {
		log.Print("Cannot get DedicatedGameServer due to: " + err.Error())
		log.Print("Seems like an orphaned Pod (i.e. Pod without DedicatedGameServer) exists. System unhealthy")
		return
	}
	log.Print("nodename:" + pod.Spec.NodeName)
	dgs.Spec.PublicIP = getNodeExternalIP(pod.Spec.NodeName, client)

	result, err := dgsclient.Azure().DedicatedGameServers(namespace).Update(dgs)

	if err != nil {
		log.Fatal("Cannot update DGS due to: " + err.Error())
	}

	log.Print("Updated DGS with PublicIP " + result.Spec.PublicIP)
}

func handlePodUpdate(oldObj interface{}, newObj interface{}) {
	log.Print("pod updated " + getPod(newObj).Name)
}

func handlePodDelete(obj interface{}) {
	log.Print("pod deleted " + getPod(obj).Name)
}

func getPod(obj interface{}) *apiv1.Pod {
	pod := obj.(*apiv1.Pod)
	return pod
}
func getDGS(obj interface{}) *dgs_v1.DedicatedGameServer {
	dgs := obj.(*dgs_v1.DedicatedGameServer)
	return dgs
}

func isOpenArena(name string) bool {
	return strings.HasPrefix(name, "openarena")
}

func getNodeExternalIP(nodename string, client *kubernetes.Clientset) string {

	node, err := client.CoreV1().Nodes().Get(nodename, meta_v1.GetOptions{})
	if err != nil {
		fmt.Printf("Error in getting IP for node %s because of error %s", nodename, err.Error())
		return ""
	}
	for _, address := range node.Status.Addresses {
		if address.Type == apiv1.NodeExternalIP {
			return address.Address
		}
	}
	return ""
}
