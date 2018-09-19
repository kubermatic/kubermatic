#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -lt 2 ] || [ "${1}" == "--help" ]; then
  cat <<EOF
Usage: $(basename $0) <branch-name> <path-to-charts>

  <branch-name>                  Name of the target branch in the kubermatic-installer repository
  <path-to-charts>               The path to the kubermatic charts to sync

Example:
  $(basename $0) v2.5 ../config
EOF
  exit 0
fi

export CHARTS='kubermatic cert-manager certs nginx-ingress-controller nodeport-proxy oauth minio iap'
export MONITORING_CHARTS='alertmanager grafana kube-state-metrics node-exporter prometheus'
export INSTALLER_BRANCH=$1
export CHARTS_DIR=$2
export TARGET_DIR='sync_target'
export TARGET_VALUES_FILE=${TARGET_DIR}/values.example.yaml
export TARGET_VALUES_SEED_FILE=${TARGET_DIR}/values.seed.example.yaml
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

rm -f ${TARGET_DIR}/values.yaml

for VALUE_FILE in ${TARGET_VALUES_FILE} ${TARGET_VALUES_SEED_FILE}; do
  rm -f ${VALUE_FILE}
  # touch ${VALUE_FILE}
  echo "# THIS FILE IS GENERATED BY https://github.com/kubermatic/kubermatic/blob/master/api/hack/sync-charts.sh" > ${VALUE_FILE}
done


cat "${CHARTS_DIR}/kubermatic/values.yaml" >> ${TARGET_VALUES_SEED_FILE}

for CHART in ${CHARTS}; do
  echo "syncing ${CHART}..."
  # doing clean copy
  rm -rf ${TARGET_DIR}/charts/${CHART}
  cp -r ${CHARTS_DIR}/${CHART} ${TARGET_DIR}/charts/${CHART}

  echo "# ====== ${CHART} ======" >> ${TARGET_VALUES_FILE}
  cat "${CHARTS_DIR}/${CHART}/values.yaml" >> ${TARGET_VALUES_FILE}
  echo "" >> ${TARGET_VALUES_FILE}
done

echo "" >> ${TARGET_VALUES_FILE}
echo "# ========================" >> ${TARGET_VALUES_FILE}
echo "# ====== Monitoring ======" >> ${TARGET_VALUES_FILE}
echo "# ========================" >> ${TARGET_VALUES_FILE}
echo "" >> ${TARGET_VALUES_FILE}
for CHART in ${MONITORING_CHARTS}; do
  echo "syncing ${CHART}..."
  # doing clean copy
  rm -rf ${TARGET_DIR}/charts/monitoring/${CHART}
  cp -r ${CHARTS_DIR}/monitoring/${CHART} ${TARGET_DIR}/charts/monitoring/${CHART}

  echo "# ====== ${CHART} ======" >> ${TARGET_VALUES_FILE}
  cat "${CHARTS_DIR}/monitoring/${CHART}/values.yaml" >> ${TARGET_VALUES_FILE}
  echo "" >> ${TARGET_VALUES_FILE}
done

cd ${TARGET_DIR}
git add .
if ! git status|grep 'nothing to commit'; then
  git commit -m "Syncing charts from commit ${COMMIT}"
  git push origin ${INSTALLER_BRANCH}
fi

cd ..
rm -rf ${TARGET_DIR}
