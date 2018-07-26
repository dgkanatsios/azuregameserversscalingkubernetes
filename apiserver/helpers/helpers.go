package helpers

import (
	"net/http"

	"github.com/dgkanatsios/azuregameserversscalingkubernetes/shared"
)

func IsAPICallAuthorized(w http.ResponseWriter, r *http.Request) bool {
	code := r.FormValue("code")

	if !shared.AuthenticateWebServerCode(code) {
		return false
	}
	return true
}
