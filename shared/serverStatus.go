package shared

import log "github.com/sirupsen/logrus"

var setServerStatusURL string

type ServerStatus struct {
	ServerName   string `json:"serverName"`
	PodNamespace string `json:"podNamespace"`
	Status       string `json:"status"`
}

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
