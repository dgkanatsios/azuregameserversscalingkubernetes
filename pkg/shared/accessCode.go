package shared

import (
	log "github.com/sirupsen/logrus"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// AuthenticateWebServerCode authenticates the user request by comparing the given code with the actual
func AuthenticateWebServerCode(code string) (bool, error) {
	if accesscode == "" {

		client, _, err := GetClientSet()
		if err != nil {
			return false, err
		}
		_, err = GetAccessCode(client)
		if err != nil {
			return false, err
		}
	}

	return code == accesscode, nil
}

var accesscode string

func GetAccessCode(client kubernetes.Interface) (string, error) {
	if accesscode == "" { //if we haven't accessed the code
		// client, _, err := GetClientSet()
		// if err != nil {
		// 	return "", err
		// }
		secret, err := client.Core().Secrets(GameNamespace).Get(APIAccessCodeSecretName, meta_v1.GetOptions{})
		if err != nil {
			log.Fatalf("Cannot get API Server access code due to %s", err.Error())
		}
		accesscode = string(secret.Data["code"])
	}
	return accesscode, nil
}
