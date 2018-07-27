package shared

import log "github.com/sirupsen/logrus"

var setActivePlayersURL string

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
