package shared

import (
	"math/rand"
	"time"

	_ "github.com/joho/godotenv/autoload" // load env variables
)

func init() {
	rand.Seed(time.Now().UTC().UnixNano()) //randomize name creation
}

// RandString creates a random string with lowercase characters
func RandString(n int) string {
	letters := []rune("abcdefghijklmnopqrstuvwxyz")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// GetServiceNameFromPodName converts the name of the pod to a similar name for the service
func GetServiceNameFromPodName(podName string) string {
	return podName
}

// GetPodNameFromServiceName converts the name of the service to a similar name for the pod
func GetPodNameFromServiceName(serviceName string) string {
	return serviceName
}
