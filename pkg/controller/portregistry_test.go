package controller

import (
	"errors"
	"testing"

	"github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/clientset/versioned/fake"
	dgsinformers "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/informers/externalversions"
	"github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/shared"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const displayVariablesDuringTesting = false

func TestPortRegistry(t *testing.T) {

	server1 := shared.NewDedicatedGameServerWithNoParent(shared.GameNamespace, "default1", corev1.PodSpec{
		Containers: []corev1.Container{
			corev1.Container{
				Name: "test",
				Ports: []corev1.ContainerPort{
					corev1.ContainerPort{
						ContainerPort: 20002,
						HostPort:      20002,
					},
					corev1.ContainerPort{
						ContainerPort: 20004,
						HostPort:      20004,
					},
					corev1.ContainerPort{
						ContainerPort: 20006,
						HostPort:      20006,
					},
					corev1.ContainerPort{
						ContainerPort: 20008,
						HostPort:      20008,
					},
				},
			},
		},
	}, []int32{20002, 20004, 20006, 20008})

	server2 := shared.NewDedicatedGameServerWithNoParent(shared.GameNamespace, "default2", corev1.PodSpec{
		Containers: []corev1.Container{
			corev1.Container{
				Name: "test",
			},
		},
	}, nil)

	dgsObjects := []runtime.Object{}
	dgsObjects = append(dgsObjects, server1)
	dgsObjects = append(dgsObjects, server2)
	dgsClient := fake.NewSimpleClientset(dgsObjects...)

	dgsInformers := dgsinformers.NewSharedInformerFactory(dgsClient, noResyncPeriodFunc())
	dgsInformers.Azuregaming().V1alpha1().DedicatedGameServers().Informer().GetIndexer().Add(server1)
	dgsInformers.Azuregaming().V1alpha1().DedicatedGameServers().Informer().GetIndexer().Add(server2)

	portRegistry, err := NewPortRegistry(dgsClient, 20000, 20010, metav1.NamespaceAll)

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
	actualPort, _ := portRegistry.GetNewPort()
	registeredPortsServer1 = append(registeredPortsServer1, actualPort)
	if actualPort != peekPort {
		t.Errorf("Wrong port returned, peekPort:%d,actualPort:%d", peekPort, actualPort)
	}
	if err != nil {
		t.Errorf("Error should be nil")
		t.Errorf("Error:%s", err.Error())
	}

	peekPort, err = peekNextPort(portRegistry)
	actualPort, _ = portRegistry.GetNewPort()
	registeredPortsServer1 = append(registeredPortsServer1, actualPort)
	if actualPort != peekPort {
		t.Errorf("Wrong port returned, peekPort:%d,actualPort:%d", peekPort, actualPort)
	}
	if err != nil {
		t.Errorf("Error should be nil")
		t.Errorf("Error:%s", err.Error())
	}

	peekPort, err = peekNextPort(portRegistry)
	actualPort, _ = portRegistry.GetNewPort()
	registeredPortsServer1 = append(registeredPortsServer1, actualPort)
	if actualPort != peekPort {
		t.Errorf("Wrong port returned, peekPort:%d,actualPort:%d", peekPort, actualPort)
	}
	if err != nil {
		t.Errorf("Error should be nil")
		t.Errorf("Error:%s", err.Error())
	}

	peekPort, err = peekNextPort(portRegistry)
	actualPort, _ = portRegistry.GetNewPort()
	registeredPortsServer1 = append(registeredPortsServer1, actualPort)
	if actualPort != peekPort {
		t.Errorf("Wrong port returned, peekPort:%d,actualPort:%d", peekPort, actualPort)
	}
	if err != nil {
		t.Errorf("Error should be nil")
		t.Errorf("Error:%s", err.Error())
	}

	peekPort, err = peekNextPort(portRegistry)
	actualPort, _ = portRegistry.GetNewPort()
	registeredPortsServer2 = append(registeredPortsServer2, actualPort)
	if actualPort != peekPort {
		t.Errorf("Wrong port returned, peekPort:%d,actualPort:%d", peekPort, actualPort)
	}
	if err != nil {
		t.Errorf("Error should be nil")
		t.Errorf("Error:%s", err.Error())
	}

	peekPort, err = peekNextPort(portRegistry)
	actualPort, _ = portRegistry.GetNewPort()
	registeredPortsServer2 = append(registeredPortsServer2, actualPort)
	if actualPort != peekPort {
		t.Errorf("Wrong port returned, peekPort:%d,actualPort:%d", peekPort, actualPort)
	}
	if err != nil {
		t.Errorf("Error should be nil")
		t.Errorf("Error:%s", err.Error())
	}

	peekPort, err = peekNextPort(portRegistry)
	actualPort, _ = portRegistry.GetNewPort() //for server2
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

	_, err = portRegistry.GetNewPort()
	if err == nil {
		t.Error("Should return an error")
	}

	if displayVariablesDuringTesting {
		portRegistry.displayRegistry()
	}

	portRegistry.DeregisterServerPorts(registeredPortsServer2)

	verifyGameServerPortsDoNotExist(portRegistry, "server2", registeredPortsServer2, t)

	if displayVariablesDuringTesting {
		portRegistry.displayRegistry()
	}

	peekPort, err = peekNextPort(portRegistry)
	actualPort, _ = portRegistry.GetNewPort()
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

	portRegistry.Stop()

}

func verifyGameServerPortsExist(portRegistry *PortRegistry, serverName string, ports []int32, t *testing.T) {
	for _, port := range ports {
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
