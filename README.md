[![Go Report Card](https://goreportcard.com/badge/github.com/dgkanatsios/AzureGameServersScalingKubernetes)](https://goreportcard.com/report/github.com/dgkanatsios/AzureGameServersScalingKubernetes)
[![Software License](https://img.shields.io/badge/license-MIT-brightgreen.svg?style=flat-square)](LICENSE)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=flat-square)](http://makeapullrequest.com)
[![unofficial Google Analytics for GitHub](https://gaforgithub.azurewebsites.net/api?repo=AzureGameServersScalingKubernetes)](https://github.com/dgkanatsios/gaforgithub)
![](https://img.shields.io/badge/status-prealpha-red.svg)

# AzureGameServersScalingKubernetes

Scaling dedicated game servers is hard. They're stateful, can't be shut down on demand (since players might be still enjoying their game) and, as a rule of thumb, their connection with the players must be of minimal network overhead and latency. This repo aims to provide a solution of managing containerized dedicated servers on Azure Platform using the managed Azure Kubernetes Service (AKS). It is based on Kubernetes Custom Resource Definition objects. 

~ This is currently a work in progress. ABSOLYTELY NOT RECOMMENDED FOR PRODUCTION USE ~

## Documentation

- [Installation](docs/installation.md)
- [FAQ](docs/FAQ.md)
- [TODO stuff](docs/TODO.md)
- [Kubernetes resources](docs/resources.md)

## Docker Hub Images

- [OpenArena for Kubernetes](https://hub.docker.com/r/dgkanatsios/docker_openarena_k8s/)
- [API Server](https://hub.docker.com/r/dgkanatsios/docker_openarena_k8s_apiserver/)
- [Custom Kubernetes Controller](https://hub.docker.com/r/dgkanatsios/docker_openarena_k8s_controller/)