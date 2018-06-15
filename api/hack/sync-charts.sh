#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -lt 2 ] || [ "${1}" == "--help" ]; then
  cat <<EOF
Usage: $(basename $0) <path-to-values-file> <path-to-charts>

  <branch-name>                  Name of the target branch in the kubermatic-installer repository
  <path-to-charts>               The path to the kubermatic charts to sync

Example:
  $(basename $0) v2.5 ../config
EOF
  exit 0
fi

export CHARTS='cert-manager certs kubermatic nginx-ingress-controller nodeport-proxy oauth minio'
export INSTALLER_BRANCH=$1
export CHARTS_DIR=$2
export TARGET_DIR='sync_target'
COMMIT=${3:-}

if [ ! -z "${COMMIT}" ]; then
    COMMIT="local folder"
fi

rm -rf ${TARGET_DIR}
mkdir ${TARGET_DIR}
git clone https://github.com/kubermatic/kubermatic-installer.git ${TARGET_DIR}
cd ${TARGET_DIR}
git checkout ${INSTALLER_BRANCH}
cd ..

for CHART in ${CHARTS}; do
  echo "syncing ${CHART}..."
  # doing clean copy
  rm -rf ${TARGET_DIR}/charts/${CHART}
  cp -r ${CHARTS_DIR}/${CHART} ${TARGET_DIR}/charts/${CHART}
done

cd ${TARGET_DIR}
git add .
git commit -m "Syncing charts from commit ${COMMIT}"
git push origin ${INSTALLER_BRANCH}

cd ..
rm -rf ${TARGET_DIR}
