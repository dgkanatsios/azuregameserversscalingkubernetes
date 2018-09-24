package shared

import "testing"

func TestGetAccessCode(t *testing.T) {
	accesscode = "code123!"
	code, _ := getAccessCode()
	if code != accesscode {
		t.Error("Codes should be the same")
	}
}

func TestAuthenticateWebServerCode(t *testing.T) {
	accesscode = "code123!"
	result, _ := AuthenticateWebServerCode("code123!")
	if !result {
		t.Error("Should be true")
	}
}
