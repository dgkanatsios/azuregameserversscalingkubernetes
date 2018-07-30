# Installation

## Create the AKS cluster

Here are the necessary commands to create a new AKS cluster, you can use either [Azure CLI](https://docs.microsoft.com/en-us/cli/azure/?view=azure-cli-latest) or [Azure Cloud Shell](https://azure.microsoft.com/en-us/features/cloud-shell/). Check [here](https://docs.microsoft.com/en-us/azure/aks/container-service-quotas#region-availability) for AKS region availability. 

```bash
az login # you don't need to do this if you're using Azure Cloud shell
# you should modify these values to your preferred ones
AKS_RESOURCE_GROUP=aksopenarenarg
AKS_NAME=aksopenarena
AKS_LOCATION=westeurope

# create a resource group
az group create --name $AKS_RESOURCE_GROUP --location $AKS_LOCATION
# create a new AKS cluster
az aks create --resource-group $AKS_RESOURCE_GROUP --name $AKS_NAME --node-count 1 --ssh-key-value ~/.ssh/id_rsa.pub --node-vm-size Standard_A1_v2 --kubernetes-version 1.10.5 --enable-rbac # this will take some time...
sudo az aks install-cli
az aks get-credentials --resource-group $AKS_RESOURCE_GROUP --name $AKS_NAME
```

## Allow Network Traffic

This project requires VMs/Kubernetes Worker Nodes to have Public IPs and be able to accept network traffic at port range 20000-30000. To allow network traffic you need to perform the following steps after your cluster gets created:

* Login to the Azure Portal
* Find the resource group where the AKS resources are kept, which should have a name like `MC_resourceGroupName_AKSName_location`. Alternative, you can type `az resource show --namespace Microsoft.ContainerService --resource-type managedClusters -g $AKS_RESOURCE_GROUP -n $AKS_NAME -o json | jq .properties.nodeResourceGroup`
* Find the Network Security Group object, which should have a name like `aks-agentpool-********-nsg`
* Select **Inbound Security Rules**
* Select **Add** to create a new Rule with **Any** as the protocol and **20000-30000** as the Destination Port Ranges. Pick a proper name and leave everything else at their default values

Alternatively, you can use the following command, after setting the `$RESOURCE_GROUP_WITH_AKS_RESOURCES` and `$NSG_NAME` variables with proper values:

```bash
az network nsg rule create \
  --resource-group $RESOURCE_GROUP_WITH_AKS_RESOURCES \
  --nsg-name $NSG_NAME \
  --name AKSDedicatedGameServerRule \
  --access Allow \
  --protocol "*" \
  --direction Inbound \
  --priority 1000 \
  --source-port-range "*" \
  --destination-port-range 20000-30000
```

## Assigning Public IPs to the existing Nodes and to potential new ones

As of now, AKS Nodes don't get a Public IP by default (even though you could use [acs-engine](https://github.com/Azure/acs-engine) to achieve that). To assign Public IPs to a Node/VM, you can find the Resource Group where the AKS resources are installed on the [portal](https://portal.azure.com) (it should have a name like `MC_resourceGroupName_AKSName_location`). Then, you can follow the instructions [here](https://blogs.technet.microsoft.com/srinathv/2018/02/07/how-to-add-a-public-ip-address-to-azure-vm-for-vm-failed-over-using-asr/) to create a new Public IP and assign it to the Node/VM. For more information on Public IPs for VM NICs, see [this document](https://docs.microsoft.com/azure/virtual-network/virtual-network-network-interface-addresses). 

Alternatively, you can use [this](https://github.com/dgkanatsios/AksNodePublicIPController) project which will take care of
- Creating and assigning Public IPs to existing Nodes
- Creating and assigning Public IPs to new Nodes, e.g. in case of a cluster scale out
- Deleting Public IPs for Nodes that get removed from the cluster, e.g. cluster scale in

## Necessary stuff to test OpenArena game

To test the project's installation using the OpenArena game, you should create a storage account to copy the OpenArena files. This will allow us to use the Docker image on the `/openarena` folder on this repo, which access the game files from a volume mount. This makes for a Docker image that is faster to load, thus a smaller dedicated game server boot time.

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
AZURE_STORAGE_CONNECTION_STRING=`az storage account show-connection-string --resource-group $AKS_RESOURCE_GROUP --name $AKS_PERS_STORAGE_ACCOUNT_NAME --output tsv`

# Create the file share
az storage share create -n $AKS_PERS_SHARE_NAME

# Get Storage credentials
STORAGE_ACCOUNT_NAME=$(az storage account list --resource-group $AKS_RESOURCE_GROUP --query "[?contains(name,'$AKS_PERS_STORAGE_ACCOUNT_NAME')].[name]" --output tsv)
echo $STORAGE_ACCOUNT_NAME

STORAGE_ACCOUNT_KEY=$(az storage account keys list --resource-group $AKS_RESOURCE_GROUP --account-name $STORAGE_ACCOUNT_NAME --query "[0].value" --output tsv)
echo $STORAGE_ACCOUNT_KEY
```

If you want to test the project locally, you should create a new .env file (based on the controller/cmd/controller/.env.sample one) and use the previous values.

Mount to copy the files (e.g. from a Linux machine) - [instructions](https://docs.microsoft.com/en-us/azure/storage/files/storage-how-to-use-files-linux):
```bash
sudo mount -t cifs //$STORAGE_ACCOUNT_NAME.file.core.windows.net/$AKS_PERS_SHARE_NAME /path -o vers=3.0,username=$STORAGE_ACCOUNT_NAME,password=$STORAGE_ACCOUNT_KEY,dir_mode=0777,file_mode=0777
```

Create a Kubernetes secret that will hold our storage account credentials:
```bash
kubectl create secret generic openarena-storage-secret --from-literal=azurestorageaccountname=$STORAGE_ACCOUNT_NAME --from-literal=azurestorageaccountkey=$STORAGE_ACCOUNT_KEY
```

Create a Kubernetes secret that will hold our access code for the API:
```bash
kubectl create secret generic apiaccesscode --from-literal=code=YOUR_CODE_HERE
```

Create DedicatedGameServer Custom Resource Definition:
```bash
cd artifacts
kubectl apply -f artifacts/crd
```

Create `apiserver` and `controller` K8s deployments:
```bash
kubectl apply -f deploy.apiserver-controller.yaml
```

To update your API and Controller deployments:
```bash
cd various
./updatedeployments.sh
```

```bash
az aks browse --resource-group $AKS_RESOURCE_GROUP --name $AKS_NAME
```