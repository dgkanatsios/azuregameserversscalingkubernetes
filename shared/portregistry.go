package shared

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	dgsv1alpha1 "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/apis/azuregaming/v1alpha1"
	dgsclientset "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/clientset/versioned"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var PortRegistry *dgsv1alpha1.PortRegistry
var clientset *dgsclientset.Clientset
var mutex = &sync.Mutex{}

func InitializePortRegistry(dgsclientset *dgsclientset.Clientset) error {
	clientset = dgsclientset
	var err error
	PortRegistry, err = clientset.AzuregamingV1alpha1().PortRegistries(GameNamespace).Get(PortRegistryName, metav1.GetOptions{})
	if err != nil {
		log.Error("Error getting PortRegistry CRD: %v", err)
	}

	if PortRegistry.Spec.Ports == nil {
		PortRegistry.Spec.GameServerPorts = make(map[string]string)
		PortRegistry.Spec.Ports = make(map[int32]bool)
		//first time, so initialize it
		for i := MinPort; i <= MaxPort; i++ {
			PortRegistry.Spec.Ports[int32(i)] = false
		}
		_, err := clientset.AzuregamingV1alpha1().PortRegistries(GameNamespace).Update(PortRegistry)

		if err != nil {
			log.Error("Error initializing PortRegistry CRD: %v", err)
		}
	}

	return nil

}

func DeregisterServerPorts(serverName string) error {
	if PortRegistry == nil {
		log.Panic("PortRegistry is not initialized")
	}

	mutex.Lock()
	defer mutex.Unlock()

	ports := strings.Split(PortRegistry.Spec.GameServerPorts[serverName], ",")

	var deleteErrors string

	for _, port := range ports {
		if port != "" {
			portInt, errconvert := strconv.Atoi(port)
			if errconvert != nil {
				deleteErrors = fmt.Sprintf("%s,%s", deleteErrors, errconvert.Error())
			}

			PortRegistry.Spec.Ports[int32(portInt)] = false
		}
	}

	delete(PortRegistry.Spec.GameServerPorts, serverName)

	err := updatePortRegistry()

	if err != nil {
		log.Error("Error saving PortRegistry CRD during port deregister: %v", err)
		return err
	}

	if deleteErrors != "" {
		return errors.New(fmt.Sprintf("Errors encountered during port de-registration%s", deleteErrors))
	}

	return nil
}

func GetNewPort(serverName string) (int32, error) {

	if PortRegistry == nil {
		log.Panic("PortRegistry is not initialized")
	}

	var port int32
	mutex.Lock()
	defer mutex.Unlock()

	//get a random port
	port = int32(GetRandomInt(MinPort, MaxPort))
	for {
		val, _ := PortRegistry.Spec.Ports[port]

		if !val { // port is not taken
			PortRegistry.Spec.Ports[port] = true
			PortRegistry.Spec.GameServerPorts[serverName] += fmt.Sprintf("%d,", port)
			break
		}
		port = int32(GetRandomInt(MinPort, MaxPort))
	}

	err := updatePortRegistry()

	if err != nil {
		log.Errorf("Error saving PortRegistry CRD during port registration: %v", err)
		return 0, err
	}

	return port, nil
}

func updatePortRegistry() error {
	tempPortRegistry, err := clientset.AzuregamingV1alpha1().PortRegistries(GameNamespace).Get(PortRegistryName, metav1.GetOptions{})

	if err != nil {
		log.Errorf("Error getting an updated Port Registry:%v", err)
		return err
	}

	tempPortRegistry.Spec.Ports = PortRegistry.Spec.Ports
	tempPortRegistry.Spec.GameServerPorts = PortRegistry.Spec.GameServerPorts

	//TODO: deep copy?

	_, err = clientset.AzuregamingV1alpha1().PortRegistries(GameNamespace).Update(tempPortRegistry)
	if err != nil {
		log.Errorf("Error saving an updated Port Registry:%v", err)
		return err
	}
	return nil
}
