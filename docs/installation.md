# Installation

## Create the AKS cluster

Here are the necessary commands to create a new AKS cluster. To do that you can use either [Azure CLI](https://docs.microsoft.com/en-us/cli/azure/?view=azure-cli-latest) or [Azure Cloud Shell](https://azure.microsoft.com/en-us/features/cloud-shell/). Check [here](https://docs.microsoft.com/en-us/azure/aks/container-service-quotas#region-availability) for AKS region availability. 

```bash
az login # you don't need to do this if you're using Azure Cloud shell
# you should modify these values to your preferred ones
AKS_RESOURCE_GROUP=aksopenarenarg # name of the resource group AKS will be installed
AKS_NAME=aksopenarena # AKS cluster name
AKS_LOCATION=westeurope # AKS datacenter location

# create a resource group
az group create --name $AKS_RESOURCE_GROUP --location $AKS_LOCATION
# create a new AKS cluster
az aks create --resource-group $AKS_RESOURCE_GROUP --name $AKS_NAME --node-count 1 --ssh-key-value ~/.ssh/id_rsa.pub --node-vm-size Standard_A1_v2 --kubernetes-version 1.11.2 --enable-rbac # this will take some time...
sudo az aks install-cli # this will install kubectl
az aks get-credentials --resource-group $AKS_RESOURCE_GROUP --name $AKS_NAME
```

## Allow Network Traffic

This project requires VMs/Kubernetes Worker Nodes to have Public IPs and be able to accept network traffic at port range 20000-30000 from the Internet. To allow do that you need to perform the following steps *after your cluster gets created*:

* Login to the Azure Portal
* Find the resource group where the AKS resources are kept, it should have a name like `MC_resourceGroupName_AKSName_location`. Alternative, you can type `az resource show --namespace Microsoft.ContainerService --resource-type managedClusters -g $AKS_RESOURCE_GROUP -n $AKS_NAME -o json | jq .properties.nodeResourceGroup` on your shell to find it.
* Find the Network Security Group object, which should have a name like `aks-agentpool-********-nsg`
* Select **Inbound Security Rules**
* Select **Add** to create a new Rule with **Any** as the protocol (you could also select between TCP or UDP, depending on your game) and **20000-30000** as the Destination Port Ranges. Pick a proper name for the rule and leave everything else at their default values

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

As of now, AKS Nodes don't get a Public IP by default (even though you could use [acs-engine](https://github.com/Azure/acs-engine) to create a self-managed K8s cluster that supports that). To assign Public IP to a Node/VM, you can find the Resource Group where the AKS resources are installed on the [portal](https://portal.azure.com) (it should have a name like `MC_resourceGroupName_AKSName_location`). Then, you can follow the instructions [here](https://blogs.technet.microsoft.com/srinathv/2018/02/07/how-to-add-a-public-ip-address-to-azure-vm-for-vm-failed-over-using-asr/) to create a new Public IP and assign it to the Node/VM. For more information on Public IPs for VM NICs, see [this document](https://docs.microsoft.com/azure/virtual-network/virtual-network-network-interface-addresses). 

Alternatively, you can use [this](https://github.com/dgkanatsios/AksNodePublicIPController) project which will take care of
- Creating and assigning Public IPs to existing Nodes
- Creating and assigning Public IPs to new Nodes, e.g. in case of a cluster scale out
- Deleting Public IPs for Nodes that get removed from the cluster, e.g. cluster scale in

You can check its [instructions](https://github.com/dgkanatsios/AksNodePublicIPController/blob/master/README.md), setup is pretty easy. 

## CRD and APIServer/Controllers installation

First of all, create a Kubernetes secret that will hold our access code for the APIServer's endpoints:
```bash
# just use a code that will be kept hidden
kubectl create secret generic apiaccesscode --from-literal=code=YOUR_CODE_HERE
```

Then, create DedicatedGameServer Custom Resource Definition:

```bash
kubectl apply -f https://raw.githubusercontent.com/dgkanatsios/azuregameserversscalingkubernetes/master/artifacts/crds/dedicatedgameservercollection.yaml 
kubectl apply -f https://raw.githubusercontent.com/dgkanatsios/azuregameserversscalingkubernetes/master/artifacts/crds/dedicatedgameserver.yaml
```

Create `apiserver` and `controller` K8s deployments:

```bash
# for an RBAC-enabled cluster (use this if you have followed the instructions step by step)
kubectl apply -f https://raw.githubusercontent.com/dgkanatsios/azuregameserversscalingkubernetes/master/artifacts/deploy.apiserver-controller.yaml
# use this file for a cluster not configured with RBAC authentication
# kubectl apply -f https://raw.githubusercontent.com/dgkanatsios/azuregameserversscalingkubernetes/master/artifacts/deploy.apiserver-controller.no-rbac.yaml
```

You're done! You can now test the Node.js echo demo app.

## Testing with Node.js echo demo app

### Creation of DedicatedGameServerCollection

Use this command to create a collection of DedicatedGameServers. The 'game' that will be created is the simple Node.js echo app, which source code is in `demos/simplenodejsudp` folder. This collection will create 5 DedicatedGameServers.

```bash
kubectl apply -f https://raw.githubusercontent.com/dgkanatsios/azuregameserversscalingkubernetes/master/artifacts/examples/simplenodejsudp/dedicatedgameservercollection.yaml
```

Hopefully, 5 instances will be created. Type `kubectl get dgsc` to see the DedicatedGameCollection as well as its status

```
NAME                                 REPLICAS   AVAILABLEREPLICAS   GAMESERVERCOLLECTIONSTATE   PODCOLLECTIONSTATE
simplenodejsudp-collection-example   5          5                   Running                     Running
```

Now, try `kubectl get dgs` to see the statuses of each individual DedicatedGameServer

```
NAME                                       ACTIVEPLAYERS   GAMESERVERSTATE   PODSTATE   PORTS                                                    PUBLICIP
simplenodejsudp-collection-example-docji   0               Running           Running    [map[containerPort:22222 hostPort:22388 protocol:UDP]]   40.115.5.154
simplenodejsudp-collection-example-jetld   0               Running           Running    [map[containerPort:22222 hostPort:25682 protocol:UDP]]   40.115.7.145
simplenodejsudp-collection-example-nkbso   0               Running           Running    [map[containerPort:22222 hostPort:20866 protocol:UDP]]   40.115.6.130
simplenodejsudp-collection-example-pgqud   0               Running           Running    [map[containerPort:22222 hostPort:20745 protocol:UDP]]   40.115.5.154
simplenodejsudp-collection-example-tjvym   0               Running           Running    [map[containerPort:22222 hostPort:24984 protocol:UDP]]   40.115.6.130
```

You can also see ActivePlayers and assigned ports. Let's try to connect to one of them, we'll use the netcat command. We'll try to connect to the first DedicatedGameServer. The Node's IP is 40.115.5.154 whereas the assigned port is 22388. We're using a UDP connection.

```bash
nc -u 40.115.5.154 22388
```

Now, if everything goes well, you can type whatever you like and the server will echo the message back:

```
hello
simplenodejsudp-collection-example-docji-vhxhr says: hello
```

You can use Ctrl-C (or Cmd-C) to disconnect.

Before we proceed, feel free to check the running pods as well

```bash
kubectl get pods
```

```
NAME                                             READY     STATUS    RESTARTS   AGE
aks-gaming-apiserver-568cc44954-qgwzf            1/1       Running   0          16h
aks-gaming-controller-6f4586895d-xg7kh           1/1       Running   0          16h
aksnodepublicipcontroller-5747674798-zp7wr       1/1       Running   0          7d
simplenodejsudp-collection-example-docji-vhxhr   1/1       Running   0          31m
simplenodejsudp-collection-example-jetld-iaumv   1/1       Running   0          31m
simplenodejsudp-collection-example-nkbso-dfmvi   1/1       Running   0          31m
simplenodejsudp-collection-example-pgqud-cozlk   1/1       Running   0          31m
simplenodejsudp-collection-example-tjvym-zfcqt   1/1       Running   0          31m
```

First two pods correspond to APIServer and Controller executable whereas the third is responsible for assigning Public IPs to Kubernetes Worker Nodes. The last 5 correspond to our DedicatedGameServers.

### Scaling

Let's scale out our DedicatedGameServerCollection to 8 replicas.

```bash
kubectl scale dgsc simplenodejsudp-collection-example --replicas=8
```

If everything goes well, eventually you will have 8 available replicas

```bash
kubectl get dgsc
```

```
NAME                                 REPLICAS   AVAILABLEREPLICAS   GAMESERVERCOLLECTIONSTATE   PODCOLLECTIONSTATE
simplenodejsudp-collection-example   8          8                   Running                     Running
```

Great! Let's mock our system so that all DedicatedGameServers have 5 active players.

```bash
# get DGS names
dgs=`kubectl get dgs -l DedicatedGameServerCollectionName=simplenodejsudp-collection-example | cut -d ' ' -f 1 | sed 1,1d`
# update DGS.Spec.ActivePlayers
kubectl patch dgs $dgs -p '[{ "op": "replace", "path": "/status/activePlayers", "value": 5 },]' --type='json'
# update DGS.Labels[ActivePlayers]
kubectl label dgs $dgs ActivePlayers=5 --overwrite
```

Let's scale our DedicatedGameServerCollection to 6 replicas

```bash
kubectl scale dgsc simplenodejsudp-collection-example --replicas=6
```

DedicatedGameServerCollection has 6 available replicas

```bash
kubectl get dgsc
```

```bash
kubectl scale dgsc simplenodejsudp-collection-example --replicas=2
```

```
NAME                                 REPLICAS   AVAILABLEREPLICAS   GAMESERVERCOLLECTIONSTATE   PODCOLLECTIONSTATE
simplenodejsudp-collection-example   6          6                   Running                     Running
```

However, there are still 8 DedicatedGameServers on our cluster. Check them out, including labels

```bash
kubectl get dgs --show-labels
```

```
NAME                                       ACTIVEPLAYERS   GAMESERVERSTATE     PODSTATE   PORTS                                                    PUBLICIP       LABELS
simplenodejsudp-collection-example-docji   5               MarkedForDeletion   Running    [map[containerPort:22222 hostPort:22388 protocol:UDP]]   40.115.5.154   ActivePlayers=5,DedicatedGameServerState=MarkedForDeletion,OriginalDedicatedGameServerCollectionName=simplenodejsudp-collection-example,PodState=Running,ServerName=simplenodejsudp-collection-example-docji
simplenodejsudp-collection-example-fhlcm   5               Running             Running    [map[containerPort:22222 hostPort:24580 protocol:UDP]]   40.115.5.154   ActivePlayers=5,DedicatedGameServerCollectionName=simplenodejsudp-collection-example,DedicatedGameServerState=Running,PodState=Running,ServerName=simplenodejsudp-collection-example-fhlcm
simplenodejsudp-collection-example-gcmig   5               Running             Running    [map[containerPort:22222 hostPort:26580 protocol:UDP]]   40.115.6.130   ActivePlayers=5,DedicatedGameServerCollectionName=simplenodejsudp-collection-example,DedicatedGameServerState=Running,PodState=Running,ServerName=simplenodejsudp-collection-example-gcmig
simplenodejsudp-collection-example-jetld   5               Running             Running    [map[containerPort:22222 hostPort:25682 protocol:UDP]]   40.115.7.145   ActivePlayers=5,DedicatedGameServerCollectionName=simplenodejsudp-collection-example,DedicatedGameServerState=Running,PodState=Running,ServerName=simplenodejsudp-collection-example-jetld
simplenodejsudp-collection-example-nkbso   5               Running             Running    [map[containerPort:22222 hostPort:20866 protocol:UDP]]   40.115.6.130   ActivePlayers=5,DedicatedGameServerCollectionName=simplenodejsudp-collection-example,DedicatedGameServerState=Running,PodState=Running,ServerName=simplenodejsudp-collection-example-nkbso
simplenodejsudp-collection-example-pgqud   5               Running             Running    [map[containerPort:22222 hostPort:20745 protocol:UDP]]   40.115.5.154   ActivePlayers=5,DedicatedGameServerCollectionName=simplenodejsudp-collection-example,DedicatedGameServerState=Running,PodState=Running,ServerName=simplenodejsudp-collection-example-pgqud
simplenodejsudp-collection-example-tjvym   5               MarkedForDeletion   Running    [map[hostPort:24984 protocol:UDP containerPort:22222]]   40.115.6.130   ActivePlayers=5,DedicatedGameServerState=MarkedForDeletion,OriginalDedicatedGameServerCollectionName=simplenodejsudp-collection-example,PodState=Running,ServerName=simplenodejsudp-collection-example-tjvym
simplenodejsudp-collection-example-xgugp   5               Running             Running    [map[containerPort:22222 hostPort:24710 protocol:UDP]]   40.115.7.145   ActivePlayers=5,DedicatedGameServerCollectionName=simplenodejsudp-collection-example,DedicatedGameServerState=Running,PodState=Running,ServerName=simplenodejsudp-collection-example-xgugp
```

As you can see, 2 of them are MarkedForDeletion and do not belong to the DedicatedGameServerCollection anymore. Still, they are not deleted, since there are players still enjoying the game! Let's update these two game servers so that the system thinks that the game has finished and the players have left the server.

*Make sure to change the value of dgs2 variable with the names of your DedicatedGameServers that are MarkedForDeletion*

```bash
dgs2="simplenodejsudp-collection-example-docji simplenodejsudp-collection-example-tjvym"
# update DGS.Spec.ActivePlayers
kubectl patch dgs $dgs2 -p '[{ "op": "replace", "path": "/status/activePlayers", "value": 0 },]' --type='json'
# update DGS.Labels[ActivePlayers]
kubectl label dgs $dgs2 ActivePlayers=0 --overwrite
```

Now, if you run `kubectl get dgs` you will see that the two MarkedForDeletion servers have disappeared, since the players that were connected to them left the game.

```bash
NAME                                       ACTIVEPLAYERS   GAMESERVERSTATE   PODSTATE   PORTS                                                    PUBLICIP
simplenodejsudp-collection-example-fhlcm   5               Running           Running    [map[hostPort:24580 protocol:UDP containerPort:22222]]   40.115.5.154
simplenodejsudp-collection-example-gcmig   5               Running           Running    [map[containerPort:22222 hostPort:26580 protocol:UDP]]   40.115.6.130
simplenodejsudp-collection-example-jetld   5               Running           Running    [map[protocol:UDP containerPort:22222 hostPort:25682]]   40.115.7.145
simplenodejsudp-collection-example-nkbso   5               Running           Running    [map[containerPort:22222 hostPort:20866 protocol:UDP]]   40.115.6.130
simplenodejsudp-collection-example-pgqud   5               Running           Running    [map[protocol:UDP containerPort:22222 hostPort:20745]]   40.115.5.154
simplenodejsudp-collection-example-xgugp   5               Running           Running    [map[containerPort:22222 hostPort:24710 protocol:UDP]]   40.115.7.145
```

Congratulations, you have this project up and running!. You can type `kubectl delete dgsc simplenodejsudp-collection-example` to delete the sample application from your cluster.

## OpenArena

We have created a Docker container for the open source game [OpenArena](http://openarena.wikia.com/wiki/Main_Page). Here are the steps that you can use to try this game on your cluster.

### Necessary stuff to test OpenArena game

To test the project's installation using the OpenArena game, you should create a storage account to copy the OpenArena files. This will allow us to use the [Docker image](https://hub.docker.com/r/dgkanatsios/docker_openarena_k8s/) that we have built (source is on the `demos/openarena` folder). Our Docker image accesses the game files from a volume mount, on an Azure File share. Thus, main game files are not copied into each running Docker image but pulled dynamically on container creation. As you can understand, this makes for a Docker image that is smaller and faster to load, thus a smaller dedicated game server boot time.

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

Then, you can use this command to launch a DedicatedGameServerCollection with 5 OpenArena DedicatedGameServers.

```bash
kubectl create -f https://raw.githubusercontent.com/dgkanatsios/azuregameserversscalingkubernetes/master/artifacts/examples/openarena/dedicatedgameservercollection.yaml
```

Good luck and don't forget that you can open an [issue](https://github.com/dgkanatsios/azuregameserversscalingkubernetes/issues) in case you're stuck.