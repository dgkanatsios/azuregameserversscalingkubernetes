#!/bin/bash

# custom script for e2e testing

set -o errexit # script exits when a command fails == set -e
set -o nounset # script exits when tries to use undeclared variables == set -u
#set -o xtrace # trace what's executed == set -x (useful for debugging)
set -o pipefail

#https://stackoverflow.com/questions/59895/getting-the-source-directory-of-a-bash-script-from-within
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null && pwd )"

KUBECONFIG_FILE=$1
BUILD=${2:-remote} # setting a default value for $BUILD

if [ "$BUILD" = "local" ]; then
  kind load docker-image ${APISERVER_NAME}:"${TAG}"
  kind load docker-image ${CONTROLLER_NAME}:"${TAG}"
fi

function finish {
  echo "-----Cleaning up-----"
  if [ "$BUILD" = "local" ]; then
    make -C ${DIR}/.. cleank8slocal
  else
    make -C ${DIR}/.. cleank8sremotedebug
  fi
}

trap finish EXIT

echo "-----Compiling, building and deploying to local Kubernetes cluster-----"
if [ "$BUILD" = "local" ]; then
  make -C ${DIR}/.. deployk8slocal
else
  make -C ${DIR}/.. deployk8sremotedebug
fi

echo "-----Waiting for APIServer and Controller deployments-----"
${DIR}/wait-for-deployment.sh -n dgs-system aks-gaming-apiserver
${DIR}/wait-for-deployment.sh -n dgs-system aks-gaming-controller

echo "-----Deploying simplenodejsudp collection-----"
kubectl create -f ${DIR}/../artifacts/examples/simplenodejsudp/dedicatedgameservercollection.yaml

echo "-----Running Go DGSTester-----"
RUN_IN_K8S=false KUBECONFIG=${HOME}/.kube/${KUBECONFIG_FILE} go run ${DIR}/cmd/*.go