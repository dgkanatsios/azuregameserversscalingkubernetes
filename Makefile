# Go parameters
GOCMD=go
GOBUILD=CGO_ENABLED=0 GOOS=linux $(GOCMD) build -a -installsuffix cgo
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
VERSION=0.0.45
REGISTRY ?= docker.io

# set these two for remote e2e
REMOTE_DEBUG_CLUSTER_NAME=aksopenarena
REMOTE_DEBUG_CONFIG_FILENAME=config-openarena

# this one is for local e2e with kind (kubernetes in docker)
export KIND_CLUSTER_NAME=1

export APISERVER_NAME=dgkanatsios/aks_gaming_apiserver
export CONTROLLER_NAME=dgkanatsios/aks_gaming_controller
export TAG?=$(shell git rev-list HEAD --max-count=1 --abbrev-commit)


all: test build
deps:
		$(GOCMD) get -t -v ./...
buildremote: clean
		docker build -f ./cmd/apiserver/Dockerfile -t $(REGISTRY)/$(APISERVER_NAME):$(VERSION) .
		docker build -f ./cmd/controller/Dockerfile -t $(REGISTRY)/$(CONTROLLER_NAME):$(VERSION) .
		docker tag $(REGISTRY)/$(APISERVER_NAME):$(VERSION) $(REGISTRY)/$(APISERVER_NAME):latest
		docker tag $(REGISTRY)/$(CONTROLLER_NAME):$(VERSION) $(REGISTRY)/$(CONTROLLER_NAME):latest
pushremote:
		docker push $(REGISTRY)/$(APISERVER_NAME):$(VERSION)
		docker push $(REGISTRY)/$(CONTROLLER_NAME):$(VERSION)
		docker push $(REGISTRY)/$(APISERVER_NAME):latest
		docker push $(REGISTRY)/$(CONTROLLER_NAME):latest
test: 
		$(GOTEST) -v ./...
clean: 
		$(GOCLEAN)
		rm -f ./bin/apiserver
		rm -f ./bin/controller
travis: clean deps
		$(GOTEST) -v ./... -race -coverprofile=coverage.txt -covermode=atomic
authorsfile: ## Update the AUTHORS file from the git logs
		git log --all --format='%aN <%cE>' | sort -u > AUTHORS
createcrds:
		kubectl apply -f ./artifacts/crds
cleancrds:
		kubectl delete -f ./artifacts/crds

# local development and testing - make sure you have kubernetes-sigs/kind installed!
# you should run 'make builddockerlocal' before running 'deployk8slocal'
createcluster:
		kind create cluster
deletecluster:
		kind delete cluster
buildlocal:
		$(GOBUILD)  -o ./bin/apiserver ./cmd/apiserver
		$(GOBUILD)  -o ./bin/controller ./cmd/controller 
builddockerlocal: buildlocal
		docker build -f various/Dockerfile.apiserver.local -t $(APISERVER_NAME):$(TAG) . 
		docker build -f various/Dockerfile.controller.local -t $(CONTROLLER_NAME):$(TAG) .	
deployk8slocal: createcrds
		sed "s/%TAG%/$(TAG)/g" ./e2e/deploy.apiserver-controller.local.yaml | kubectl apply -f -
cleank8slocal: cleancrds
		sed "s/%TAG%/$(TAG)/g" ./e2e/deploy.apiserver-controller.local.yaml | kubectl delete -f -
e2elocal: test
		kubectl config use-context kubernetes-admin@kind-$(KIND_CLUSTER_NAME)
		./e2e/run.sh kind-config-$(KIND_CLUSTER_NAME) local

# remote building and deploying
buildremotedebug: clean
		docker build -f ./cmd/apiserver/Dockerfile -t $(REGISTRY)/$(APISERVER_NAME):$(TAG) .
		docker build -f ./cmd/controller/Dockerfile -t $(REGISTRY)/$(CONTROLLER_NAME):$(TAG) .
pushremotedebug:
		docker push $(REGISTRY)/$(APISERVER_NAME):$(TAG)
		docker push $(REGISTRY)/$(CONTROLLER_NAME):$(TAG)
deployk8sremotedebug: createcrds
		sed "s/%TAG%/$(TAG)/g" ./e2e/deploy.apiserver-controller.remote.yaml | kubectl apply -f -
cleank8sremotedebug: cleancrds
		sed "s/%TAG%/$(TAG)/g" ./e2e/deploy.apiserver-controller.remote.yaml | kubectl delete -f -
e2eremotedebug: test
		kubectl config use-context $(REMOTE_DEBUG_CLUSTER_NAME)
		./e2e/run.sh $(REMOTE_DEBUG_CONFIG_FILENAME) remote 