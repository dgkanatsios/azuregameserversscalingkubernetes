#!/bin/bash

# custom script for e2e testing

set -o errexit # script exits when a command fails == set -e
set -o nounset # script exits when tries to use undeclared variables == set -u
#set -o xtrace # trace what's executed == set -x
set -o pipefail

export KIND_CONTAINER_NAME="kind-${KIND_CLUSTER_NAME}-control-plane"

#https://stackoverflow.com/questions/59895/getting-the-source-directory-of-a-bash-script-from-within
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null && pwd )"

"${DIR}/build_images.sh"

echo "-----Running Go tests-----"
make -C ${DIR}/.. test 

echo "-----Compiling, building and deploying to local Kubernetes cluster-----"
make -C ${DIR}/.. deployk8slocal

echo "-----Waiting for APIServer and Controller deployments-----"
${DIR}/wait-for-deployment.sh -n dgs-system aks-gaming-apiserver
${DIR}/wait-for-deployment.sh -n dgs-system aks-gaming-controller

echo "-----Deploying simplenodejsudp collection-----"
kubectl create -f ${DIR}/../artifacts/examples/simplenodejsudp/dedicatedgameservercollection.yaml

echo "-----Running Go DGSTester-----"
RUN_IN_K8S=false go run ${DIR}/dgstester/*.go

echo "-----Cleaning up-----"
make -C ${DIR}/.. cleank8slocal