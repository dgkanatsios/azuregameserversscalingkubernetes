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
az aks create --resource-group $AKS_RESOURCE_GROUP --name $AKS_NAME --node-count 1 --ssh-key-value ~/.ssh/id_rsa.pub --node-vm-size Standard_A1_v2 --kubernetes-version 1.11.3 --enable-rbac # this command will take some time...
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

## Assigning Public IPs to the existing Nodes in the cluster

As of now, AKS Nodes don't get a Public IP by default (even though you could use [acs-engine](https://github.com/Azure/acs-engine) to create a self-managed K8s cluster that supports that). To assign Public IP to a Node/VM, you can find the Resource Group where the AKS resources are installed on the [portal](https://portal.azure.com) (it should have a name like `MC_resourceGroupName_AKSName_location`). Then, you can follow the instructions [here](https://blogs.technet.microsoft.com/srinathv/2018/02/07/how-to-add-a-public-ip-address-to-azure-vm-for-vm-failed-over-using-asr/) to create a new Public IP and assign it to the Node/VM. For more information on Public IPs for VM NICs, see [this document](https://docs.microsoft.com/azure/virtual-network/virtual-network-network-interface-addresses). 

Alternatively, you can use [this](https://github.com/dgkanatsios/AksNodePublicIPController) project which will take care of
- Creating and assigning Public IPs to existing Nodes
- Creating and assigning Public IPs to new Nodes, e.g. in case of a cluster scale out
- Deleting Public IPs for Nodes that get removed from the cluster, e.g. cluster scale in

You can check its [instructions](https://github.com/dgkanatsios/AksNodePublicIPController/blob/master/README.md), setup is pretty easy. 

## CRD and APIServer/Controllers installation

First of all, create a Kubernetes secret that will hold the access code for the API Server's endpoints:
```bash
# use a code that will be kept secret
kubectl create secret generic apiaccesscode --from-literal=code=YOUR_CODE_HERE
```

Then, create the DedicatedGameServer Custom Resource Definition:

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

## Testing with Node.js demo app (an echo HTTP server)

### Creation of DedicatedGameServerCollection

Use this command to create a collection of DedicatedGameServers. The 'game' that will be created is the simple Node.js echo app, which source code is in `demos/simplenodejsudp` folder. This collection will create 5 DedicatedGameServers.

```bash
kubectl apply -f https://raw.githubusercontent.com/dgkanatsios/azuregameserversscalingkubernetes/master/artifacts/examples/simplenodejsudp/dedicatedgameservercollection.yaml
```

If everything works good, 5 instances will be created. Type `kubectl get dgsc` to see the DedicatedGameCollection as well as its status

```
NAME              REPLICAS   AVAILABLE   DGSCOLHEALTH   PODCOLLECTIONSTATE
simplenodejsudp   5          5           Healthy        Running
```

If you don't see "Running" and "Healthy" in the beginning, wait a few minutes and try again. Remember that the flow of events is:

- DedicatedGameServerCollection will create 5 DedicatedGameServers
- Each DedicatedGameServer will create a single Pod
- Kubernetes will pull the Docker image for the Pod and start it
- As soon as Pod is running, "Running" state will be reported in its parent DedicatedGameServer
- When the game server begins executing, it should report "Healthy" health state to the API Server
- The DedicatedGameServerController will have DGSCOLHEALTH equal to "Healthy" state only if all DedicatedGameServers are "Healthy". Same applies to PodCollectionState for "Running" value.

Now, try `kubectl get dgs` to see the statuses of each individual DedicatedGameServer. As mentioned, all should be "Healthy" and "Running".

```
NAME                    PLAYERS   DGSSTATE   PODPHASE   HEALTH    PORTS                                                    PUBLICIP        MFD
simplenodejsudp-gamng   0         Idle       Running    Healthy   [map[hostPort:28682 protocol:UDP containerPort:22222]]   13.73.179.116   false
simplenodejsudp-rpzio   0         Idle       Running    Healthy   [map[containerPort:22222 hostPort:29041 protocol:UDP]]   13.73.179.116   false
simplenodejsudp-wdosf   0         Idle       Running    Healthy   [map[hostPort:24598 protocol:UDP containerPort:22222]]   13.73.179.116   false
simplenodejsudp-wxkzm   0         Idle       Running    Healthy   [map[containerPort:22222 hostPort:29430 protocol:UDP]]   13.73.179.116   false
simplenodejsudp-xjaji   0         Idle       Running    Healthy   [map[containerPort:22222 hostPort:24317 protocol:UDP]]   13.73.179.116   false
```

Here you can also see ActivePlayers, assigned ports, PublicIP and MarkedForDeletion (MFD) info. Let's try to connect to one of them to test our installation. To do that, we'll use the netcat command. Let's try connect to the first DedicatedGameServer. The Node's IP is 13.73.179.116 (change it accordingly) whereas the assigned port is 28682. We're using the [netcat](https://en.wikipedia.org/wiki/Netcat) utility with a UDP connection, thus the *-u* parameter.

```bash
nc -u 13.73.179.116 28682
```

Now, if everything goes well, you can type whatever you like and the server will echo the message back:

```
hello
simplenodejsudp-collection-example-gamng-vhxhr says: hello
```

The demo app supports two extra commands for setting active players and server status. 

- Setting Active Players: If you write `players|3`, then the demo app will send a message to the project's API Server that there are 5 connected players.
- Setting DedicatedGameServer state: If you write `status|Running`, then the demo app will send a message to the project's API Server that its state is *Running*.
- Setting DedicatedGameServer health: If you write `health|Healthy`, then the demo app will send a message to the project's API Server that its health is *Healthy*.
- Setting DedicatedGameServer MarkedForDeletion state: If you write `markedfordeletion|true`, then the demo app will send a message to the project's API Server that its MarkedForDeletion state is *true*.

You can use Ctrl-C (or Cmd-C) to disconnect from the demo app.

Before we proceed, feel free to check the running pods as well:

```bash
kubectl get pods
```

```
NAME                          READY   STATUS    RESTARTS   AGE
simplenodejsudp-gamng-fmpai   1/1     Running   0          25m
simplenodejsudp-rpzio-arduj   1/1     Running   0          25m
simplenodejsudp-wdosf-dnrjs   1/1     Running   0          25m
simplenodejsudp-wxkzm-kikkj   1/1     Running   0          25m
simplenodejsudp-xjaji-aoldx   1/1     Running   0          25m
```

Those pods host our game server containers and are children to the DedicatedGameServers.

Now it's a good time to check our web frontend. Type `kubectl get svc -n dgs-system` to see the available Kubernetes services.

```
NAME                       TYPE           CLUSTER-IP     EXTERNAL-IP   PORT(S)        AGE
aks-gaming-apiserver       LoadBalancer   10.0.156.246   104.214.226.21     80:31186/TCP   27m
aks-gaming-webhookserver   ClusterIP      10.0.213.246   <none>        443/TCP        27m
```

Grap the External IP of the *aks-gaming-apiserver* Service and paste it in your web browser of choice. You should see a list with all the "Healthy" DedicatedGameServers.

### Scaling

Let's scale out our DedicatedGameServerCollection to 8 replicas.

```bash
kubectl scale dgsc simplenodejsudp --replicas=8
```

If everything goes well, eventually you will have 8 available replicas.

```bash
kubectl get dgsc
```

```
NAME              REPLICAS   AVAILABLE   DGSCOLHEALTH   PODCOLLECTIONSTATE
simplenodejsudp   8          8           Healthy        Running

```

Great! Let's trick our system so that all DedicatedGameServers have 5 active players. Normally, each DedicatedGameServer would have to call the respective APIServer REST method to set the number of active players.

```bash
# get DGS names
dgs=`kubectl get dgs -l DedicatedGameServerCollectionName=simplenodejsudp | cut -d ' ' -f 1 | sed 1,1d`
# update DGS.Spec.ActivePlayers
kubectl patch dgs $dgs -p '[{ "op": "replace", "path": "/status/activePlayers", "value": 5 },]' --type='json'
```

Let's scale our DedicatedGameServerCollection to 6 replicas

```bash
kubectl scale dgsc simplenodejsudp --replicas=6
```

Use the following command to see that DedicatedGameServerCollection has 6 available replicas

```bash
kubectl get dgsc
```

```
NAME              REPLICAS   AVAILABLE   DGSCOLHEALTH   PODCOLLECTIONSTATE
simplenodejsudp   6          6           Healthy        Running
```

However, there are still 8 DedicatedGameServers on our cluster. Check them out, including their labels

```bash
kubectl get dgs --show-labels
```

```
NAME                    PLAYERS   DGSSTATE   PODPHASE   HEALTH    PORTS                                                    PUBLICIP        MFD     LABELS
simplenodejsudp-gamng   5         Idle       Running    Healthy   [map[protocol:UDP containerPort:22222 hostPort:28682]]   13.73.179.116   false   DedicatedGameServerCollectionName=simplenodejsudp
simplenodejsudp-qguma   5         Idle       Running    Healthy   [map[containerPort:22222 hostPort:26522 protocol:UDP]]   13.73.179.116   false   DedicatedGameServerCollectionName=simplenodejsudp
simplenodejsudp-rpzio   5         Idle       Running    Healthy   [map[hostPort:29041 protocol:UDP containerPort:22222]]   13.73.179.116   false   DedicatedGameServerCollectionName=simplenodejsudp
simplenodejsudp-ssujb   5         Idle       Running    Healthy   [map[protocol:UDP containerPort:22222 hostPort:20528]]   13.73.179.116   true    OriginalDedicatedGameServerCollectionName=simplenodejsudp
simplenodejsudp-tpxpa   5         Idle       Running    Healthy   [map[containerPort:22222 hostPort:21715 protocol:UDP]]   13.73.179.116   false   DedicatedGameServerCollectionName=simplenodejsudp
simplenodejsudp-wdosf   5         Idle       Running    Healthy   [map[hostPort:24598 protocol:UDP containerPort:22222]]   13.73.179.116   false   DedicatedGameServerCollectionName=simplenodejsudp
simplenodejsudp-wxkzm   5         Idle       Running    Healthy   [map[containerPort:22222 hostPort:29430 protocol:UDP]]   13.73.179.116   true    OriginalDedicatedGameServerCollectionName=simplenodejsudp
simplenodejsudp-xjaji   5         Idle       Running    Healthy   [map[containerPort:22222 hostPort:24317 protocol:UDP]]   13.73.179.116   false   DedicatedGameServerCollectionName=simplenodejsudp
```

As you can see, 2 of them have the field MarkedForDeletion set to true and do not belong to the DedicatedGameServerCollection anymore. Still, they are not deleted, since there are players enjoying the game! Let's update (well, trick) these two game servers so that the system thinks that the game has finished and the players have left the server.

*Make sure to change the value of dgs2 variable with the names of your DedicatedGameServers that are MarkedForDeletion*

```bash
dgs2="simplenodejsudp-ssujb simplenodejsudp-wxkzm"
# update DGS.Spec.ActivePlayers
kubectl patch dgs $dgs2 -p '[{ "op": "replace", "path": "/status/activePlayers", "value": 0 },]' --type='json'
```

Now, if you run `kubectl get dgs` you will see that the two MarkedForDeletion servers have disappeared, since the players that were connected to them have left the game and disconnected from the server.

```bash
NAME                    PLAYERS   DGSSTATE   PODPHASE   HEALTH    PORTS                                                    PUBLICIP        MFD
simplenodejsudp-gamng   5         Idle       Running    Healthy   [map[containerPort:22222 hostPort:28682 protocol:UDP]]   13.73.179.116   false
simplenodejsudp-qguma   5         Idle       Running    Healthy   [map[containerPort:22222 hostPort:26522 protocol:UDP]]   13.73.179.116   false
simplenodejsudp-rpzio   5         Idle       Running    Healthy   [map[protocol:UDP containerPort:22222 hostPort:29041]]   13.73.179.116   false
simplenodejsudp-tpxpa   5         Idle       Running    Healthy   [map[containerPort:22222 hostPort:21715 protocol:UDP]]   13.73.179.116   false
simplenodejsudp-wdosf   5         Idle       Running    Healthy   [map[containerPort:22222 hostPort:24598 protocol:UDP]]   13.73.179.116   false
simplenodejsudp-xjaji   5         Idle       Running    Healthy   [map[containerPort:22222 hostPort:24317 protocol:UDP]]   13.73.179.116   false
```

Congratulations, you have this project up and running!. You can type `kubectl delete dgsc simplenodejsudp` to delete the sample application from your cluster.

## OpenArena

We have created a Docker container for the open source game [OpenArena](http://openarena.wikia.com/wiki/Main_Page). Here are the steps that you can use to try this game on your cluster.

### Necessary stuff to test OpenArena game

To test the project's installation using the OpenArena game, you should create a storage account to copy the OpenArena asset files. This will allow us to use the [Docker image](https://hub.docker.com/r/dgkanatsios/docker_openarena_k8s/) that we have built (source is on the `demos/openarena` folder). Our Docker image accesses the game files from a volume mount, on an Azure File share. So, main game files are not copied into each running Docker image but pulled dynamically on container creation. As you can understand, this makes for a Docker image that is smaller and faster to load.

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

Don't forget that you can open an [issue](https://github.com/dgkanatsios/azuregameserversscalingkubernetes/issues) in case you need any help!