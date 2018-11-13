[![Go Report Card](https://goreportcard.com/badge/github.com/dgkanatsios/azuregameserversscalingkubernetes)](https://goreportcard.com/report/github.com/dgkanatsios/azuregameserversscalingkubernetes)
[![Build Status](https://travis-ci.org/dgkanatsios/azuregameserversscalingkubernetes.svg?branch=master)](https://travis-ci.org/dgkanatsios/azuregameserversscalingkubernetes)
[![Software License](https://img.shields.io/badge/license-MIT-brightgreen.svg?style=flat-square)](LICENSE)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=flat-square)](http://makeapullrequest.com)
[![unofficial Google Analytics for GitHub](https://gaforgithub.azurewebsites.net/api?repo=AzureGameServersScalingKubernetes)](https://github.com/dgkanatsios/gaforgithub)
![](https://img.shields.io/badge/status-alpha-red.svg)
[![codecov](https://codecov.io/gh/dgkanatsios/azuregameserversscalingkubernetes/branch/master/graph/badge.svg)](https://codecov.io/gh/dgkanatsios/azuregameserversscalingkubernetes)

# Scaling multiplayer Dedicated Game Servers on Azure Kubernetes Service

Scaling Dedicated Game Servers is a hard problem (DGS). They're stateful (having the bulk of player action data stored in server RAM), can't be explicitly shut down (since players might be still enjoying their game) and, as a rule of thumb, their connection with the players must be of minimal latency, especially for real-time multiplayer games.

This repository aims to provide a solution/guidance/building blocks for managing containerized dedicated game servers using the [Kubernetes](https://k8s.io) orchestrator on Azure using the managed [Azure Kubernetes Service (AKS)](https://azure.microsoft.com/en-us/services/kubernetes-service/). You could potentially use parts of the project to scale memory-stateful workloads.

~ This is currently a work in progress. Not recommended for production use ~

## Documentation

- [Installation](docs/installation.md)
- [Kubernetes resources](docs/resources.md)
- [Controllers](docs/controllers.md)
- [Development and e2e testing](docs/development.md)
- [Dedicated Game Server Health](docs/dgshealth.md)
- [Autoscaling](docs/scaling.md)
- [FAQ](docs/FAQ.md)

## Architecture

### Kubernetes Custom Resource Definitions

First of all, if you don't know what Kubernetes is, check [here](https://kubernetes.io/docs/concepts/overview/what-is-kubernetes/) on the official documentation or watch [this](https://www.youtube.com/watch?v=4ht22ReBjno) great introductory video.

We are using Kubernetes [Custom Resource Definition (CRDs)](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/) objects to represent our dedicated game server objects. 

Specifically, we have two core entities in our project:

- **DedicatedGameServer** ([YAML](/artifacts/crds/dedicatedgameserver.yaml), [Go](/pkg/apis/azuregaming/v1alpha1/dedicatedgameserver.go)): this represents the multiplayer game server itself. Each DedicatedGameServer has a single corresponding child [Pod](https://kubernetes.io/docs/concepts/workloads/pods/pod/) which will run the container with your game server code. In the source code, you will find that it is also mentioned as *DGS*.
- **DedicatedGameServerCollection** ([YAML](/artifacts/crds/dedicatedgameservercollection.yaml), [Go](/pkg/apis/azuregaming/v1alpha1/dedicatedgameservercollection.go)): this represents a collection/set of related DedicatedGameServers that will run the same Pod template and can be scaled in/out within the collection. Dedicated Game Servers that are members of the same Collection share some similarities in their execution environment, e.g. all of them could launch the same multiplayer map or the same type of game. So, you could have one collection for a "Capture the flag" mode of your game and another collection for a "Conquest" mode. Or, a collection for players playing on map "X" and a collection for players playin on map "Y".

When you create a new DedicatedGameServerCollection definition file, these are the fields you need to declare:

- **replicas** (integer): number of requested DedicatedGameServer instances
- **portsToExpose** (array of integers): these are the ports that you want to be exposed in the [Node/VM](https://kubernetes.io/docs/concepts/architecture/nodes/) when the Pod is created. Each Pod you create will have >=1 number of containers. There, each container will have its own *Ports* definition. If a port in this definition is included in the *portsToExpose* array, this port will be publicly exposed in the Node/VM. This is accomplished by the creation of a **hostPort** value on the Pod's definition, a procedure that is managed exclusively by our solution
- **template** (PodSpec): this is the actual Kubernetes [Pod template](https://kubernetes.io/docs/concepts/workloads/pods/pod-overview/#pod-templates)

For examples, fell free to take a look in the example files in the `artifacts/examples` folder.

### Solution Components

This solution contains 2 main components, both of which are by default created as single instance Kubernetes [Deployments](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/) in the namespace **dgs-system**:

#### API Server components

This is our project's API server *(nothing to do with Kubernetes API Server)*. It contains two sub-components, our project's API Server as well as a Kubernetes [admission webhook](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#admission-webhooks).

##### API Server subcomponent

The API server subcomponent provides some REST APIs that can be called by either the game server or an external scheduling system (e.g. a lobby service or a matchmaker). There APIs are split into two categories.

The first category contains two HTTP methods that are to be called by the Dedicated Game Servers:

- **/setactiveplayers**: This method allows the dedicated game server to notify the API Server about currently connected players.
Definition of the POST data that each DedicatedGameServer should send is:
```go
type ServerActivePlayers struct {
	ServerName  string `json:"serverName"`
	PlayerCount int    `json:"playerCount"`
}
```
- **/setserverstatus**: This methods allows the DedicatedGameServer to notify the API Server about the status of the DedicatedGameServer process.
Definition of the POST data is:
```go
type ServerStatus struct {
	ServerName string `json:"serverName"`
	Status     string `json:"status"`
}
```

`status` field can have one of these four values:
- Creating
- Running
- MarkedForDeletion
- Failed

The second set contains these HTTP methods:

- **/create**: This will create a new DedicatedGameServerCollection instance
- **/delete**: This will delete a DedicatedGameServerCollection instance
- **/running**: This will return all the running DedicatedGameServer instances in JSON format

Moreover, if called on root URL (**/**) the API Server will return an HTML page that displays data from the `/running` endpoint, so it can easily be accessed by a web browser.

All API methods are protected via an access code, represented as string and kept in a [Kubernetes Secret](https://kubernetes.io/docs/concepts/configuration/secret/) called `apiaccesscode`. This is created during project's installation and should be passed in all method calls `code` GET parameter. The only method that does not require authentication by default is the `/running` one. However, this behavior can be changed in the API Server process command line arguments.

##### Webhook subcomponent

The webhook component contains a Kubernetes [mutating admission webhook](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#admission-webhooks) which validates and modifies requests to the Kubernetes API Server concerning our CRDs. Specifically, it acts both as validating and a mutating admission webhook by performing these two operations:

- It checks if the Pods specified in the DedicatedGameServerCollection template have a [Resources section with CPU/Memory requests and limits](https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/#resource-requests-and-limits-of-pod-and-container). If the containers in the Pod lack this information, the webhook will reject the submission
- It mutates the Pods so as to add [Pod Affinity](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#affinity-and-anti-affinity) information. This helps the Kubernetes scheduler group the DedicatedGameServer Pods in Nodes, instead of distributing them in the cluster.

#### Controller

This component contains custom [Kubernetes controllers](https://github.com/kubernetes/sample-controller) for our CRDs. These controllers will perform various activities on the system when DedicatedGameServerCollections or DedicatedGameServers are created or updated. Moreover, there is an additional controller that handles the autoscaling part on each DedicatedGameServerCollection. This controller, called DGSAutoScalerController is optionally started.

## Other

### Demos

In order to demonstrate this project, we've built a simple "echo" UDP server in Node.js and we've also adapted the [OpenArena](http://openarena.wikia.com/wiki/Main_Page) open source game so it can work with our solution. Both Dockerfiles are stored in the `demos` folder whereas the corresponding Kubernetes deployment YAML files are located in the `artifacts/examples` folder.

### Environment variables

Three environment variables are created on each dedicated game server pod, when it's created:

- SERVER_NAME: contains the name of the dedicated game server instance
- SET_ACTIVE_PLAYERS_URL: the API Server URL for setting active players. It should be used by the game server's code
- SET_SERVER_STATUS_URL: the API Server URL for setting dedicated game server status. It should be used by the game server's code

### Docker Hub Images

Images used for this project are hosted on Docker Hub:

- [OpenArena game sample](https://hub.docker.com/r/dgkanatsios/docker_openarena_k8s/)
- [A Node.js UDP echo server](https://hub.docker.com/r/dgkanatsios/simplenodejsudp/)
- [API Server](https://hub.docker.com/r/dgkanatsios/aks_gaming_apiserver/)
- [Controllers](https://hub.docker.com/r/dgkanatsios/aks_gaming_controller/)

---
This is not an official Microsoft product.