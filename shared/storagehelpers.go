package shared

import (
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"

	storage "github.com/Azure/azure-sdk-for-go/storage"
)

//good samples here: https://github.com/luigialves/sample-golang-with-azure-table-storage/blob/master/sample.go

// StorageEntity represents a pod
type StorageEntity struct {
	Name           string
	PublicIP       string
	NodeName       string
	Status         string
	Port           string
	ActiveSessions *int
}

// UpsertEntity upserts entity
func UpsertEntity(pod *StorageEntity) error {
	storageclient := GetStorageClient()

	tableservice := storageclient.GetTableService()

	table := tableservice.GetTableReference(TableName)
	table.Create(Timeout, storage.MinimalMetadata, nil)

	//partition key is the same as row key (pod's name)
	entity := table.GetEntityReference(pod.Name, pod.Name)

	props := make(map[string]interface{})

	if pod.PublicIP != "" {
		props["PublicIP"] = pod.PublicIP
	}

	if pod.NodeName != "" {
		props["NodeName"] = pod.NodeName
	}

	if pod.Status != "" {
		props["Status"] = pod.Status
	}

	if pod.Port != "" {
		props["Port"] = pod.Port
	}

	if pod.ActiveSessions != nil {
		props["ActiveSessions"] = *pod.ActiveSessions
	}

	entity.Properties = props

	return entity.InsertOrMerge(nil)
}

// GetEntity gets table entity
func GetEntity(name string) (*storage.Entity, error) {
	storageclient := GetStorageClient()

	tableservice := storageclient.GetTableService()

	table := tableservice.GetTableReference(TableName)
	table.Create(Timeout, storage.MinimalMetadata, nil)

	//partition key is the same as row key (pod's name)
	entity := table.GetEntityReference(name, name)

	err := entity.Get(Timeout, storage.MinimalMetadata, nil)

	return entity, err
}

// DeleteEntity deletes table entity. Will suppress 404 errors
func DeleteEntity(name string) error {
	storageclient := GetStorageClient()

	tableservice := storageclient.GetTableService()

	table := tableservice.GetTableReference(TableName)
	table.Create(Timeout, storage.MinimalMetadata, nil)

	//partition key is the same as row key (pod's name)
	entity := table.GetEntityReference(name, name)

	err := entity.Delete(true, nil)

	if err != nil && !strings.Contains(err.Error(), "StatusCode=404, ErrorCode=ResourceNotFound") {
		return err
	}
	return nil
}

// GetRunningEntities returns all entities in the running state
func GetRunningEntities() ([]*storage.Entity, error) {
	storageclient := GetStorageClient()

	tableservice := storageclient.GetTableService()

	table := tableservice.GetTableReference(TableName)
	table.Create(Timeout, storage.MinimalMetadata, nil)

	result, err := table.QueryEntities(Timeout, storage.MinimalMetadata, &storage.QueryOptions{
		Filter: "Status eq 'Running'",
	})

	return result.Entities, err
}

// IsPortUsed reports whether a specified port is used by a pod
func IsPortUsed(port int) bool {
	storageclient := GetStorageClient()

	tableservice := storageclient.GetTableService()

	table := tableservice.GetTableReference(TableName)
	table.Create(Timeout, storage.MinimalMetadata, nil)

	result, err := table.QueryEntities(Timeout, storage.MinimalMetadata, &storage.QueryOptions{
		Filter: "Port eq '" + strconv.Itoa(port) + "'",
	})

	if err != nil {
		log.Fatalf("Cannot get entities due to %s", err)
	}

	return len(result.Entities) > 0
}
