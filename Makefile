# Go parameters
GOCMD=go
GOBUILD=CGO_ENABLED=0 GOOS=linux $(GOCMD) build -a -installsuffix cgo
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
TAG?=$(shell git rev-list HEAD --max-count=1 --abbrev-commit)
export TAG

all: test build
build:
		$(GOBUILD)  -o ./apiserver/cmd/apiserver/apiserver ./apiserver/cmd/apiserver
		$(GOBUILD)  -o ./controller/cmd/controller/controller ./controller/cmd/controller
builddocker: build
		docker build -f various/Dockerfile.apiserver.local -t dgkanatsios/aks_gaming_apiserver:$(TAG) .	
		docker build -f various/Dockerfile.controller.local -t dgkanatsios/aks_gaming_controller:$(TAG) .	
buildk8s: build builddocker
		sed "s/%TAG%/$(TAG)/g" ./artifacts/deploy.apiserver-controller.local.yaml | kubectl apply -f -
deployk8s:
		sed "s/%TAG%/$(TAG)/g" ./artifacts/deploy.apiserver-controller.local.yaml | kubectl apply -f -
cleank8s:
		sed "s/%TAG%/$(TAG)/g" ./artifacts/deploy.apiserver-controller.local.yaml | kubectl delete -f -
test: 
		$(GOTEST) -v ./...
clean: 
		$(GOCLEAN)
		rm -f ./apiserver/cmd/apiserver/apiserver
		rm -f ./controller/cmd/controller/controller