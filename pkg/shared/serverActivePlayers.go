package shared

import log "github.com/sirupsen/logrus"

var setActivePlayersURL string

func initializeSetActivePlayersURL() error {

	code, err := getAccessCode()
	if err != nil {
		return err
	}

	setActivePlayersURL = setActivePlayersURLPrefix + code

	log.Println("Initialized setActivePlayersURL:", setActivePlayersURL)

	return nil
}

func GetActivePlayersSetURL() string {
	if setActivePlayersURL == "" {
		initializeSetActivePlayersURL()
	}
	return setActivePlayersURL
}
