# Go parameters
GOCMD=go
GOBUILD=CGO_ENABLED=0 GOOS=linux $(GOCMD) build -a -installsuffix cgo
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
VERSION=0.0.36
REGISTRY ?= docker.io/dgkanatsios
TAG?=$(shell git rev-list HEAD --max-count=1 --abbrev-commit)
export TAG


all: test build
deps:
		$(GOCMD) get -t -v ./...
builddockerhub: clean
		docker build -f ./apiserver/Dockerfile -t $(REGISTRY)/aks_gaming_apiserver:$(VERSION) .
		docker build -f ./controller/Dockerfile -t $(REGISTRY)/aks_gaming_controller:$(VERSION) .
		docker tag $(REGISTRY)/aks_gaming_apiserver:$(VERSION) $(REGISTRY)/aks_gaming_apiserver:latest
		docker tag $(REGISTRY)/aks_gaming_controller:$(VERSION) $(REGISTRY)/aks_gaming_controller:latest
pushdockerhub:
		docker push docker.io/dgkanatsios/aks_gaming_apiserver:$(VERSION)
		docker push docker.io/dgkanatsios/aks_gaming_controller:$(VERSION)
		docker push docker.io/dgkanatsios/aks_gaming_apiserver:latest
		docker push docker.io/dgkanatsios/aks_gaming_controller:latest
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
		docker build -f various/Dockerfile.apiserver.local -t dgkanatsios/aks_gaming_apiserver:$(TAG) .	
		docker build -f various/Dockerfile.controller.local -t dgkanatsios/aks_gaming_controller:$(TAG) .	
deployk8slocal: buildlocal builddockerlocal
		sed "s/%TAG%/$(TAG)/g" ./artifacts/deploy.apiserver-controller.local.yaml | kubectl apply -f -
cleank8slocal:
		sed "s/%TAG%/$(TAG)/g" ./artifacts/deploy.apiserver-controller.local.yaml | kubectl delete -f -
