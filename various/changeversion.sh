#!/bin/bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null && pwd )"

set -e

if [ "$1" == "" ] || [ "$2" == "" ]; then
   echo "Usage: ./changeversion.sh OLD_VERSION NEW_VERSION"
   exit 1
fi

# sed and OSX https://stackoverflow.com/questions/19456518/invalid-command-code-despite-escaping-periods-using-sed
# sed can use whatever delimiter you like: https://stackoverflow.com/questions/16790793/how-to-replace-strings-containing-slashes-with-sed
sed -i "s#docker.io/dgkanatsios/aks_gaming_controller:$1#docker.io/dgkanatsios/aks_gaming_controller:$2#g" $DIR/../artifacts/deploy.apiserver-controller.yaml
sed -i "s#docker.io/dgkanatsios/aks_gaming_apiserver:$1#docker.io/dgkanatsios/aks_gaming_apiserver:$2#g" $DIR/../artifacts/deploy.apiserver-controller.yaml
sed -i "s#docker.io/dgkanatsios/aks_gaming_controller:$1#docker.io/dgkanatsios/aks_gaming_controller:$2#g" $DIR/../artifacts/deploy.apiserver-controller.no-rbac.yaml
sed -i "s#docker.io/dgkanatsios/aks_gaming_apiserver:$1#docker.io/dgkanatsios/aks_gaming_apiserver:$2#g" $DIR/../artifacts/deploy.apiserver-controller.no-rbac.yaml
sed -i "s/VERSION=$1/VERSION=$2/g" $DIR/../Makefile

echo "Changed from $1 to $2"