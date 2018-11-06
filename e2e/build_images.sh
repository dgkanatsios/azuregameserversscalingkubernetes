#!/bin/bash

# https://kvz.io/blog/2013/11/21/bash-best-practices/
set -o errexit # script exits when a command fails == set -e
set -o nounset # script exits when tries to use undeclared variables == set -u
#set -o xtrace # trace what's executed == set -x
set -o pipefail

# based on https://github.com/jetstack/cert-manager/blob/master/hack/ci/lib/build_images.sh

build_images(){
    local TMP_DIR=$(mktemp -d)
    local BUNDLE_FILE="${TMP_DIR}"/aksgamingbundle.tar.gz

    docker save \
        ${APISERVER_NAME}:"${TAG}" \
        ${CONTROLLER_NAME}:"${TAG}" \
        -o "${BUNDLE_FILE}"

    # Copy docker archive into the kind container
    docker cp "${BUNDLE_FILE}" "${KIND_CONTAINER_NAME}":/aksgamingbundle.tar.gz

    # Import file into kind docker daemon
    docker exec "${KIND_CONTAINER_NAME}" docker load -i /aksgamingbundle.tar.gz

    # Cleanup
    rm -Rf "${TMP_DIR}"
}

build_images