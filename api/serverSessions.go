package main

import (
	"fmt"
	"log"
	"strconv"

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ServerSessions is a struct that represents active sessions (connected players) per pod
type ServerSessions struct {
	Name           string `json:"name"`
	ActiveSessions int    `json:"activeSessions"`
}

func initializeSetSessionsURL() {

	endpoint, err := endpointsClient.Get("docker-openarena-k8s-api", meta_v1.GetOptions{})
	if err != nil {
		log.Fatal("Cannot initialize setSessionsURL due to", err.Error())
	}

	ip := endpoint.Subsets[0].Addresses[0].IP + ":" + strconv.Itoa(int(endpoint.Subsets[0].Ports[0].Port))

	setSessionsURL = "http://" + ip + "/setsessions?code=" + getAccessCode()

	fmt.Println("Initializes setSessionsURL:", setSessionsURL)
}
