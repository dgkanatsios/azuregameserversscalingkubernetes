package shared

import log "github.com/sirupsen/logrus"

var setServerStatusURL string

func initializeSetServerStatusURL() {

	setServerStatusURL = setServerStatusURLPrefix + getAccessCode()

	log.Println("Initialized SetServerStatusURL:", setServerStatusURL)
}

func GetServerStatusSetURL() string {
	if setServerStatusURL == "" {
		initializeSetServerStatusURL()
	}
	return setServerStatusURL
}
