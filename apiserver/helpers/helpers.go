package helpers

import (
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/dgkanatsios/azuregameserversscalingkubernetes/shared"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var accesscode string
var Clientset, Dedicatedgameserverclientset = shared.GetClientSet()

const Namespace string = "game"

var secretsClient = Clientset.Core().Secrets(Namespace)

func IsAPICallAuthorized(w http.ResponseWriter, r *http.Request) bool {
	code := r.FormValue("code")

	if !authenticateCode(code) {
		w.WriteHeader(401)
		w.Write([]byte("Unathorized"))
		return false
	}
	return true
}

func authenticateCode(code string) bool {
	return code == getAccessCode()
}

func getAccessCode() string {
	if accesscode == "" { //if we haven't accessed the code
		secret, err := secretsClient.Get("apiaccesscode", meta_v1.GetOptions{})
		if err != nil {
			log.Fatalf("Cannot get code due to %s", err.Error())
		}
		accesscode = string(secret.Data["code"])
	}
	return accesscode
}
