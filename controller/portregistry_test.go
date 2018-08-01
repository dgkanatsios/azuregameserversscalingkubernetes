package controller

import (
	"errors"
	"strconv"
	"strings"
	"testing"
)

func TestPortRegistry(t *testing.T) {

	portRegistryTest := NewIndexedDictionary(20000, 20010)

	registeredPortsServer1 := []int32{20002, 20004, 20006, 20008}
	registeredPortsServer2 := []int32{}

	portRegistryTest.AssignRegisteredPorts(registeredPortsServer1, "server1")

	verifyGameServerPortsExist(portRegistryTest, "server1", registeredPortsServer1, t)

	portRegistryTest.displayRegistry()

	portRegistryTest.AssignUnregisteredPorts()

	verifyUnregisteredPorts(portRegistryTest, t)

	portRegistryTest.displayRegistry()

	peekPort, err := peekNextPort(portRegistryTest)
	actualPort, _ := portRegistryTest.GetNewPort("server1")
	registeredPortsServer1 = append(registeredPortsServer1, actualPort)
	if actualPort != peekPort {
		t.Errorf("Wrong port returned")
	}
	if err != nil {
		t.Errorf("Error should be nil")
	}

	peekPort, err = peekNextPort(portRegistryTest)
	actualPort, _ = portRegistryTest.GetNewPort("server1")
	registeredPortsServer1 = append(registeredPortsServer1, actualPort)
	if actualPort != peekPort {
		t.Errorf("Wrong port returned")
	}
	if err != nil {
		t.Errorf("Error should be nil")
	}

	peekPort, err = peekNextPort(portRegistryTest)
	actualPort, _ = portRegistryTest.GetNewPort("server1")
	registeredPortsServer1 = append(registeredPortsServer1, actualPort)
	if actualPort != peekPort {
		t.Errorf("Wrong port returned")
	}
	if err != nil {
		t.Errorf("Error should be nil")
	}

	peekPort, err = peekNextPort(portRegistryTest)
	actualPort, _ = portRegistryTest.GetNewPort("server1")
	registeredPortsServer1 = append(registeredPortsServer1, actualPort)
	if actualPort != peekPort {
		t.Errorf("Wrong port returned")
	}
	if err != nil {
		t.Errorf("Error should be nil")
	}

	peekPort, err = peekNextPort(portRegistryTest)
	actualPort, _ = portRegistryTest.GetNewPort("server2")
	registeredPortsServer2 = append(registeredPortsServer2, actualPort)
	if actualPort != peekPort {
		t.Errorf("Wrong port returned")
	}
	if err != nil {
		t.Errorf("Error should be nil")
	}

	peekPort, err = peekNextPort(portRegistryTest)
	actualPort, _ = portRegistryTest.GetNewPort("server2")
	registeredPortsServer2 = append(registeredPortsServer2, actualPort)
	if actualPort != peekPort {
		t.Errorf("Wrong port returned")
	}
	if err != nil {
		t.Errorf("Error should be nil")
	}

	peekPort, err = peekNextPort(portRegistryTest)
	actualPort, _ = portRegistryTest.GetNewPort("server2")
	registeredPortsServer2 = append(registeredPortsServer2, actualPort)
	if actualPort != peekPort {
		t.Errorf("Wrong port returned")
	}
	if err != nil {
		t.Errorf("Error should be nil")
	}

	verifyGameServerPortsExist(portRegistryTest, "server1", registeredPortsServer1, t)
	verifyGameServerPortsExist(portRegistryTest, "server2", registeredPortsServer2, t)

	portRegistryTest.displayRegistry()

	_, err = peekNextPort(portRegistryTest)
	if err == nil {
		t.Errorf("Error should not be nil")
	}

	_, err = portRegistryTest.GetNewPort("server2")
	if err == nil {
		t.Error("Should return an error")
	}

	portRegistryTest.displayRegistry()

	portRegistryTest.DeregisterServerPorts("server2")

	verifyGameServerPortsDoNotExist(portRegistryTest, "server2", registeredPortsServer2, t)

	portRegistryTest.displayRegistry()

	peekPort, err = peekNextPort(portRegistryTest)
	actualPort, _ = portRegistryTest.GetNewPort("server1")
	if actualPort != peekPort {
		t.Errorf("Wrong port returned")
	}
	if err != nil {
		t.Errorf("Error should be nil")
	}

	registeredPortsServer1 = append(registeredPortsServer1, actualPort)
	verifyGameServerPortsExist(portRegistryTest, "server1", registeredPortsServer1, t)

	portRegistryTest.displayRegistry()

}

func verifyGameServerPortsExist(portRegistry *IndexedDictionary, serverName string, ports []int32, t *testing.T) {
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

func verifyUnregisteredPorts(portRegistry *IndexedDictionary, t *testing.T) {
	if len(portRegistry.Ports) != int(portRegistry.Max-portRegistry.Min+1) {
		t.Errorf("Wrong length on portregistry ports")
	}
}

func verifyGameServerPortsDoNotExist(portRegistry *IndexedDictionary, serverName string, ports []int32, t *testing.T) {
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

func peekNextPort(portRegistry *IndexedDictionary) (int32, error) {
	tempPointer := portRegistry.NextFreePortIndex
	tempPointerCopy := tempPointer
	for {
		if !portRegistry.Ports[portRegistry.Indexes[tempPointer]] { //port not taken
			return portRegistry.Indexes[tempPointer], nil
		}

		tempPointer++
		if tempPointer == (portRegistry.Max - portRegistry.Min + 1) {
			tempPointer = 0
		}

		if tempPointer == tempPointerCopy {
			//we've done a full circle, no ports available
			return 0, errors.New("No ports available")
		}
	}
}
