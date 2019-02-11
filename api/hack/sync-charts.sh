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
export LOGGING_CHARTS='elasticsearch kibana fluentbit'
export BACKUP_CHARTS='ark ark-config'
export INSTALLER_BRANCH=$1
export CHARTS_DIR=$2
export TARGET_DIR='sync_target'
export TARGET_VALUES_FILE=${TARGET_DIR}/values.example.yaml
export TARGET_VALUES_SEED_FILE=${TARGET_DIR}/values.seed.example.yaml
COMMIT=${3:-}

if [ ! -z "${COMMIT}" ]; then
  COMMIT="local folder"
fi

# create fresh clone of the installer repository
rm -rf ${TARGET_DIR}
mkdir ${TARGET_DIR}
git clone https://github.com/kubermatic/kubermatic-installer.git ${TARGET_DIR}
cd ${TARGET_DIR}
git checkout ${INSTALLER_BRANCH}
cd ..

# re-assemble example values.yaml
rm -f ${TARGET_DIR}/values.yaml

for VALUE_FILE in ${TARGET_VALUES_FILE} ${TARGET_VALUES_SEED_FILE}; do
  rm -f ${VALUE_FILE}
  echo "# THIS FILE IS GENERATED BY https://github.com/kubermatic/kubermatic/blob/master/api/hack/sync-charts.sh" > ${VALUE_FILE}
done

cat "${CHARTS_DIR}/kubermatic/values.yaml" >> ${TARGET_VALUES_SEED_FILE}

# ensure that charts we don't know yet (from a future version)
# also get cleaned up
rm -rf ${TARGET_DIR}/charts
mkdir ${TARGET_DIR}/charts

# sync base charts
for CHART in ${CHARTS}; do
  echo "syncing ${CHART}..."
  cp -r ${CHARTS_DIR}/${CHART} ${TARGET_DIR}/charts/${CHART}

  echo "# ====== ${CHART} ======" >> ${TARGET_VALUES_FILE}
  cat "${CHARTS_DIR}/${CHART}/values.yaml" >> ${TARGET_VALUES_FILE}
  echo "" >> ${TARGET_VALUES_FILE}
done

# sync monitoring charts
echo "" >> ${TARGET_VALUES_FILE}
echo "# ========================" >> ${TARGET_VALUES_FILE}
echo "# ====== Monitoring ======" >> ${TARGET_VALUES_FILE}
echo "# ========================" >> ${TARGET_VALUES_FILE}
echo "" >> ${TARGET_VALUES_FILE}
mkdir -p "${TARGET_DIR}/charts/monitoring"
for CHART in ${MONITORING_CHARTS}; do
  echo "syncing ${CHART}..."
  cp -r ${CHARTS_DIR}/monitoring/${CHART} ${TARGET_DIR}/charts/monitoring/${CHART}

  echo "# ====== ${CHART} ======" >> ${TARGET_VALUES_FILE}
  cat "${CHARTS_DIR}/monitoring/${CHART}/values.yaml" >> ${TARGET_VALUES_FILE}
  echo "" >> ${TARGET_VALUES_FILE}
done

# sync logging charts
echo "" >> ${TARGET_VALUES_FILE}
echo "# =======================" >> ${TARGET_VALUES_FILE}
echo "# ======= Logging =======" >> ${TARGET_VALUES_FILE}
echo "# =======================" >> ${TARGET_VALUES_FILE}
echo "" >> ${TARGET_VALUES_FILE}
mkdir -p "${TARGET_DIR}/charts/logging"
for CHART in ${LOGGING_CHARTS}; do
  echo "syncing ${CHART}..."
  cp -r ${CHARTS_DIR}/logging/${CHART} ${TARGET_DIR}/charts/logging/${CHART}

  echo "# ====== ${CHART} ======" >> ${TARGET_VALUES_FILE}
  cat "${CHARTS_DIR}/logging/${CHART}/values.yaml" >> ${TARGET_VALUES_FILE}
  echo "" >> ${TARGET_VALUES_FILE}
done

# sync backup charts
echo "" >> ${TARGET_VALUES_FILE}
echo "# =======================" >> ${TARGET_VALUES_FILE}
echo "# ======= Backups =======" >> ${TARGET_VALUES_FILE}
echo "# =======================" >> ${TARGET_VALUES_FILE}
echo "" >> ${TARGET_VALUES_FILE}
mkdir -p "${TARGET_DIR}/charts/backup"
for CHART in ${BACKUP_CHARTS}; do
  echo "syncing ${CHART}..."
  cp -r ${CHARTS_DIR}/backup/${CHART} ${TARGET_DIR}/charts/backup/${CHART}

  echo "# ====== ${CHART} ======" >> ${TARGET_VALUES_FILE}
  cat "${CHARTS_DIR}/backup/${CHART}/values.yaml" >> ${TARGET_VALUES_FILE}
  echo "" >> ${TARGET_VALUES_FILE}
done

# merge duplicate top-level keys
KNOWN_KEYS=()

while IFS= read LINE; do
  MATCH=$(echo "${LINE}" | grep -oE "^[a-zA-Z0-9]+" || true)

  if [ -z "${MATCH}" ]; then
    echo "${LINE}"
  else
    if ! [[ " ${KNOWN_KEYS[@]} " =~ " ${MATCH} " ]]; then
      KNOWN_KEYS+=("${MATCH}")
      echo "${LINE}"
    fi
  fi
done < ${TARGET_VALUES_FILE} > ${TARGET_DIR}/values.example.tmp.yaml

mv ${TARGET_DIR}/values.example.{tmp.,}yaml

# commit and push
cd ${TARGET_DIR}
git add .
if ! git status|grep 'nothing to commit'; then
  git commit -m "Syncing charts from commit ${COMMIT}"
  git push origin ${INSTALLER_BRANCH}
fi

cd ..
rm -rf ${TARGET_DIR}
