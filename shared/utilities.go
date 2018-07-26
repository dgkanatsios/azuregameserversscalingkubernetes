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

// GetRandomInt returns a random number
func GetRandomInt(min int, max int) int {
	return rand.Intn(max-min) + min
}

func GetRandomPort() (int, error) {
	var port int
	//get a random port
	port = GetRandomInt(MinPort, MaxPort)
	for {
		result, err := CreatePortEntity(port)

		if err != nil {
			return 0, err
		}

		if result {
			break
		} else {
			port = GetRandomInt(MinPort, MaxPort)
		}
	}
	return port, nil
}

// GetRandomIndexes will return *count* random integers from a hypothetical slice of *length*
// For example, we'll take two random indexes from a length-five slice
func GetRandomIndexes(length int, count int) []int {

	if count > length {
		panic("Count > length, something is really wrong here")
	}

	sliceToReturn := make([]int, count)

	for i := 0; i < count; i++ {
		var rand int
		rand = GetRandomInt(0, length-1)

		for {
			found := false
			for j := 0; j < i; j++ {
				if sliceToReturn[j] == rand {
					found = true
					break
				}
			}
			if !found {
				break
			} else {
				rand = GetRandomInt(0, length-1)
			}
		}
		sliceToReturn[i] = rand
	}
	return sliceToReturn
}
