package helpers

import (
	"fmt"
)

var SetSessionsURL string

// ServerSessions is a struct that represents active sessions (connected players) per pod
type ServerSessions struct {
	Name           string `json:"name"`
	ActiveSessions int    `json:"activeSessions"`
}

func InitializeSetSessionsURL() {

	SetSessionsURL = "http://docker-openarena-k8s-api/setsessions?code=" + getAccessCode()

	fmt.Println("Initializes setSessionsURL:", SetSessionsURL)
}
