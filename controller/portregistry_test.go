package controller

import (
	"fmt"
	"testing"
)

func TestPortRegistry(t *testing.T) {
	portRegistry := NewIndexedDictionary(20000, 20010)
	lala := []int32{20002, 20004, 20006, 20008}
	portRegistry.AssignRegisteredPorts(lala, "server1")

	portRegistry.displayRegistry()

	portRegistry.AssignUnregisteredPorts()

	portRegistry.displayRegistry()

	port1, _ := portRegistry.GetNewPort("server1")
	port1, _ = portRegistry.GetNewPort("server1")
	port1, _ = portRegistry.GetNewPort("server1")
	port1, _ = portRegistry.GetNewPort("server1")
	port1, _ = portRegistry.GetNewPort("server2")
	port1, _ = portRegistry.GetNewPort("server2")
	port1, _ = portRegistry.GetNewPort("server2")

	fmt.Print(port1)

	portRegistry.displayRegistry()

	_, err := portRegistry.GetNewPort("server2")

	if err == nil {
		fmt.Printf("err should be nil...\n")
		return
	}
	fmt.Println("correctly returned error")

	portRegistry.displayRegistry()

	portRegistry.DeregisterServerPorts("server2")

	portRegistry.displayRegistry()

	port1, _ = portRegistry.GetNewPort("server1")

	portRegistry.displayRegistry()

}
