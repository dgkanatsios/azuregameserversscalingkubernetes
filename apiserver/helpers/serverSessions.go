package helpers

import log "github.com/sirupsen/logrus"

var SetSessionsURL string

// ServerSessions is a struct that represents active sessions (connected players) per pod
type ServerSessions struct {
	Name           string `json:"name"`
	ActiveSessions int    `json:"activeSessions"`
}

func InitializeSetSessionsURL() {

	SetSessionsURL = "http://docker-openarena-k8s-apiserver/setsessions?code=" + getAccessCode()

	log.Println("Initialized setSessionsURL:", SetSessionsURL)
}
