package main

import (
	"log"

	storage "github.com/Azure/azure-sdk-for-go/storage"
	"github.com/dgkanatsios/AzureGameServersScalingKubernetes/shared"
)

//good samples here: https://github.com/luigialves/sample-golang-with-azure-table-storage/blob/master/sample.go

func upsertEntity(name string, ip string, node string) {
	storageclient := shared.GetStorageClient()

	tableservice := storageclient.GetTableService()

	table := tableservice.GetTableReference(shared.TableName)
	table.Create(shared.Timeout, storage.MinimalMetadata, nil)

	//partition key is the same as row key (pod's name)
	entity := table.GetEntityReference(name, name)

	props := map[string]interface{}{
		"PublicIP": ip,
		"NodeName": node,
	}

	entity.Properties = props

	if err := entity.InsertOrMerge(nil); err != nil {
		log.Fatalf("Cannot insert or merge entity due to ", err)
	}
}

func getEntity(name string) *storage.Entity {
	storageclient := shared.GetStorageClient()

	tableservice := storageclient.GetTableService()

	table := tableservice.GetTableReference(shared.TableName)
	table.Create(shared.Timeout, storage.MinimalMetadata, nil)

	//partition key is the same as row key (pod's name)
	entity := table.GetEntityReference(name, name)

	if err := entity.Get(shared.Timeout, storage.MinimalMetadata, nil); err != nil {
		log.Fatalf("Cannot get entity due to ", err)
	}

	return entity
}
