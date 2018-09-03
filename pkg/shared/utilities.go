package shared

import (
	"math/rand"
	"time"

	dgsv1alpha1 "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/apis/azuregaming/v1alpha1"
	_ "github.com/joho/godotenv/autoload" // load env variables
)

func init() {
	rand.Seed(time.Now().UTC().UnixNano()) //randomize name creation
}

// randString creates a random string with lowercase characters
func randString(n int) string {
	letters := []rune("abcdefghijklmnopqrstuvwxyz")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// GetRandomInt returns a random number
func GetRandomInt(min int, max int) int {
	if max-min == 0 { //Intn panics if argument is <=0
		return 0
	}
	return rand.Intn(max-min) + min
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

// GenerateRandomName generates a random name with a prefix
var GenerateRandomName func(prefix string) string

func generateRandomName(prefix string) string {
	return prefix + "-" + randString(5)
}

// HasDedicatedGameServerChanged returns true if *all* of the following DGS properties have changed
// dgsState, podState, publicIP, nodeName, activePlayers
// As expected, it returns false if at least one has changed
func HasDedicatedGameServerChanged(oldDGS, newDGS *dgsv1alpha1.DedicatedGameServer) bool {
	// we check if all of the following fields are the same

	if oldDGS.Status.DedicatedGameServerState == newDGS.Status.DedicatedGameServerState &&
		oldDGS.Status.PodState == newDGS.Status.PodState &&
		oldDGS.Spec.PublicIP == newDGS.Spec.PublicIP &&
		oldDGS.Spec.NodeName == newDGS.Spec.NodeName &&
		oldDGS.Spec.ActivePlayers == newDGS.Spec.ActivePlayers &&
		areMapsSame(oldDGS.Labels, newDGS.Labels) &&
		oldDGS.Spec.Template.Containers[0].Image == newDGS.Spec.Template.Containers[0].Image {

		//we should also check for ports as well
		//or not :)

		return false
	}

	return true
}

func areMapsSame(map1, map2 map[string]string) bool {
	if len(map1) != len(map2) {
		return false
	}

	for map1key, map1value := range map1 {
		if map2value, ok := map2[map1key]; ok {
			if map2value != map1value {
				return false
			}
		} else {
			return false
		}

	}

	return true
}
