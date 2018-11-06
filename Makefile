# Go parameters
GOCMD=go
GOBUILD=CGO_ENABLED=0 GOOS=linux $(GOCMD) build -a -installsuffix cgo
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
VERSION=0.0.43
REGISTRY ?= docker.io
APISERVER_NAME=dgkanatsios/aks_gaming_apiserver
CONTROLLER_NAME=dgkanatsios/aks_gaming_controller
TAG?=$(shell git rev-list HEAD --max-count=1 --abbrev-commit)
export TAG

KIND_CLUSTER_NAME=1
KUBECONFIG_LOCAL=~/.kube/kind-config-${KIND_CLUSTER_NAME}

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

# local development and testing - make sure you have kubernetes-sigs/kind installed!
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
# you should run 'make builddockerlocal' before running 'deployk8slocal'
deployk8slocal: 
		KUBECONFIG=$(KUBECONFIG_LOCAL) kubectl apply -f ./artifacts/crds
		sed "s/%TAG%/$(TAG)/g" ./artifacts/deploy.apiserver-controller.local.yaml | KUBECONFIG=$(KUBECONFIG_LOCAL) kubectl apply -f -
cleank8slocal:
		KUBECONFIG=$(KUBECONFIG_LOCAL) kubectl delete -f ./artifacts/crds
		sed "s/%TAG%/$(TAG)/g" ./artifacts/deploy.apiserver-controller.local.yaml | KUBECONFIG=$(KUBECONFIG_LOCAL) kubectl delete -f -
.PHONY: e2e
e2e:
		KIND_CLUSTER_NAME=${KIND_CLUSTER_NAME} CONTROLLER_NAME=$(CONTROLLER_NAME) \
		 APISERVER_NAME=$(APISERVER_NAME) TAG=$(TAG) KUBECONFIG=$(KUBECONFIG_LOCAL) ./e2e/run.sh
