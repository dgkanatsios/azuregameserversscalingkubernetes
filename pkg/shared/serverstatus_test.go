package shared

import "testing"

func TestInitializeSetServerStatusURL(t *testing.T) {

	accesscode = "code123!"
	err := initializeSetServerStatusURL()
	if err != nil {
		t.Error("Error should be nil")
	}

	if setServerStatusURL != setServerStatusURLPrefix+accesscode {
		t.Error("Error in setServerStatusURL")
	}
}

func TestGetServerStatusSetURL(t *testing.T) {
	accesscode = "code123!"
	result := GetServerStatusSetURL()
	if result != setServerStatusURLPrefix+accesscode {
		t.Error("Error in setServerStatusURL")
	}
}
