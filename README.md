[![Go Report Card](https://goreportcard.com/badge/github.com/dgkanatsios/azuregameserversscalingkubernetes)](https://goreportcard.com/report/github.com/dgkanatsios/azuregameserversscalingkubernetes)
[![Build Status](https://travis-ci.org/dgkanatsios/azuregameserversscalingkubernetes.svg?branch=master)](https://travis-ci.org/dgkanatsios/azuregameserversscalingkubernetes)
[![Software License](https://img.shields.io/badge/license-MIT-brightgreen.svg?style=flat-square)](LICENSE)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=flat-square)](http://makeapullrequest.com)
[![unofficial Google Analytics for GitHub](https://gaforgithub.azurewebsites.net/api?repo=AzureGameServersScalingKubernetes)](https://github.com/dgkanatsios/gaforgithub)
![](https://img.shields.io/badge/status-alpha-red.svg)

# AzureGameServersScalingKubernetes

Scaling dedicated game servers is hard. They're stateful, can't (well, shouldn't) be explicitly shut down (since players might be still enjoying their game) and, as a rule of thumb, their connection with the players must be of minimal latency, especially for real-time games.

This repository aims to provide a solution/guidance for managing containerized dedicated game servers on Azure Platform using the managed [Azure Kubernetes Service (AKS)](https://azure.microsoft.com/en-us/services/kubernetes-service/).

~ This is currently a work in progress. Not recommended for production use ~

## Documentation

- [Installation](docs/installation.md)
- [Kubernetes resources](docs/resources.md)
- [Controllers](docs/controllers.md)
- [FAQ](docs/FAQ.md)

## Architecture

We are using [Kubernetes Custom Resource Definitions (CRDs)](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/) to represent our dedicated game servers. Specifically, we have two core entities:

- **DedicatedGameServer** ([YAML](/artifacts/crds/dedicatedgameserver.yaml), [Go](/pkg/apis/azuregaming/v1alpha1/dedicatedgameserver.go)): represents the game server itself. Each DedicatedGameServer has a single corresponding child Pod which will run the Docker container for your game. You may find it referenced as DGS in the source code.
- **DedicatedGameServerCollection** ([YAML](/artifacts/crds/dedicatedgameservercollection.yaml), [Go](/pkg/apis/azuregaming/v1alpha1/dedicatedgameservercollection.go)): represents a collection of related DedicatedGameServers that will run the same Pod template and can optionally be scaled in/out within the collection.

This project contains 2 main components:

- **APIServer**: this is the API server for our project. It allows the game server to call some REST APIs to signal that the game server is running/failed and inform about the number of the active players currently playing the game 
- **Controller**: this contains custom [Kubernetes controllers](https://github.com/kubernetes/sample-controller) for our CRDs. These controllers will perform various activities on the system when DedicatedGameServerCollections or DedicatedGameServers are created or updated. Moreover, there is a controller (optionally started) that handles the autoscaling part on each DedicatedGameServerCollection

Both of these are created as Kubernetes deployments.

## API Server

The API Server exposes various HTTP operations, split into two categories. The first set contains methods that are to be called by the Dedicated Game Servers themselves to notify the API Server of various status changes whereas the second set contains methods that are to be called by an external interface (e.g. a matchmaker or a lobby service or just a system user) 

### Dedicated Game Server callable methods

The API Server exposes two HTTP methods.

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

### Methods that are to be called by an external interface

This set contains these HTTP operations:

- **/create**: will create a new DedicatedGameServer instance
- **/createcollection**: will create a new DedicatedGameServerCollection instance
- **/delete**: will delete a DedicatedGameServer instance
- **/running**: will return all the running DedicatedGameServer instances in JSON format

Moreover, on root URL (**/**) the API Server will return an HTML page that displays data from the `/running` endpoint, so it can easily be accessed by a web browser.

### Access code

All API methods are protected via a code, represented as string and kept in a Kubernetes Secred called `apiaccesscode` (created during project's installation). This code should be appended in the URL's query string via `code` GET variable in each and every call. The only method that does not require authentication by default is the `/running` one (although this behavior can be changed in the API Server command line arguments).

## Local development and building this project

We have tested the project locally with Docker for [Windows](https://docs.docker.com/docker-for-windows/)/[Mac](https://docs.docker.com/docker-for-mac/install/) and its Kubernetes support. You can use the `Makefile` we have to test and deploy the application.

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

### PodAutoscaler

Project contains an **experimental** Pod autoscaler controller. This autoscaler can scale DedicatedGameServer instances within a DedicatedGameServerCollections. Its use is optional and can be configured when you are deploying a DedicatedGameServerCollection resource. The autoscaler lives on the aks-gaming-controller executable and can be optionally enabled.
The decision about whether there should be a scaling activity is determined based on the `ActivePlayers` metric. We take into account that each DedicatedGameServer can hold a specific amount of players. If the sum of the active players on all the running servers of the DedicatedGameServerCollection is above a specified threshold (or below, for scale in activity), then the system is clearly in need of more DedicatedGameServer instances, so a scale out activity will occur, increasing the requested replicas of the DedicatedGameServerCollection by one. Moreover, there is a cooldown timeout so that a minimum amount of time will pass between two successive scaling activities.
Here you can see a configuration example, fields are self-explainable:

```yaml
# field of DedicatedGameServerCollection.Spec
podAutoScalerDetails:
  minimumReplicas: 5
  maximumReplicas: 10
  scaleInThreshold: 60
  scaleOutThreshold: 80
  enabled: true
  coolDownInMinutes: 5
  maxPlayersPerServer: 10
```

---
This is not an official Microsoft product.