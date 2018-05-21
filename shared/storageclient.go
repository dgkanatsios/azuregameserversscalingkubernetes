package shared

import (
	"os"

	log "github.com/Sirupsen/logrus"

	"github.com/Azure/azure-sdk-for-go/storage"
)

var instance *storage.Client

// GetStorageClient gets a singleton Azure storage client
func GetStorageClient() *storage.Client {
	if instance == nil {
		var err error
		instance, err = getBasicClient() // <--- NOT THREAD SAFE
		if err != nil {
			log.Fatalf("Cannot instantiate storage client due to %s ", err)
		}
	}
	return instance
}

func getBasicClient() (*storage.Client, error) {
	name := os.Getenv("STORAGE_ACCOUNT_NAME")
	key := os.Getenv("STORAGE_ACCOUNT_KEY")
	cli, err := storage.NewBasicClient(name, key)
	return &cli, err
}
