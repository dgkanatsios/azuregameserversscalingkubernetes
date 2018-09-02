package helpers

import (
	"net/http"

	"github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/shared"
)

func IsAPICallAuthenticated(w http.ResponseWriter, r *http.Request) (bool, error) {
	code := r.FormValue("code")

	result, err := shared.AuthenticateWebServerCode(code)

	if err != nil {
		return false, err
	}

	if !result {
		return false, nil
	}
	return true, nil
}
