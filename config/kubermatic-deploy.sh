#!/usr/bin/env bash

if [ "$#" -lt 1 ] || [ "${1}" == "--help" ]; then
  cat <<EOF
Usage: $(basename $0) (master|seed) path/to/VALUES_FILE  path/to/CHART_FOLDER "[EXTRA HELM ARGUMENTS]"
EOF
  exit 0
fi
set -euo pipefail

if [[ ! -f ${2} ]]; then
    echo "VALUES_FILE not found! $2"
    exit 1
fi
if [[ ! -d ${3} ]]; then
    echo "CHART_FOLDER not found! $3"
    exit 1
fi

HELM_EXTRA_ARGS=${@:4}
VALUES_FILE=$(realpath ${2})
CHART_FOLDER=$(realpath ${3})
if [[ "${1}" = "master" ]]; then
  MASTER_FLAG="--set=kubermatic.isMaster=true"
else
  MASTER_FLAG="--set=kubermatic.isMaster=false"
fi
DEPLOY_NODEPORT_PROXY=${DEPLOY_NODEPORT_PROXY:-true}
DEPLOY_ALERTMANAGER=${DEPLOY_ALERTMANAGER:-true}
DEPLOY_MINIO=${DEPLOY_MINIO:-true}
DEPLOY_STACK=${DEPLOY_STACK:-kubermatic}
TILLER_NAMESPACE=${TILLER_NAMESPACE:-kubermatic}
HELM_INIT_ARGS=${HELM_INIT_ARGS:-""}
SKIP_TILLER_INIT=${SKIP_TILLER_INIT:-true}
#replaces var KUBERMATIC_TAG in master charts of https://github.com/kubermatic/kubermatic/tree/master/config
KUBERMATIC_TAG=${KUBERMATIC_TAG:-""}

source $(dirname "$0")/lib.sh

