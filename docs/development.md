# Development

## Local development and e2e testing

For local development and testing we have been using project [Kind - Kubernetes IN Docker](https://github.com/kubernetes-sigs/kind) to run a local Kubernetes environment. To use it on your local workstation:
- install Docker (if not already installed)
- [install kind](https://github.com/kubernetes-sigs/kind#installation-and-usage) using `go get sigs.k8s.io/kind`
- `make createcluster` to create a local cluster using
- `make builddockerlocal` to build Docker images locally
- `make e2e` to run the e2e tests as well as the unit tests
- `make deletecluster`

You can also run the tests on Minikube or Docker for Mac/Windows. Don't forget to `go get github.com/stretchr/testify/assert` as it's needed for testing!

## Upload to container registry

To build and upload images to your container registry of choice, you should customize the Makefile with the necessary details (specifically image names and registry URL). Then, you can run the following commands:

- `./various/changeversion.sh oldVersion newVersion` to update version in the Makefile and in the deployment YAML files
- `make buildremote` to build the images
- `make pushremote` to push the images to the container registry