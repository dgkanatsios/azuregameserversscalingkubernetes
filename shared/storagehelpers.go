package shared

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	storage "github.com/Azure/azure-sdk-for-go/storage"
	dgsv1alpha1 "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/apis/azuregaming/v1alpha1"
)

//good samples here: https://github.com/luigialves/sample-golang-with-azure-table-storage/blob/master/sample.go

// GameServerEntity represents a pod
type GameServerEntity struct {
	// don't forget to update UpsertGameEntity each time you add a field here
	Name                          string
	Namespace                     string
	PublicIP                      string
	NodeName                      string
	PodStatus                     string
	Ports                         string
	ActivePlayers                 string
	GameServerStatus              string
	DedicatedGameServerCollection string
}

func GetPortInfoFromPortString(portsStr string) ([]dgsv1alpha1.PortInfoExtended, error) {
	var ports []dgsv1alpha1.PortInfoExtended
	err := json.Unmarshal([]byte(portsStr), &ports)

	if err != nil {
		return nil, err
	}

	return ports, nil
}

func (gs *GameServerEntity) SetPortsFromPortInfo(ports []dgsv1alpha1.PortInfoExtended) error {

	b, err := json.Marshal(ports)
	if err != nil {
		return err
	}

	gs.Ports = string(b)
	return nil
}

// UpsertGameServerEntity upserts entity
func UpsertGameServerEntity(pod *GameServerEntity) error {

	if pod.Name == "" || pod.Namespace == "" {
		return errors.New("New pod should include both Name and Namespace properties")
	}

	storageclient := GetStorageClient()

	tableservice := storageclient.GetTableService()

	table := tableservice.GetTableReference(GameServersTableName)
	table.Create(Timeout, storage.MinimalMetadata, nil)

	//partition key is the same as row key (pod's name)
	entity := table.GetEntityReference(pod.Namespace, pod.Name)

	props := make(map[string]interface{})

	if pod.PublicIP != "" {
		props["PublicIP"] = pod.PublicIP
	}

	if pod.NodeName != "" {
		props["NodeName"] = pod.NodeName
	}

	if pod.PodStatus != "" {
		props["PodStatus"] = pod.PodStatus
	}

	if pod.Ports != "" {
		props["Ports"] = pod.Ports
	}

	if pod.ActivePlayers != "" {
		props["ActivePlayers"] = pod.ActivePlayers
	}

	if pod.GameServerStatus != "" {
		props["GameServerStatus"] = pod.GameServerStatus
	}

	if pod.DedicatedGameServerCollection != "" {
		props["DedicatedGameServerCollection"] = pod.DedicatedGameServerCollection
	}

	entity.Properties = props

	return entity.InsertOrMerge(nil)
}

// GetEntity gets table entity
func GetEntity(namespace, name string) (*storage.Entity, error) {
	storageclient := GetStorageClient()

	tableservice := storageclient.GetTableService()

	table := tableservice.GetTableReference(GameServersTableName)
	table.Create(Timeout, storage.MinimalMetadata, nil)

	entity := table.GetEntityReference(namespace, name)

	err := entity.Get(Timeout, storage.MinimalMetadata, nil)

	return entity, err
}

// DeleteDedicatedGameServerEntityAndPods deletes DedicatedGameServer table entity. Will suppress 404 errors
func DeleteDedicatedGameServerEntityAndPods(namespace, name string) error {
	storageclient := GetStorageClient()

	tableservice := storageclient.GetTableService()

	table := tableservice.GetTableReference(GameServersTableName)
	table.Create(Timeout, storage.MinimalMetadata, nil)

	// retrieve entire object
	entity, err := GetEntity(namespace, name)

	if err != nil {
		return err
	}

	if portsString, ok := entity.Properties["Ports"]; ok {

		ports, errPorts := GetPortInfoFromPortString(portsString.(string))

		if errPorts != nil {
			//suppress and keep deleting
			log.Printf("Cannot get portinfo from string %s due to %s", portsString.(string), errPorts)
		}

		for _, portInfo := range ports {
			errDeletePort := DeletePortEntity(portInfo.HostPort)
			if errDeletePort != nil {
				//suppress and keep deleting
				log.Printf("Cannot delete port %d due to %s", portInfo.HostPort, errDeletePort)
			}
		}
	}

	errDelete := entity.Delete(true, nil)

	if errDelete != nil && !strings.Contains(err.Error(), "StatusCode=404, ErrorCode=ResourceNotFound") {
		return errDelete
	}
	return nil
}

// GetRunningEntities returns all entities in the running state
func GetRunningEntities() ([]*storage.Entity, error) {
	storageclient := GetStorageClient()

	tableservice := storageclient.GetTableService()

	table := tableservice.GetTableReference(GameServersTableName)
	table.Create(Timeout, storage.MinimalMetadata, nil)

	result, err := table.QueryEntities(Timeout, storage.MinimalMetadata, &storage.QueryOptions{
		Filter: fmt.Sprintf("PodStatus eq '%s' and GameServerStatus eq '%s'", "Running", GameServerStatusRunning),
	})

	if err != nil {
		return nil, err
	}

	return result.Entities, nil
}

// GetEntitiesMarkedForDeletionWithZeroPlayers returns all entities marked for deletion with 0 active players
func GetEntitiesMarkedForDeletionWithZeroPlayers() ([]*storage.Entity, error) {
	storageclient := GetStorageClient()

	tableservice := storageclient.GetTableService()

	table := tableservice.GetTableReference(GameServersTableName)
	table.Create(Timeout, storage.MinimalMetadata, nil)

	result, err := table.QueryEntities(Timeout, storage.MinimalMetadata, &storage.QueryOptions{
		Filter: fmt.Sprintf("GameServerStatus eq '%s' and ActivePlayers eq '0'", GameServerStatusMarkedForDeletion),
	})

	if err != nil {
		return nil, err
	}

	return result.Entities, nil
}

func CreatePortEntity(port int) (bool, error) {
	storageclient := GetStorageClient()
	tableservice := storageclient.GetTableService()
	table := tableservice.GetTableReference(PortsTableName)
	table.Create(Timeout, storage.MinimalMetadata, nil)
	stringPort := strconv.Itoa(port)
	entity := table.GetEntityReference(stringPort, stringPort)
	err := entity.Insert(storage.MinimalMetadata, nil)
	if err != nil {
		if strings.Contains(err.Error(), "StatusCode=409") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// DeletePortEntity deletes Port table entity. Will suppress 404 errors
func DeletePortEntity(port int32) error {
	storageclient := GetStorageClient()

	tableservice := storageclient.GetTableService()

	table := tableservice.GetTableReference(PortsTableName)
	table.Create(Timeout, storage.MinimalMetadata, nil)

	stringPort := strconv.Itoa(int(port))

	entity := table.GetEntityReference(stringPort, stringPort)

	err := entity.Delete(true, nil)

	if err != nil && !strings.Contains(err.Error(), "StatusCode=404, ErrorCode=ResourceNotFound") {
		return err
	}
	return nil
}