function replace() {
    if [ -z "$KUBERMATIC_TAG" ]; then
        echo "skip replacement of tags -> no deployment of master branch"
    else
        echo "replace '__KUBERMATIC_TAG__' with '$KUBERMATIC_TAG'!"
        sed -i "s/__KUBERMATIC_TAG__/${KUBERMATIC_TAG}/g" $CHART_FOLDER/kubermatic/Chart.yaml
        sed -i "s/__KUBERMATIC_TAG__/${KUBERMATIC_TAG}/g" $CHART_FOLDER/kubermatic/values.yaml
        sed -i "s/__KUBERMATIC_TAG__/${KUBERMATIC_TAG}/g" $CHART_FOLDER/kubermatic-operator/*
    fi
}

function deploy {
  local name=$1
  local namespace=$2
  local path="$CHART_FOLDER/$3"
  local timeout=${4:-300}

  if [[ ! -d $path ]]; then
    echo "chart not found! $path"
    exit 1
  fi
  TEST_NAME="[Helm] Deploy chart ${name}"

  inital_revision="$(retry 5 helm list --tiller-namespace=${TILLER_NAMESPACE} ${name} --output=json|jq '.Releases[0].Revision')"

  echodate "Upgrading ${name}..."
  retry 5 helm --tiller-namespace ${TILLER_NAMESPACE} upgrade --install --atomic --timeout $timeout ${MASTER_FLAG} ${HELM_EXTRA_ARGS} --values ${VALUES_FILE} --namespace ${namespace} ${name} ${path}

  if [ "${CANARY_DEPLOYMENT:-}" = "true" ]; then
    TEST_NAME="[Helm] Rollback chart ${name}"
    echodate "Rolling back ${name} to revision ${inital_revision} as this was only a canary deployment"
    retry 5 helm --tiller-namespace ${TILLER_NAMESPACE} rollback --wait --timeout $timeout ${name} ${inital_revision}
  fi

  unset TEST_NAME
}

function initTiller() {
    if [[ "${SKIP_TILLER_INIT}" = true ]]; then
        echo "[Helm] skipp Tiller init"
    else
      TEST_NAME="[Helm] Init Tiller"
      echodate "Initializing Tiller in namespace ${TILLER_NAMESPACE}"
      helm version --client
      kubectl create serviceaccount -n ${TILLER_NAMESPACE} tiller-sa --dry-run -oyaml|kubectl apply -f -
      kubectl create clusterrolebinding tiller-cluster-role --clusterrole=cluster-admin --serviceaccount=${TILLER_NAMESPACE}:tiller-sa  --dry-run -oyaml|kubectl apply -f -
      retry 5 helm --tiller-namespace ${TILLER_NAMESPACE} init --service-account tiller-sa --replicas 3 --history-max 10 --upgrade --force-upgrade --wait ${HELM_INIT_ARGS}
      echodate "Tiller initialized successfully"
      unset TEST_NAME
  fi
}

replace
echodate "Deploying ${DEPLOY_STACK} stack..."
case "${DEPLOY_STACK}" in
  monitoring)
    initTiller
    deploy "node-exporter" "monitoring" ./monitoring/node-exporter/
    deploy "kube-state-metrics" "monitoring" ./monitoring/kube-state-metrics/
    deploy "grafana" "monitoring" ./monitoring/grafana/
    deploy "blackbox-exporter" "monitoring" ./monitoring/blackbox-exporter/
    if [[ "${DEPLOY_ALERTMANAGER}" = true ]]; then
      deploy "alertmanager" "monitoring" ./monitoring/alertmanager/
      if [[ "${1}" = "master" ]]; then
        deploy "karma" "monitoring" ./monitoring/karma/
      fi
    fi

    # Prometheus can take a long time to become ready, depending on the WAL size.
    # We try to accomodate by waiting for 15 instead of 5 minutes.
    deploy "prometheus" "monitoring" ./monitoring/prometheus/ 900
    ;;

  logging)
    initTiller
    deploy "elasticsearch" "logging" ./logging/elasticsearch/
    deploy "fluentbit" "logging" ./logging/fluentbit/
    deploy "kibana" "logging" ./logging/kibana/
    ;;

  kubermatic)
    initTiller

    echodate "Deploying the CRD's..."
    retry 5 kubectl apply -f $CHART_FOLDER/kubermatic/crd/

    if [[ "${1}" = "master" ]]; then
      deploy "nginx-ingress-controller" "nginx-ingress-controller" ./nginx-ingress-controller/
      deploy "cert-manager" "cert-manager" ./cert-manager/
      deploy "certs" "default" ./certs/
      deploy "oauth" "oauth" ./oauth/
      # We might have not configured IAP which results in nothing being deployed. This triggers https://github.com/helm/helm/issues/4295 and marks this as failed
      # We hack around this by grepping for a string that is mandatory in the values file of IAP
      # to determine if its configured, because am empty chart leads to Helm doing weird things
      if grep -q discovery_url ${VALUES_FILE}; then
        deploy "iap" "iap" ./iap/
      else
        echodate "Skipping IAP deployment because discovery_url is unset in values file"
      fi
    fi

    # CI has its own Minio deployment as a proxy for GCS, so we do not install the default Helm chart here.
    if [[ "${DEPLOY_MINIO}" = true ]]; then
      deploy "minio" "minio" ./minio/
      deploy "s3-exporter" "kube-system" ./s3-exporter/
    fi

    # The NodePort proxy is only relevant in cloud environments (Where LB services can be used)
    if [[ "${DEPLOY_NODEPORT_PROXY}" = true ]]; then
      deploy "nodeport-proxy" "nodeport-proxy" ./nodeport-proxy/
    fi

    # Kubermatic
    deploy "kubermatic" "kubermatic" ./kubermatic/
    ;;

  kubermatic-deployment-only)
    # Kubermatic only without other components
    deploy "kubermatic" "kubermatic" ./kubermatic/
    ;;

  kubermatic-operator)
    kubectl create namespace kubermatic-operator || true
    kubectl apply -f ./kubermatic-operator/
    ;;
esac
