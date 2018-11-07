[![Go Report Card](https://goreportcard.com/badge/github.com/dgkanatsios/azuregameserversscalingkubernetes)](https://goreportcard.com/report/github.com/dgkanatsios/azuregameserversscalingkubernetes)
[![Build Status](https://travis-ci.org/dgkanatsios/azuregameserversscalingkubernetes.svg?branch=master)](https://travis-ci.org/dgkanatsios/azuregameserversscalingkubernetes)
[![Software License](https://img.shields.io/badge/license-MIT-brightgreen.svg?style=flat-square)](LICENSE)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=flat-square)](http://makeapullrequest.com)
[![unofficial Google Analytics for GitHub](https://gaforgithub.azurewebsites.net/api?repo=AzureGameServersScalingKubernetes)](https://github.com/dgkanatsios/gaforgithub)
![](https://img.shields.io/badge/status-alpha-red.svg)
[![codecov](https://codecov.io/gh/dgkanatsios/azuregameserversscalingkubernetes/branch/master/graph/badge.svg)](https://codecov.io/gh/dgkanatsios/azuregameserversscalingkubernetes)

# Scaling Dedicated Game Servers on Azure Kubernetes Service

Scaling dedicated game servers is a hard problem. They're stateful (having the bulk of player action data stored in server RAM), can't be explicitly shut down (since players might be still enjoying their game) and, as a rule of thumb, their connection with the players must be of minimal latency, especially for real-time games.

This repository aims to provide a solution/guidance/building blocks for managing containerized dedicated game servers using the [Kubernetes](https://k8s.io) orchestrator on Azure using the managed [Azure Kubernetes Service (AKS)](https://azure.microsoft.com/en-us/services/kubernetes-service/).

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

We are using Kubernetes [Custom Resource Definition (CRDs)](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/) objects to represent our dedicated game server objects. If you don't know what Kubernetes is, check [here](https://kubernetes.io/docs/concepts/overview/what-is-kubernetes/) on the official documentation or watch [this](https://www.youtube.com/watch?v=4ht22ReBjno) great introductory video.

Specifically, we have two core entities in our project:

- **DedicatedGameServer** ([YAML](/artifacts/crds/dedicatedgameserver.yaml), [Go](/pkg/apis/azuregaming/v1alpha1/dedicatedgameserver.go)): it represents the game server itself. Each DedicatedGameServer has a single corresponding child Pod which will run the Docker container for your game. You may find it referenced as DGS in the source code.
- **DedicatedGameServerCollection** ([YAML](/artifacts/crds/dedicatedgameservercollection.yaml), [Go](/pkg/apis/azuregaming/v1alpha1/dedicatedgameservercollection.go)): it represents a collection/set of related DedicatedGameServers that will run the same Pod template and can optionally be scaled in/out within the collection. Dedicated Game Servers in the pod share some similarities in their launch environment, e.g. all of them could launch the same multiplayer map or the same type of game. So, you could have one collection for a "Capture the flag" mode of your game and another collection for a "Conquest" mode.

When you create a new DedicatedGameServerCollection definition file, these are the fields you need to declare:

- **replicas** (integer): number of DedicatedGameServer instances
- **portsToExpose** (array of integers): these are the ports that you want to be exposed in the Node/VM when the Pod is created. Each Pod you create will have >=1 containers, each container will have its *Ports* definition. If a port in this definition is included in the *portsToExpose* array, this port will be exposed in the Node/VM. Expose takes place by the creation of a **hostPort** value on the Pod's definition. This is a procedure that is managed exclusively by our project
- **template** (PodSpec): this is the actual Kubernetes Pod template

You can also take a look in the example files in the `artifacts/examples` folder.

### Components

This project contains 2 main components, both of which are by default created Kubernetes Deployments in the namespace 'dgs-system':

#### API Server

This is our projects API server (nothing to do with Kubernetes API Server). It contains two subcomponents, our project's API Server as well as a admission webhook.

##### API Server subcomponent

The API server component allows the game server to call some REST APIs to send status messages about the game server (like running/failed) and about the number of the active players currently playing the game. Specifically, the API Server exposes various HTTP operations, split into two categories.

The first set contains two HTTP REST APImethods that are to be called by the Dedicated Game Servers themselves.

- **/setactiveplayers**: This method allows the dedicated game server to notify the API Server about currently connected players.
Definition of the POST data that each DedicatedGameServer should send is:
```go
type ServerActivePlayers struct {
	ServerName  string `json:"serverName"`
	PlayerCount int    `json:"playerCount"`
}
```
- **/setserverstatus**: This methods allows the DedicatedGameServer to notify the API Server about its status.
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

All API methods are protected via an access code, represented as string and kept in a Kubernetes Secred called `apiaccesscode` (created during project's installation). This code should be appended in the URL's query string via `code` GET variable in each and every call. The only method that does not require authentication by default is the `/running` one (although this behavior can be changed in the API Server command line arguments).

##### Webhook subcomponent

The webhook component contains a Kubernetes [mutating admission webhook](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#admission-webhooks) which validates and modifies requests to the Kubernetes API Server concerning our CRDs. Specifically, it acts both as validating and a mutating admission webhook by executing these operations:

- It checks if the Pods specified in the DedicatedGameServerCollection template have a [Resources section with CPU/Memory requests and limits](https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/#resource-requests-and-limits-of-pod-and-container). If the containers in the Pod lack this information, the webhook will reject the submission
- It mutates the Pods so as to add [Pod Affinity](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#affinity-and-anti-affinity) information. This helps the Kubernetes scheduler group the DedicatedGameServer Pods in Nodes, instead of distributing them in the cluster.

#### Controller

This contains custom [Kubernetes controllers](https://github.com/kubernetes/sample-controller) for our CRDs. These controllers will perform various activities on the system when DedicatedGameServerCollections or DedicatedGameServers are created or updated. Moreover, there is an additional controller (optionally started, controlled by command-line argument) that handles the autoscaling part on each DedicatedGameServerCollection.

## Development and building

### Local development

We have tested the project locally with Docker for [Windows](https://docs.docker.com/docker-for-windows/)/[Mac](https://docs.docker.com/docker-for-mac/install/) and its Kubernetes support. You can use the `Makefile` we have to test and deploy the application. Supposing that `$KUBECONFIG` points to your local cluster, you could do 

```bash
make deployk8slocal
```

to deploy the cluster locally. Then, you can install the examples in the artifacts/examples folder to test. To remove everything from your local installation, run 

```bash
make cleank8slocal
```

### Remote development and pushing

To build the project and upload it to your Docker Container registry:

- change the `REGISTRY` variable on the Makefile
- run `./various/changeversion.sh OLDVERSION NEWVERSION` (after replacing *OLDVERSION* and *NEWVERSION* with your values, of course - *OLDVERSION* refers to the one you overwrote to in the Makefile, *NEWVERSION* refers to the new value). This changes the docker tag on the project's installation YAML files as well as in the Makefile
- run `make builddockerhub` and `make pushdockerhub` to build and push your Docker images
- check [installation document](docs/installation.md) for the next steps

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