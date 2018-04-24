package main

import (
	"fmt"
)

// ServerSessions is a struct that represents active sessions (connected players) per pod
type ServerSessions struct {
	Name           string `json:"name"`
	ActiveSessions int    `json:"activeSessions"`
}

func initializeSetSessionsURL() {

	setSessionsURL = "http://docker-openarena-k8s-api/setsessions?code=" + getAccessCode()

	fmt.Println("Initializes setSessionsURL:", setSessionsURL)
}
