# Local development and e2e testing

For local development and testing we have been using project [Kind - Kubernetes IN Docker](https://github.com/kubernetes-sigs/kind). We have been running our e2e tests there as well. To use it
- install Docker
- [install kind](https://github.com/kubernetes-sigs/kind#installation-and-usage)
- `kind create cluster`
- `make e2e`
- (when you're done) `kind delete cluster`

You can also run it on Minikube or Docker for Mac/Windows