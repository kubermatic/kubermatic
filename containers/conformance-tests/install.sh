#!/usr/bin/env bash

ROOT_DIR="/opt/kube-test"

set -euox pipefail

for VERSION in 1.9 1.10 1.11 1.12; do
    DIRECTORY="${ROOT_DIR}/${VERSION}"
    if [ ! -d "${DIRECTORY}" ]; then

        FULL_VERSION=$(curl -Ss https://storage.googleapis.com/kubernetes-release/release/stable-${VERSION}.txt)
        echo "kube-test for ${VERSION} not found. Downloading to ${DIRECTORY} ..."

        TMP_DIR="./.install-tmp/${VERSION}"
        mkdir -p ${TMP_DIR}
        mkdir -p ${DIRECTORY}

        curl -L http://gcsweb.k8s.io/gcs/kubernetes-release/release/${FULL_VERSION}/kubernetes.tar.gz -o ${TMP_DIR}/kubernetes.tar.gz
        tar -zxvf ${TMP_DIR}/kubernetes.tar.gz -C ${TMP_DIR}

        cd ${TMP_DIR} && KUBE_VERSION="${FULL_VERSION}" KUBERNETES_DOWNLOAD_TESTS=true KUBERNETES_SKIP_CONFIRM=true ./kubernetes/cluster/get-kube-binaries.sh
        cd -

        mv ${TMP_DIR}/kubernetes/cluster ${DIRECTORY}/
        mv ${TMP_DIR}/kubernetes/platforms/linux/amd64/e2e.test ${DIRECTORY}/
        mv ${TMP_DIR}/kubernetes/platforms/linux/amd64/ginkgo ${DIRECTORY}/
        mv ${TMP_DIR}/kubernetes/platforms/linux/amd64/kubectl ${DIRECTORY}/

        rm -rf ${TMP_DIR}
    fi
done
