package shared

import log "github.com/sirupsen/logrus"

var setActivePlayersURL string

// ServerActivePlayers is a struct that represents active sessions (connected players) per pod
type ServerActivePlayers struct {
	ServerName   string `json:"serverName"`
	PodNamespace string `json:"podNamespace"`
	PlayerCount  int    `json:"playerCount"`
}

func initializeSetActivePlayersURL() {

	setActivePlayersURL = setActivePlayersURLPrefix + getAccessCode()

	log.Println("Initialized setActivePlayersURL:", setActivePlayersURL)
}

func GetActivePlayersSetURL() string {
	if setActivePlayersURL == "" {
		initializeSetActivePlayersURL()
	}
	return setActivePlayersURL
}
