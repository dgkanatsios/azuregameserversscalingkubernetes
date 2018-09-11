package controller

import (
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/clientset/versioned/fake"
	dgsinformers "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/informers/externalversions"
	"github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/shared"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const displayVariablesDuringTesting = false

func TestPortRegistry(t *testing.T) {

	namegenerator := shared.NewFakeRandomNameGenerator()
	namegenerator.SetName("server1")
	server1 := shared.NewDedicatedGameServerWithNoParent(shared.GameNamespace, "default", corev1.PodSpec{
		Containers: []corev1.Container{
			corev1.Container{
				Name: "test",
				Ports: []corev1.ContainerPort{
					corev1.ContainerPort{
						HostPort: 20002,
					},
					corev1.ContainerPort{
						HostPort: 20004,
					},
					corev1.ContainerPort{
						HostPort: 20006,
					},
					corev1.ContainerPort{
						HostPort: 20008,
					},
				},
			},
		},
	}, namegenerator)

	namegenerator.SetName("server2")
	server2 := shared.NewDedicatedGameServerWithNoParent(shared.GameNamespace, "default", corev1.PodSpec{
		Containers: []corev1.Container{
			corev1.Container{
				Name: "test",
			},
		},
	}, namegenerator)

	dgsObjects := []runtime.Object{}
	dgsObjects = append(dgsObjects, server1)
	dgsObjects = append(dgsObjects, server2)
	dgsClient := fake.NewSimpleClientset(dgsObjects...)

	dgsInformers := dgsinformers.NewSharedInformerFactory(dgsClient, noResyncPeriodFunc())
	dgsInformers.Azuregaming().V1alpha1().DedicatedGameServers().Informer().GetIndexer().Add(server1)
	dgsInformers.Azuregaming().V1alpha1().DedicatedGameServers().Informer().GetIndexer().Add(server2)

	portRegistry, err := NewPortRegistry(dgsClient, 20000, 20010, shared.GameNamespace)

	if err != nil {
		t.Fatalf("Cannot initialize PortRegistry due to: %s", err.Error())
	}

	registeredPortsServer1 := []int32{20002, 20004, 20006, 20008}
	registeredPortsServer2 := []int32{}

	if displayVariablesDuringTesting {
		portRegistry.displayRegistry()
	}

	verifyGameServerPortsExist(portRegistry, "server1", registeredPortsServer1, t)

	verifyUnregisteredPorts(portRegistry, t)

	if displayVariablesDuringTesting {
		portRegistry.displayRegistry()
	}

	go portRegistry.portProducer()
	// end of initialization

	peekPort, err := peekNextPort(portRegistry)
	actualPort, _ := portRegistry.GetNewPort("server1")
	registeredPortsServer1 = append(registeredPortsServer1, actualPort)
	if actualPort != peekPort {
		t.Errorf("Wrong port returned, peekPort:%d,actualPort:%d", peekPort, actualPort)
	}
	if err != nil {
		t.Errorf("Error should be nil")
		t.Errorf("Error:%s", err.Error())
	}

	peekPort, err = peekNextPort(portRegistry)
	actualPort, _ = portRegistry.GetNewPort("server1")
	registeredPortsServer1 = append(registeredPortsServer1, actualPort)
	if actualPort != peekPort {
		t.Errorf("Wrong port returned, peekPort:%d,actualPort:%d", peekPort, actualPort)
	}
	if err != nil {
		t.Errorf("Error should be nil")
		t.Errorf("Error:%s", err.Error())
	}

	peekPort, err = peekNextPort(portRegistry)
	actualPort, _ = portRegistry.GetNewPort("server1")
	registeredPortsServer1 = append(registeredPortsServer1, actualPort)
	if actualPort != peekPort {
		t.Errorf("Wrong port returned, peekPort:%d,actualPort:%d", peekPort, actualPort)
	}
	if err != nil {
		t.Errorf("Error should be nil")
		t.Errorf("Error:%s", err.Error())
	}

	peekPort, err = peekNextPort(portRegistry)
	actualPort, _ = portRegistry.GetNewPort("server1")
	registeredPortsServer1 = append(registeredPortsServer1, actualPort)
	if actualPort != peekPort {
		t.Errorf("Wrong port returned, peekPort:%d,actualPort:%d", peekPort, actualPort)
	}
	if err != nil {
		t.Errorf("Error should be nil")
		t.Errorf("Error:%s", err.Error())
	}

	peekPort, err = peekNextPort(portRegistry)
	actualPort, _ = portRegistry.GetNewPort("server2")
	registeredPortsServer2 = append(registeredPortsServer2, actualPort)
	if actualPort != peekPort {
		t.Errorf("Wrong port returned, peekPort:%d,actualPort:%d", peekPort, actualPort)
	}
	if err != nil {
		t.Errorf("Error should be nil")
		t.Errorf("Error:%s", err.Error())
	}

	peekPort, err = peekNextPort(portRegistry)
	actualPort, _ = portRegistry.GetNewPort("server2")
	registeredPortsServer2 = append(registeredPortsServer2, actualPort)
	if actualPort != peekPort {
		t.Errorf("Wrong port returned, peekPort:%d,actualPort:%d", peekPort, actualPort)
	}
	if err != nil {
		t.Errorf("Error should be nil")
		t.Errorf("Error:%s", err.Error())
	}

	peekPort, err = peekNextPort(portRegistry)
	actualPort, _ = portRegistry.GetNewPort("server2")
	registeredPortsServer2 = append(registeredPortsServer2, actualPort)
	if actualPort != peekPort {
		t.Errorf("Wrong port returned, peekPort:%d,actualPort:%d", peekPort, actualPort)
	}
	if err != nil {
		t.Errorf("Error should be nil")
		t.Errorf("Error:%s", err.Error())
	}

	verifyGameServerPortsExist(portRegistry, "server1", registeredPortsServer1, t)
	verifyGameServerPortsExist(portRegistry, "server2", registeredPortsServer2, t)

	if displayVariablesDuringTesting {
		portRegistry.displayRegistry()
	}

	_, err = peekNextPort(portRegistry)
	if err == nil {
		t.Errorf("Error should not be nil")
		t.Errorf("Error:%s", err.Error())
	}

	_, err = portRegistry.GetNewPort("server2")
	if err == nil {
		t.Error("Should return an error")
	}

	if displayVariablesDuringTesting {
		portRegistry.displayRegistry()
	}

	portRegistry.DeregisterServerPorts("server2")

	verifyGameServerPortsDoNotExist(portRegistry, "server2", registeredPortsServer2, t)

	if displayVariablesDuringTesting {
		portRegistry.displayRegistry()
	}

	peekPort, err = peekNextPort(portRegistry)
	actualPort, _ = portRegistry.GetNewPort("server1")
	if actualPort != peekPort {
		t.Errorf("Wrong port returned, peekPort:%d,actualPort:%d", peekPort, actualPort)
	}
	if err != nil {
		t.Errorf("Error should be nil")
		t.Errorf("Error:%s", err.Error())
	}

	registeredPortsServer1 = append(registeredPortsServer1, actualPort)
	verifyGameServerPortsExist(portRegistry, "server1", registeredPortsServer1, t)

	if displayVariablesDuringTesting {
		portRegistry.displayRegistry()
	}

	portRegistry.StopPortRegistry()

}

func verifyGameServerPortsExist(portRegistry *PortRegistry, serverName string, ports []int32, t *testing.T) {
	val, ok := portRegistry.GameServerPorts[serverName]
	if !ok {
		t.Errorf("Should contain gameserver %s", serverName)
	}

	for _, port := range ports {
		if !strings.Contains(val, strconv.Itoa(int(port))) {
			t.Errorf("Should contain gameserverport %d", port)
		}

		valB, ok := portRegistry.Ports[port]

		if !ok {
			t.Errorf("There should be an entry about port %d", port)
		}

		if !valB {
			t.Errorf("Port %d should be registered", port)
		}

	}

}

func verifyUnregisteredPorts(portRegistry *PortRegistry, t *testing.T) {
	if len(portRegistry.Ports) != int(portRegistry.Max-portRegistry.Min+1) {
		t.Errorf("Wrong length on portregistry ports")
	}
}

func verifyGameServerPortsDoNotExist(portRegistry *PortRegistry, serverName string, ports []int32, t *testing.T) {
	_, ok := portRegistry.GameServerPorts[serverName]
	if ok {
		t.Errorf("Should not contain gameserver %s", serverName)
	}

	for _, port := range ports {
		valB, ok := portRegistry.Ports[port]
		if !ok {
			t.Errorf("There should be an entry about port %d", port)
		}

		if valB {
			t.Errorf("Port %d should not be registered", port)
		}
	}
}

func peekNextPort(pr *PortRegistry) (int32, error) {
	tempPointer := pr.NextFreePortIndex
	tempPointerCopy := tempPointer
	for {
		if !pr.Ports[pr.Indexes[tempPointer]] { //port not taken
			return pr.Indexes[tempPointer], nil
		}

		tempPointer++
		if tempPointer == (pr.Max - pr.Min + 1) {
			tempPointer = 0
		}

		if tempPointer == tempPointerCopy {
			//we've done a full circle, no ports available
			return 0, errors.New("No ports available")
		}
	}
}
