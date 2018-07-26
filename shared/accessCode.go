package shared

import (
	log "github.com/sirupsen/logrus"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var accesscode string

var Clientset, Dedicatedgameserverclientset = GetClientSet()
var secretsClient = Clientset.Core().Secrets(GameNamespace)

func AuthenticateWebServerCode(code string) bool {
	return code == getAccessCode()
}

func getAccessCode() string {
	if accesscode == "" { //if we haven't accessed the code
		secret, err := secretsClient.Get(APIAccessCodeSecretName, meta_v1.GetOptions{})
		if err != nil {
			log.Fatalf("Cannot get API Server access code due to %s", err.Error())
		}
		accesscode = string(secret.Data["code"])
	}
	return accesscode
}
