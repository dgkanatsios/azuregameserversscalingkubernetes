package shared

import "testing"

func TestInitializeSetActivePlayersURL(t *testing.T) {

	accesscode = "code123!"
	err := initializeSetActivePlayersURL()
	if err != nil {
		t.Error("Error should be nil")
	}

	if setActivePlayersURL != setActivePlayersURLPrefix+accesscode {
		t.Error("Error in setActivePlayersURL")
	}
}

func TestGetActivePlayersSetURL(t *testing.T) {
	accesscode = "code123!"
	result := GetActivePlayersSetURL()
	if result != setActivePlayersURLPrefix+accesscode {
		t.Error("Error in setActivePlayersURL")
	}
}
