# Go parameters
GOCMD=go
GOBUILD=CGO_ENABLED=0 GOOS=linux $(GOCMD) build -a -installsuffix cgo
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
VERSION=0.0.37
REGISTRY ?= docker.io
APISERVER_NAME=dgkanatsios/aks_gaming_apiserver
CONTROLLER_NAME=dgkanatsios/aks_gaming_controller
TAG?=$(shell git rev-list HEAD --max-count=1 --abbrev-commit)
export TAG


all: test build
deps:
		$(GOCMD) get -t -v ./...
builddockerhub: clean
		docker build -f ./apiserver/Dockerfile -t $(REGISTRY)/$(APISERVER_NAME):$(VERSION) .
		docker build -f ./controller/Dockerfile -t $(REGISTRY)/$(CONTROLLER_NAME):$(VERSION) .
		docker tag $(REGISTRY)/$(APISERVER_NAME):$(VERSION) $(REGISTRY)/$(APISERVER_NAME):latest
		docker tag $(REGISTRY)/$(CONTROLLER_NAME):$(VERSION) $(REGISTRY)/$(CONTROLLER_NAME):latest
pushdockerhub:
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

#local development
buildlocal:
		$(GOBUILD)  -o ./bin/apiserver ./apiserver/cmd/apiserver
		$(GOBUILD)  -o ./bin/controller ./controller/cmd/controller
builddockerlocal: buildlocal
		docker build -f various/Dockerfile.apiserver.local -t $(APISERVER_NAME):$(TAG) .	
		docker build -f various/Dockerfile.controller.local -t $(CONTROLLER_NAME):$(TAG) .	
deployk8slocal: buildlocal builddockerlocal
		kubectl apply -f ./artifacts/crds
		sed "s/%TAG%/$(TAG)/g" ./artifacts/deploy.apiserver-controller.local.yaml | kubectl apply -f -
cleank8slocal:
		kubectl delete -f ./artifacts/crds
		sed "s/%TAG%/$(TAG)/g" ./artifacts/deploy.apiserver-controller.local.yaml | kubectl delete -f -
