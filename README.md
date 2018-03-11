# AzureGameServersScalingKubernetes

~ WORK IN PROGRESS ~

Create a new AKS cluster:

```bash
AKS_RESOURCE_GROUP=aksopenarenarg
AKS_NAME=aksopenarena
AKS_LOCATION=eastus # for better performance, choose the location that your Azure Container Instances will be deployed

az provider register -n Microsoft.ContainerService
az login
az group create --name $AKS_RESOURCE_GROUP --location $AKS_LOCATION
az aks create --resource-group $AKS_RESOURCE_GROUP --name $AKS_NAME --node-count 1 --ssh-key-value ~/.ssh/id_rsa.pub --node-vm-size Standard_A1_v2 #this will take some time...
sudo az aks install-cli
az aks get-credentials --resource-group $AKS_RESOURCE_GROUP --name $AKS_NAME
```

Copy the OpenArena files:

```bash
# Change these parameters as needed
AKS_PERS_STORAGE_ACCOUNT_NAME=openarena$RANDOM
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
STORAGE_ACCOUNT=$(az storage account list --resource-group $AKS_RESOURCE_GROUP --query "[?contains(name,'$AKS_PERS_STORAGE_ACCOUNT_NAME')].[name]" --output tsv)
echo $STORAGE_ACCOUNT

STORAGE_KEY=$(az storage account keys list --resource-group $AKS_RESOURCE_GROUP --account-name $STORAGE_ACCOUNT --query "[0].value" --output tsv)
echo $STORAGE_KEY
```

Mount to copy the files (e.g. from a Linux machine)
```bash
sudo mount -t cifs //accountname.file.core.windows.net/openarenadata /path -o vers=3.0,username=accountname,password=...,dir_mode=0777,file_mode=0777
```