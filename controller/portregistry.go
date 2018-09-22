package controller

import (
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"

	dgsclientset "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/clientset/versioned"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PortRegistry implements a custom map for the port registry
type PortRegistry struct {
	Ports             map[int32]bool
	GameServerPorts   map[string]string
	Indexes           []int32
	NextFreePortIndex int32
	Min               int32
	Max               int32
	portRequests      chan string
	portResponses     chan int32
}

// NewPortRegistry initializes the IndexedDictionary that holds the port registry.
func NewPortRegistry(dgsclientset dgsclientset.Interface, min, max int32, namespace string) (*PortRegistry, error) {

	pr := &PortRegistry{
		Ports:           make(map[int32]bool, max-min+1),
		GameServerPorts: make(map[string]string, max-min+1),
		Indexes:         make([]int32, max-min+1),
		Min:             min,
		Max:             max,
		portRequests:    make(chan string, 100),
		portResponses:   make(chan int32, 100),
	}

	dgsList, err := dgsclientset.AzuregamingV1alpha1().DedicatedGameServers(namespace).List(metav1.ListOptions{})
	if err != nil {
		log.Errorf("Error getting Dedicated Game Servers List: %v", err)
		return nil, err
	}

	if len(dgsList.Items) > 0 {
		for _, dgs := range dgsList.Items {

			if len(dgs.Spec.Template.Containers) == 0 {
				log.Errorf("DGS with name %s has a problem in its Pod Template: %#v", dgs.Name, dgs)
				continue
			}

			for j := 0; j < len(dgs.Spec.Template.Containers); j++ {
				ports := make([]int32, len(dgs.Spec.Template.Containers[j].Ports))
				for i, portInfo := range dgs.Spec.Template.Containers[j].Ports {
					if portInfo.HostPort == 0 {
						log.Errorf("HostPort for DGS %s and ContainerPort %d is zero, ignoring", dgs.Name, portInfo.ContainerPort)
						continue
					}
					ports[i] = portInfo.HostPort
				}
				pr.assignRegisteredPorts(ports, dgs.Name)
			}

		}
	}

	pr.assignUnregisteredPorts()

	go pr.portProducer()

	return pr, nil

}

func (pr *PortRegistry) initializePortRegistry() {

}

func (pr *PortRegistry) displayRegistry() {
	fmt.Printf("-------------------------------------\n")
	fmt.Printf("Ports: %v\n", pr.Ports)
	fmt.Printf("GameServerPorts: %s\n", pr.GameServerPorts)
	fmt.Printf("Indexes: %v\n", pr.Indexes)
	fmt.Printf("NextIndex: %d\n", pr.NextFreePortIndex)
	fmt.Printf("-------------------------------------\n")
}

// GetNewPort returns and registers a new port for the designated game server. Locks a mutex
func (pr *PortRegistry) GetNewPort(serverName string) (int32, error) {
	pr.portRequests <- serverName

	port := <-pr.portResponses

	if port == -1 {
		return -1, errors.New("Cannot register a new port. No more slots")
	}

	return port, nil
}

func (pr *PortRegistry) portProducer() {
	for serverName := range pr.portRequests { //wait till a new request comes

		initialIndex := pr.NextFreePortIndex
		for {
			if pr.Ports[pr.Indexes[pr.NextFreePortIndex]] == false {
				//we found a port
				port := pr.Indexes[pr.NextFreePortIndex]
				pr.Ports[port] = true
				pr.GameServerPorts[serverName] = fmt.Sprintf("%d,%s", port, pr.GameServerPorts[serverName])
				pr.increaseNextFreePortIndex()

				//port is set
				pr.portResponses <- port
				break
			}

			pr.increaseNextFreePortIndex()

			if initialIndex == pr.NextFreePortIndex {
				//we did a full loop - no empty ports
				pr.portResponses <- -1
				break
			}
		}
	}
}

// Stop stops port registry mechanism by closing requests and responses channels
func (pr *PortRegistry) Stop() {
	close(pr.portRequests)
	close(pr.portResponses)
}

// DeregisterServerPorts deregisters all ports for the designated server. Locks a mutex
func (pr *PortRegistry) DeregisterServerPorts(serverName string) {

	ports := strings.Split(pr.GameServerPorts[serverName], ",")

	var deleteErrors string

	for _, port := range ports {
		if port != "" {
			portInt, errconvert := strconv.Atoi(port)
			if errconvert != nil {
				deleteErrors = fmt.Sprintf("%s,%s", deleteErrors, errconvert.Error())
			}

			pr.Ports[int32(portInt)] = false
		}
	}

	delete(pr.GameServerPorts, serverName)

}

func (pr *PortRegistry) assignRegisteredPorts(ports []int32, serverName string) {

	var portsString string
	for i := 0; i < len(ports); i++ {
		pr.Ports[ports[i]] = true
		pr.Indexes[i] = ports[i]
		pr.increaseNextFreePortIndex()

		portsString = fmt.Sprintf("%d,%s", ports[i], portsString)

	}
	pr.GameServerPorts[serverName] = portsString

}

func (pr *PortRegistry) assignUnregisteredPorts() {
	i := pr.NextFreePortIndex
	for _, port := range pr.getPermutatedPorts() {
		if _, ok := pr.Ports[port]; !ok {
			pr.Ports[port] = false
			pr.Indexes[i] = port
			i++
		}
	}
}

func (pr *PortRegistry) increaseNextFreePortIndex() {
	pr.NextFreePortIndex++
	//reset the index if needed
	if pr.NextFreePortIndex == pr.Max-pr.Min+1 {
		pr.NextFreePortIndex = 0
	}

}

func (pr *PortRegistry) getPermutatedPorts() []int32 {
	ports := make([]int32, pr.Max-pr.Min+1)
	for i := 0; i < len(ports); i++ {
		ports[i] = int32(pr.Min)
	}
	perm := rand.Perm(int((pr.Max - pr.Min + 1)))

	for i := 0; i < len(ports); i++ {
		ports[i] += int32(perm[i])
	}
	return ports
}
