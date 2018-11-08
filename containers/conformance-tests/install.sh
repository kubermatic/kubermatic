#!/usr/bin/env bash

ROOT_DIR="/opt/kube-test"

set -euox pipefail

TMP_ROOT="./.install-tmp"

for VERSION in 1.9 1.10 1.11 1.12; do
    DIRECTORY="${ROOT_DIR}/${VERSION}"
    if [ ! -d "${DIRECTORY}" ]; then

        FULL_VERSION=$(curl -Ss https://storage.googleapis.com/kubernetes-release/release/stable-${VERSION}.txt)
        echo "kube-test for ${VERSION} not found. Downloading to ${DIRECTORY} ..."

        TMP_DIR="${TMP_ROOT}/${VERSION}"
        mkdir -p ${TMP_DIR}
        mkdir -p ${DIRECTORY}

        curl -L http://gcsweb.k8s.io/gcs/kubernetes-release/release/${FULL_VERSION}/kubernetes.tar.gz -o ${TMP_DIR}/kubernetes.tar.gz
        tar -zxvf ${TMP_DIR}/kubernetes.tar.gz -C ${TMP_DIR}
        mv ${TMP_DIR}/kubernetes/* ${DIRECTORY}/

        cd ${DIRECTORY}/ && KUBE_VERSION="${FULL_VERSION}" KUBERNETES_DOWNLOAD_TESTS=true KUBERNETES_SKIP_CONFIRM=true ./cluster/get-kube-binaries.sh
        cd -
    fi
done

rm -rf ${TMP_ROOT}
