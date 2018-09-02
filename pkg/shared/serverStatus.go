package shared

import log "github.com/sirupsen/logrus"

var setServerStatusURL string

func initializeSetServerStatusURL() error {

	code, err := getAccessCode()
	if err != nil {
		return err
	}

	setServerStatusURL = setServerStatusURLPrefix + code

	log.Println("Initialized SetServerStatusURL:", setServerStatusURL)

	return nil
}

func GetServerStatusSetURL() string {
	if setServerStatusURL == "" {
		initializeSetServerStatusURL()
	}
	return setServerStatusURL
}
