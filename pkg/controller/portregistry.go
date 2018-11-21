package controllers

import (
	"errors"
	"fmt"
	"math/rand"

	dgsclientset "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/clientset/versioned"
	"github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/shared"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PortRegistry implements a custom map for the port registry
type PortRegistry struct {
	Ports             map[int32]bool
	Indexes           []int32
	NextFreePortIndex int32
	Min               int32         // Minimum Port
	Max               int32         // Maximum Port
	portRequests      chan struct{} // buffered channel to store port requests
	portResponses     chan int32    // buffered channel to store port responses (system returns the HostPorts)
}

// NewPortRegistry initializes the IndexedDictionary that holds the port registry.
func NewPortRegistry(dgsclientset dgsclientset.Interface, min, max int32, namespace string) (*PortRegistry, error) {

	pr := &PortRegistry{
		Ports:         make(map[int32]bool, max-min+1),
		Indexes:       make([]int32, max-min+1),
		Min:           min,
		Max:           max,
		portRequests:  make(chan struct{}, 100),
		portResponses: make(chan int32, 100),
	}

	dgsList, err := dgsclientset.AzuregamingV1alpha1().DedicatedGameServers(namespace).List(metav1.ListOptions{})
	if err != nil {
		log.Errorf("Error getting Dedicated Game Servers List: %v", err)
		return nil, err
	}

	// gather ports for existing DGS
	if len(dgsList.Items) > 0 {
		for _, dgs := range dgsList.Items {
			if len(dgs.Spec.Template.Containers) == 0 {
				log.Errorf("DGS with name %s has no containers in its Pod Template: %#v", dgs.Name, dgs)
				continue
			}
			if len(dgs.Spec.PortsToExpose) == 0 {
				continue //no ports exported for this DGS
			}

			portsExposed := make([]int32, len(dgs.Spec.PortsToExpose))
			portsExposedIndex := 0
			for j := 0; j < len(dgs.Spec.Template.Containers); j++ {
				for _, portInfo := range dgs.Spec.Template.Containers[j].Ports {
					if portInfo.HostPort == 0 {
						log.Errorf("HostPort for DGS %s and ContainerPort %d is zero, ignoring", dgs.Name, portInfo.ContainerPort)
						continue
					}
					// if this port is to be exposed
					if shared.SliceContains(dgs.Spec.PortsToExpose, portInfo.ContainerPort) {
						portsExposed[portsExposedIndex] = portInfo.HostPort
						portsExposedIndex++
					}
				}
			}
			pr.assignRegisteredPorts(portsExposed)
		}
	}

	pr.assignUnregisteredPorts()

	go pr.portProducer()

	return pr, nil

}

func (pr *PortRegistry) displayRegistry() {
	fmt.Printf("-------------------------------------\n")
	fmt.Printf("Ports: %v\n", pr.Ports)
	fmt.Printf("Indexes: %v\n", pr.Indexes)
	fmt.Printf("NextIndex: %d\n", pr.NextFreePortIndex)
	fmt.Printf("-------------------------------------\n")
}

// GetNewPort returns and registers a new port for the designated game server. Locks a mutex
func (pr *PortRegistry) GetNewPort() (int32, error) {
	pr.portRequests <- struct{}{}

	port := <-pr.portResponses

	if port == -1 {
		return -1, errors.New("Cannot register a new port. No available ports")
	}

	return port, nil
}

func (pr *PortRegistry) portProducer() {
	for range pr.portRequests { //wait till a new request comes

		initialIndex := pr.NextFreePortIndex
		for {
			if pr.Ports[pr.Indexes[pr.NextFreePortIndex]] == false {
				//we found a port
				port := pr.Indexes[pr.NextFreePortIndex]
				pr.Ports[port] = true

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

// DeregisterServerPorts deregisters all ports
func (pr *PortRegistry) DeregisterServerPorts(ports []int32) {
	for i := 0; i < len(ports); i++ {
		pr.Ports[ports[i]] = false
	}
}

func (pr *PortRegistry) assignRegisteredPorts(ports []int32) {
	for i := 0; i < len(ports); i++ {
		pr.Ports[ports[i]] = true
		pr.Indexes[i] = ports[i]
		pr.increaseNextFreePortIndex()
	}
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
