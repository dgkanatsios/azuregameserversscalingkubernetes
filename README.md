[![Go Report Card](https://goreportcard.com/badge/github.com/dgkanatsios/azuregameserversscalingkubernetes)](https://goreportcard.com/report/github.com/dgkanatsios/azuregameserversscalingkubernetes)
[![Build Status](https://travis-ci.org/dgkanatsios/azuregameserversscalingkubernetes.svg?branch=master)](https://travis-ci.org/dgkanatsios/azuregameserversscalingkubernetes)
[![Software License](https://img.shields.io/badge/license-MIT-brightgreen.svg?style=flat-square)](LICENSE)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=flat-square)](http://makeapullrequest.com)
[![unofficial Google Analytics for GitHub](https://gaforgithub.azurewebsites.net/api?repo=AzureGameServersScalingKubernetes)](https://github.com/dgkanatsios/gaforgithub)
![](https://img.shields.io/badge/status-alpha-red.svg)

# AzureGameServersScalingKubernetes

Scaling dedicated game servers is hard. They're stateful, can't (well, shouldn't) be explicitly shut down (since players might be still enjoying their game) and, as a rule of thumb, their connection with the players must be of minimal latency, especially for real-time games.

This repository aims to provide a solution for managing containerized dedicated game servers on Azure Platform using the managed [Azure Kubernetes Service (AKS)](https://azure.microsoft.com/en-us/services/kubernetes-service/).

~ This is currently a work in progress. ABSOLYTELY NOT RECOMMENDED FOR PRODUCTION USE ~

## Documentation

- [Installation](docs/installation.md)
- [FAQ](docs/FAQ.md)
- [Kubernetes resources](docs/resources.md)
- [Controllers](docs/controllers.md)

## Architecture

We are using [Kubernetes Custom Resource Definitions (CRDs)](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/) to represent our dedicated game servers. Specifically, we have two core entities:

- DedicatedGameServer ([YAML](/artifacts/crds/dedicatedgameserver.yaml), [Go](/pkg/apis/azuregaming/v1alpha1/dedicatedgameserver.go)): represents the game server itself. You may find referenced as DGS in the source code. Each DedicatedGameServer has a corresponding Pod which will run the Docker container for your game. We are using Kubernetes `hostPort` to run the dedicated game servers. Ports are bound automatically via a custom portregistry implementation.
- DedicatedGameServerCollection ([YAML](/artifacts/crds/dedicatedgameservercollection.yaml), [Go](/pkg/apis/azuregaming/v1alpha1/dedicatedgameservercollection.go)): represents a collection of related DedicatedGameServers. These game servers can be scaled in/out, share a common container image and are created as a set/collection.

Project is composed of 2 main components:

- **APIServer**: this is the API server for our project, allows the game server to call some basic REST APIs to signal that the game server is running/failed and inform about the number of the active players
- **controller**: this contains custom [Kubernetes controllers](https://github.com/kubernetes/sample-controller) for our CRDs. These controllers will perform various activities on the system when DedicatedGameServerCollections or DedicatedGameServers are created

## API Server

The API Server exposes various HTTP operations, split into two categories. The first set contains methods that are to be called by the Dedicated Game Servers themselves to notify the API Server of various status changes whereas the second set contains methods that are to be called by an external interface (e.g. a matchmaker or a lobby service or just a system user) 

### Dedicated Game Server callable methods

The API Server currently exposes two HTTP methods.

- **/setactiveplayers**: This method allows the dedicated game server to notify the API Server about currently connected players.
Definition of the POST data is:
```go
type ServerActivePlayers struct {
	ServerName  string `json:"serverName"`
	PlayerCount int    `json:"playerCount"`
}
```
- **/setserverstatus**: This methods allows the dedicated game server to notify the API Server about its status.
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
- **/running**: will return all the running DedicatedGameServer instances

Moreover, on root URL (**/**) the API Server will return an HTML page that displays data from the `/running` endpoint, so it can easily be accessed by a web browser.

### Access code

All API methods are protected via a simple code, represented as string and kept in a Kubernetes secred called `apiaccesscode` (created during installation). This code is appended in the URL's query string via `code` GET variable. The only method that does not require authentication by default is the `/running` one (although this behavior can be changed in the API Server command line arguments).

## Other

### Demos

In order to demonstrate this project, we've built a simple echo UDP server in Node.js and we've modified the [OpenArena](http://openarena.wikia.com/wiki/Main_Page) open source game to work with it. Both Docker image sources are stored in the `demos` folder whereas the corresponding Kubernetes deployment YAML files are located in the `artifacts/examples` folder.

### Environment variables

Three environment variables are created on each dedicated game server pod, when it's created:

- SERVER_NAME: contains the name of the dedicated game server instance
- SET_ACTIVE_PLAYERS_URL: the API Server URL for setting active players
- SET_SERVER_STATUS_URL: the API Server URL for setting dedicated game server status

### Docker Hub Images

Various images used for this project are hosted on Docker Hub

- [OpenArena game sample, build for this project](https://hub.docker.com/r/dgkanatsios/docker_openarena_k8s/)
- [A simple Node.js UDP echo server](https://hub.docker.com/r/dgkanatsios/simplenodejsudp/)
- [API Server](https://hub.docker.com/r/dgkanatsios/aks_gaming_apiserver/)
- [Custom Kubernetes Controller](https://hub.docker.com/r/dgkanatsios/aks_gaming_controller/)

### PodAutoscaler

Project contains an **experimental** Pod autoscaler controller. This autoscaler can scale individual DedicatedGameServerCollections and you optionally can configure it when deploying your DedicatedGameServerCollection resource. This autoscaler (if enabled) will scale in/out the DedicatedGameServers in the collection it belongs to. The scaling is determined according to the `ActivePlayers` metric. We take into account that each DedicatedGameServer can hold a specific amount of players. If the specified threshold is surpassed, then we are clearly in need of more DedicatedGameServer instances. Here you can see a configuration example:

```yaml
autoScalerDetails:
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