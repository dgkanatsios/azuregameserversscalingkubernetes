package shared

import (
	"math/rand"
	"time"

	"github.com/sirupsen/logrus"

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
	if max-min <= 0 { //Intn panics if argument is <=0
		return 0
	}
	return rand.Intn(max-min) + min
}

// GetRandomIndexes will return *count* random integers from a hypothetical slice of *length*
// For example, think that we want to take two random indexes from a length-five slice
func GetRandomIndexes(length int, count int) []int {

	if count > length {
		panic("Count > length, something is really wrong here")
	}

	sliceToReturn := make([]int, count)

	for i := 0; i < count; i++ {
		rand := GetRandomInt(0, length-1)

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

// AreMapsSame compares two map[string]string objects
func AreMapsSame(map1, map2 map[string]string) bool {
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

// Logger returns a default Logger
func Logger() *logrus.Logger {
	return logrus.New()
}

// SliceContains returns true if the specific int32 value is contained in the slice
func SliceContains(slice []int32, value int32) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

func generateName(prefix string) string {
	return prefix + "-" + randString(RandStringSize)
}
