#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -lt 1 ] || [ "${1}" == "--help" ]; then
  cat <<EOF
Usage: $(basename $0) (master|seed) path/to/${VALUES_FILE} "[EXTRA HELM ARGUMENTS]"
EOF
  exit 0
fi

if [[ ! -f ${2} ]]; then
    echo "File not found!"
    exit 1
fi

HELM_EXTRA_ARGS=${@:3}
VALUES_FILE=$(realpath ${2})
if [[ "${1}" = "master" ]]; then
  MASTER_FLAG="--set=kubermatic.isMaster=true"
else
  MASTER_FLAG="--set=kubermatic.isMaster=false"
fi
DEPLOY_NODEPORT_PROXY=${DEPLOY_NODEPORT_PROXY:-true}
DEPLOY_ALERTMANAGER=${DEPLOY_ALERTMANAGER:-true}
TILLER_NAMESPACE=${TILLER_NAMESPACE:-kubermatic}

cd "$(dirname "$0")/../../"

source ./api/hack/lib.sh

function deploy {
  local name=$1
  local namespace=$2
  local path=$3

  echo "Upgrading ${name}..."
  retry 5 helm --tiller-namespace ${TILLER_NAMESPACE} upgrade --install --atomic ${MASTER_FLAG} ${HELM_EXTRA_ARGS} --values ${VALUES_FILE} --namespace ${namespace} ${name} ${path}
}

sed -i "s/__KUBERMATIC_TAG__/$GIT_HEAD_HASH/g" ./config/kubermatic/Chart.yaml

echodate "Initializing Tiller in namespace ${TILLER_NAMESPACE}"
# In clusters which have not been initialized yet, this will fail
helm version --tiller-namespace ${TILLER_NAMESPACE} || true
kubectl create serviceaccount -n ${TILLER_NAMESPACE} tiller-sa || true
kubectl create clusterrolebinding tiller-cluster-role --clusterrole=cluster-admin --serviceaccount=${TILLER_NAMESPACE}:tiller-sa  || true
retry 5 helm --tiller-namespace ${TILLER_NAMESPACE} init --service-account tiller-sa --replicas 3 --history-max 10 --force-upgrade --wait
echodate "Tiller initialized successfully"

echo "Deploying the CRD's..."
retry 5 kubectl apply -f ./config/kubermatic/crd/

if [[ "${1}" = "master" ]]; then
    deploy "nginx-ingress-controller" "nginx-ingress-controller" ./config/nginx-ingress-controller/
    deploy "cert-manager" "cert-manager" ./config/cert-manager/
    deploy "certs" "default" ./config/certs/
    deploy "oauth" "oauth" ./config/oauth/
    # We might have not configured IAP which results in nothing being deployed. This triggers https://github.com/helm/helm/issues/4295 and marks this as failed
    deploy "iap" "iap" ./config/iap/ || true
fi

deploy "minio" "minio" ./config/minio/
deploy "s3-exporter" "s3-exporter" ./config/s3-exporter/
# The NodePort proxy is only relevant in cloud environments (Where LB services can be used)
if [[ "${DEPLOY_NODEPORT_PROXY}" = true ]]; then
  deploy "nodeport-proxy" "nodeport-proxy" ./config/nodeport-proxy/
fi

#Monitoring
deploy "prometheus" "monitoring" ./config/monitoring/prometheus/
deploy "node-exporter" "monitoring" ./config/monitoring/node-exporter/
deploy "kube-state-metrics" "monitoring" ./config/monitoring/kube-state-metrics/
deploy "grafana" "monitoring" ./config/monitoring/grafana/
if [[ "${DEPLOY_ALERTMANAGER}" = true ]]; then
  deploy "alertmanager" "monitoring" ./config/monitoring/alertmanager/
fi

#Logging
deploy "elasticsearch" "logging" ./config/logging/elasticsearch/
deploy "fluentbit" "logging" ./config/logging/fluentbit/
deploy "kibana" "logging" ./config/logging/kibana/

#Kubermatic
deploy "kubermatic" "kubermatic" ./config/kubermatic/
