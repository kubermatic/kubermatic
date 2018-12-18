#!/usr/bin/env bash

ROOT_DIR="/opt/kube-test"

set -euox pipefail

TMP_ROOT="./.install-tmp"

for VERSION in 1.10 1.11 1.12 1.13; do
    DIRECTORY="${ROOT_DIR}/${VERSION}"
    if [[ ! -d "${DIRECTORY}" ]]; then
        FULL_VERSION=$(curl -Ss https://storage.googleapis.com/kubernetes-release/release/stable-${VERSION}.txt)
        echo "kube-test for ${VERSION} not found. Downloading to ${DIRECTORY} ..."

        TMP_DIR="${TMP_ROOT}/${VERSION}"
        mkdir -p ${TMP_DIR}
        mkdir -p ${DIRECTORY}

        curl -L http://gcsweb.k8s.io/gcs/kubernetes-release/release/${FULL_VERSION}/kubernetes.tar.gz -o ${TMP_DIR}/kubernetes.tar.gz
        tar -zxvf ${TMP_DIR}/kubernetes.tar.gz -C ${TMP_DIR}
        mv ${TMP_DIR}/kubernetes/* ${DIRECTORY}/

        cd ${DIRECTORY}/
        KUBERNETES_SERVER_ARCH=amd64 KUBE_VERSION="${FULL_VERSION}" KUBERNETES_DOWNLOAD_TESTS=true KUBERNETES_SKIP_CONFIRM=true ./cluster/get-kube-binaries.sh
        cd -

        # Delete all binaries for non amd64 architectures
        # We keep the windows and darwin ones, in case someone want's to test locally
        rm -rf ${DIRECTORY}/platforms/linux/arm
        rm -rf ${DIRECTORY}/platforms/linux/arm64
        rm -rf ${DIRECTORY}/platforms/linux/ppc64le
        rm -rf ${DIRECTORY}/platforms/linux/s390x
        rm -rf ${DIRECTORY}/platforms/linux/amd64/gendocs
        rm -rf ${DIRECTORY}/platforms/linux/amd64/genman
        rm -rf ${DIRECTORY}/platforms/linux/amd64/genswaggertypedocs
        rm -rf ${DIRECTORY}/platforms/linux/amd64/genyaml
        rm -rf ${DIRECTORY}/platforms/linux/amd64/kubemark
        find ${DIRECTORY} -name "*.tar.gz" -type f | xargs rm -f
    fi
done

rm -rf ${TMP_ROOT}
