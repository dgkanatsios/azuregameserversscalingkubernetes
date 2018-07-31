package gc

import (
	"fmt"
	"time"

	shared "github.com/dgkanatsios/azuregameserversscalingkubernetes/shared"
	log "github.com/sirupsen/logrus"
)

// Run starts Garbage Collector, it will check for MarkedForDeletion and 0 players every 'd' Duration
func Run(d time.Duration) {

	_, dgsClient, err := shared.GetClientSet()
	if err != nil {
		log.Panicf("Cannot initialize connection to cluster due to %v", err)
	}

	log.Println("Starting Garbage Collector")
	for {

		//check if there are any dedicated game servers with status 'MarkedForDeletion' and zero players
		entities, err := shared.GetDedicatedGameServersMarkedForDeletionWithZeroPlayers()

		if err != nil {
			// we should probably examine the error and exit if fatal
			// just log it
			log.Printf("error in GC: %s", err.Error())
		}

		for _, entity := range entities {
			err = dgsClient.AzuregamingV1alpha1().DedicatedGameServers(entity.Namespace).Delete(entity.Name, nil)
			if err != nil {
				msg := fmt.Sprintf("cannot delete DedicatedGameServer due to %s", err.Error())
				log.Print(msg)
			}
		}

		time.Sleep(d)
	}
}
