[![Go Report Card](https://goreportcard.com/badge/github.com/dgkanatsios/AzureGameServersScalingKubernetes)](https://goreportcard.com/report/github.com/dgkanatsios/AzureGameServersScalingKubernetes)
[![Software License](https://img.shields.io/badge/license-MIT-brightgreen.svg?style=flat-square)](LICENSE)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=flat-square)](http://makeapullrequest.com)
[![unofficial Google Analytics for GitHub](https://gaforgithub.azurewebsites.net/api?repo=AzureGameServersScalingKubernetes)](https://github.com/dgkanatsios/gaforgithub)
![](https://img.shields.io/badge/status-alpha-red.svg)

# AzureGameServersScalingKubernetes

~ HEAVY WORK IN PROGRESS DO NOT USE ~

Create a new AKS cluster: 

```bash
AKS_RESOURCE_GROUP=aksopenarenarg
AKS_NAME=aksopenarena
AKS_LOCATION=eastus 

az provider register -n Microsoft.ContainerService
az login
az group create --name $AKS_RESOURCE_GROUP --location $AKS_LOCATION
az aks create --resource-group $AKS_RESOURCE_GROUP --name $AKS_NAME --node-count 1 --ssh-key-value ~/.ssh/id_rsa.pub --node-vm-size Standard_A1_v2 --kubernetes-version 1.9.6 #this will take some time...
sudo az aks install-cli
az aks get-credentials --resource-group $AKS_RESOURCE_GROUP --name $AKS_NAME
```

Create a storage account to copy the OpenArena files:

```bash
# Change these parameters as needed
AKS_PERS_STORAGE_ACCOUNT_NAME=aksopenarena$RANDOM
AKS_PERS_SHARE_NAME=openarenadata

# Create the storage account with the provided parameters
az storage account create \
    --resource-group $AKS_RESOURCE_GROUP \
    --name $AKS_PERS_STORAGE_ACCOUNT_NAME \
    --location $AKS_LOCATION \
    --sku Standard_LRS

# Export the connection string as an environment variable. The following 'az storage share create' command
# references this environment variable when creating the Azure file share.
export AZURE_STORAGE_CONNECTION_STRING=`az storage account show-connection-string --resource-group $AKS_RESOURCE_GROUP --name $AKS_PERS_STORAGE_ACCOUNT_NAME --output tsv`

# Create the file share
az storage share create -n $AKS_PERS_SHARE_NAME

# Get Storage credentials
STORAGE_ACCOUNT_NAME=$(az storage account list --resource-group $AKS_RESOURCE_GROUP --query "[?contains(name,'$AKS_PERS_STORAGE_ACCOUNT_NAME')].[name]" --output tsv)
echo $STORAGE_ACCOUNT_NAME

STORAGE_ACCOUNT_KEY=$(az storage account keys list --resource-group $AKS_RESOURCE_GROUP --account-name $STORAGE_ACCOUNT_NAME --query "[0].value" --output tsv)
echo $STORAGE_ACCOUNT_KEY
```

Mount to copy the files (e.g. from a Linux machine) - [instructions](https://docs.microsoft.com/en-us/azure/storage/files/storage-how-to-use-files-linux)
```bash
sudo mount -t cifs //$STORAGE_ACCOUNT_NAME.file.core.windows.net/$AKS_PERS_SHARE_NAME /path -o vers=3.0,username=$STORAGE_ACCOUNT_NAME,password=$STORAGE_ACCOUNT_KEY,dir_mode=0777,file_mode=0777
```

Create a Kubernetes secret that will hold our storage account credentials
```bash
kubectl create secret generic openarena-storage-secret --from-literal=azurestorageaccountname=$STORAGE_ACCOUNT_NAME --from-literal=azurestorageaccountkey=$STORAGE_ACCOUNT_KEY
```

Create a Kubernetes secret that will hold our access code for the API
```bash
kubectl create secret generic apiaccesscode --from-literal=code=YOUR_CODE_HERE
```

Create `api` and `controller` K8s deployments
```bash
cd various
kubectl apply -f various/deployapihandler.yaml
```

To update your API and Controller deployments using an ugly and non-recommended hack:
```bash
cd various
./updatedeployments.sh
```

```bash
az aks browse --resource-group $AKS_RESOURCE_GROUP --name $AKS_NAME
```