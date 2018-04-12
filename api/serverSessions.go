package main

import (
	"fmt"
	"log"

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ServerSessions is a struct that represents active sessions (connected players) per pod
type ServerSessions struct {
	Name           string `json:"name"`
	ActiveSessions int    `json:"activeSessions"`
}

func initializeSetSessionsURL() {
	service, err := servicesClient.Get("docker-openarena-k8s-api", meta_v1.GetOptions{})
	if err != nil {
		log.Fatal("Cannot initialize setSessionsURL due to", err.Error())
	}
	ip := service.Spec.ClusterIP + ":" + string(service.Spec.Ports[0].NodePort)

	setSessionsURL = "http://" + ip + "/setsessions?code=" + getAccessCode()

	fmt.Println("Initializes setSessionsURL:", setSessionsURL)
}
