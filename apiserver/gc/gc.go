package gc

import (
	"fmt"
	"time"

	helpers "github.com/dgkanatsios/azuregameserversscalingkubernetes/apiserver/helpers"
	shared "github.com/dgkanatsios/azuregameserversscalingkubernetes/shared"
	log "github.com/sirupsen/logrus"
)

// Run starts Garbage Collector
func Run() {
	log.Println("Starting Garbage Collector")
	for {
		//check if there are any dedicated game servers with status 'MarkedForDeletion' and zero sessions
		entities, err := shared.GetEntitiesMarkedForDeletionWithZeroSessions()

		if err != nil {
			// we should probably examine the error and exit if fatal
			// just log it
			log.Printf("error in GC: %s", err.Error())
		}

		for _, entity := range entities {
			err = helpers.Dedicatedgameserverclientset.AzuregamingV1alpha1().DedicatedGameServers(entity.PartitionKey).Delete(entity.RowKey, nil)
			if err != nil {
				msg := fmt.Sprintf("cannot delete DedicatedGameServer due to %s", err.Error())
				log.Print(msg)
			}
		}

		time.Sleep(1 * time.Minute)
	}
}
