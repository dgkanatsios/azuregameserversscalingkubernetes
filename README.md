[![Go Report Card](https://goreportcard.com/badge/github.com/dgkanatsios/AzureGameServersScalingKubernetes)](https://goreportcard.com/report/github.com/dgkanatsios/AzureGameServersScalingKubernetes)
[![Software License](https://img.shields.io/badge/license-MIT-brightgreen.svg?style=flat-square)](LICENSE)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=flat-square)](http://makeapullrequest.com)
[![unofficial Google Analytics for GitHub](https://gaforgithub.azurewebsites.net/api?repo=AzureGameServersScalingKubernetes)](https://github.com/dgkanatsios/gaforgithub)
![](https://img.shields.io/badge/status-prealpha-red.svg)

# AzureGameServersScalingKubernetes

Scaling dedicated game servers is hard. They're stateful, can't (well, shouldn't) be explicitly shut down (since players might be still enjoying their game) and, as a rule of thumb, their connection with the players must be of minimal latency, especially for real-time games.

This repository aims to provide a solution for managing containerized dedicated game servers on Azure Platform using the managed [Azure Kubernetes Service (AKS)](https://azure.microsoft.com/en-us/services/kubernetes-service/).

~ This is currently a work in progress. ABSOLYTELY NOT RECOMMENDED FOR PRODUCTION USE ~

## Documentation

- [Installation](docs/installation.md)
- [FAQ](docs/FAQ.md)
- [Kubernetes resources](docs/resources.md)

## Architecture

We are using [Kubernetes Custom Resource Definitions (CRDs)](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/) to represent our game server entities. Specifically, we have two core entities:

- DedicatedGameServer ([YAML](/artifacts/crds/dedicatedgameserver.yaml), [Go](/pkg/apis/azuregaming/v1alpha1/dedicatedgameserver.go)): represents the game server itself. You may find referenced as DGS in the source code. Each DedicatedGameServer has a corresponding Pod which will run the Docker container for your game. We are using Kubernetes `hostPort` to run the dedicated game servers. Ports are bound automatically via a custom portregistry object.
- DedicatedGameServerCollection ([YAML](/artifacts/crds/dedicatedgameservercollection.yaml), [Go](/pkg/apis/azuregaming/v1alpha1/dedicatedgameservercollection.go)): represents a collection of related DedicatedGameServers. These game servers can be scaled in/out, share a common container image and are created simultaneously.

Project is composed of 3 Docker images:

- OpenArena: contains the demo game server, from open source game [OpenArena](http://openarena.wikia.com/wiki/Main_Page)
- APIServer: this is the API server for our project, allows the game server to call some basic REST APIs to signal that the game server is running/failed and inform about the number of the active players
- controller: this contains custom [Kubernetes controllers](https://github.com/kubernetes/sample-controller) for our CRDs. These controllers will perform various activities on the system when DedicatedGameServerCollections or DedicatedGameServers are created

### Docker Hub Images

Beforementioned images are hosted on Docker Hub

- [OpenArena for Kubernetes](https://hub.docker.com/r/dgkanatsios/docker_openarena_k8s/)
- [API Server](https://hub.docker.com/r/dgkanatsios/aks_gaming_apiserver/)
- [Custom Kubernetes Controller](https://hub.docker.com/r/dgkanatsios/aks_gaming_controller/)

### Autoscaler

Project contains an **experimental** Pod autoscaler, you can find it in the AutoScalerController file and you can configure it when deploying your DedicatedGameServerCollection resource. This autoscaler (if enabled) will scale in/out the DedicatedGameServers in the collection based on the `ActivePlayers` metric. Each game server can hold a specific amount of players, if more of them are lining up in our matchmaking server, then we are clearly in need of more DedicatedGameServer instances (and their corresponding Pods).


This is not an official Microsoft product.