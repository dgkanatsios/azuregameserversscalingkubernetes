package controller

import (
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"

	"github.com/dgkanatsios/azuregameserversscalingkubernetes/shared"

	dgsclientset "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/clientset/versioned"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var portRegistry *IndexedDictionary
var clientset *dgsclientset.Clientset

// InitializePortRegistry initializes the IndexedDictionary that holds the port registry.
// Intended to be called only once. Locks a mutex
func InitializePortRegistry(dgsclientset *dgsclientset.Clientset) error {

	clientset = dgsclientset

	portRegistry = newIndexedDictionary(shared.MinPort, shared.MaxPort)

	dgsList, err := clientset.AzuregamingV1alpha1().DedicatedGameServers(shared.GameNamespace).List(metav1.ListOptions{})
	if err != nil {
		log.Errorf("Error getting Dedicated Game Servers List: %v", err)
		return err
	}

	if len(dgsList.Items) > 0 {
		for _, dgs := range dgsList.Items {

			if len(dgs.Spec.Template.Containers) == 0 {
				log.Errorf("DGS with name %s has a problem in its Pod Template: %#v", dgs.Name, dgs)
				continue
			}

			ports := make([]int32, len(dgs.Spec.Template.Containers[0].Ports))
			for i, portInfo := range dgs.Spec.Template.Containers[0].Ports {
				ports[i] = portInfo.HostPort
			}
			portRegistry.assignRegisteredPorts(ports, dgs.Name)
		}
	}

	portRegistry.assignUnregisteredPorts()

	go portRegistry.portProducer()

	return nil

}

func (id *IndexedDictionary) displayRegistry() {
	fmt.Printf("-------------------------------------\n")
	fmt.Printf("Ports: %v\n", id.Ports)
	fmt.Printf("GameServerPorts: %s\n", id.GameServerPorts)
	fmt.Printf("Indexes: %v\n", id.Indexes)
	fmt.Printf("NextIndex: %d\n", id.NextFreePortIndex)
	fmt.Printf("-------------------------------------\n")
}

// GetNewPort returns and registers a new port for the designated game server. Locks a mutex
func (id *IndexedDictionary) GetNewPort(serverName string) (int32, error) {
	portRequests <- serverName

	port := <-portResponses

	if port == -1 {
		return -1, errors.New("Cannot register a new port. No more slots")
	}

	return port, nil
}

var portRequests = make(chan string, 100)
var portResponses = make(chan int32, 100)

func (id *IndexedDictionary) portProducer() {
	for serverName := range portRequests { //wait till a new request comes

		initialIndex := id.NextFreePortIndex
		for {
			if id.Ports[id.Indexes[id.NextFreePortIndex]] == false {
				//we found a port
				port := id.Indexes[id.NextFreePortIndex]
				id.Ports[port] = true
				id.GameServerPorts[serverName] = fmt.Sprintf("%d,%s", port, id.GameServerPorts[serverName])
				id.increaseNextFreePortIndex()

				//port is set
				portResponses <- port
				break
			}

			id.increaseNextFreePortIndex()

			if initialIndex == id.NextFreePortIndex {
				//we did a full loop - no empty ports
				portResponses <- -1
				break
			}
		}
	}
}

func StopPortRegistry() {
	close(portRequests)
	close(portResponses)
}

// DeregisterServerPorts deregisters all ports for the designated server. Locks a mutex
func (id *IndexedDictionary) DeregisterServerPorts(serverName string) {

	ports := strings.Split(id.GameServerPorts[serverName], ",")

	var deleteErrors string

	for _, port := range ports {
		if port != "" {
			portInt, errconvert := strconv.Atoi(port)
			if errconvert != nil {
				deleteErrors = fmt.Sprintf("%s,%s", deleteErrors, errconvert.Error())
			}

			id.Ports[int32(portInt)] = false
		}
	}

	delete(id.GameServerPorts, serverName)

}

// IndexedDictionary implements a custom map for the port registry
type IndexedDictionary struct {
	Ports             map[int32]bool
	GameServerPorts   map[string]string
	Indexes           []int32
	NextFreePortIndex int32
	Min               int32
	Max               int32
}

// newIndexedDictionary initializes a new IndexedDictionary that will hold the port registry
// user needs to provide min and max port
func newIndexedDictionary(min, max int32) *IndexedDictionary {
	id := &IndexedDictionary{
		Ports:           make(map[int32]bool, max-min+1),
		GameServerPorts: make(map[string]string, max-min+1),
		Indexes:         make([]int32, max-min+1),
		Min:             min,
		Max:             max,
	}
	return id
}

func (id *IndexedDictionary) assignRegisteredPorts(ports []int32, serverName string) {

	var portsString string
	for i := 0; i < len(ports); i++ {
		id.Ports[ports[i]] = true
		id.Indexes[i] = ports[i]
		id.increaseNextFreePortIndex()

		portsString = fmt.Sprintf("%d,%s", ports[i], portsString)

	}
	id.GameServerPorts[serverName] = portsString

}

func (id *IndexedDictionary) assignUnregisteredPorts() {

	i := id.NextFreePortIndex
	for _, port := range id.getPermutatedPorts() {
		if _, ok := id.Ports[port]; !ok {
			id.Ports[port] = false
			id.Indexes[i] = port
			i++
		}
	}
}

func (id *IndexedDictionary) increaseNextFreePortIndex() {
	id.NextFreePortIndex++
	//reset the index if needed
	if id.NextFreePortIndex == id.Max-id.Min+1 {
		id.NextFreePortIndex = 0
	}

}

func (id *IndexedDictionary) getPermutatedPorts() []int32 {
	ports := make([]int32, id.Max-id.Min+1)
	for i := 0; i < len(ports); i++ {
		ports[i] = int32(id.Min)
	}
	perm := rand.Perm(int((id.Max - id.Min + 1)))

	for i := 0; i < len(ports); i++ {
		ports[i] += int32(perm[i])
	}
	return ports
}
