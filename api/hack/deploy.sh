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

cd "$(dirname "$0")/../../"

source ./api/hack/helper.sh

function deploy {
  local name=$1
  local namespace=$2
  local path=$3

  retry 5 helm upgrade --install --force --wait --timeout 300 ${MASTER_FLAG} ${HELM_EXTRA_ARGS} --values ${VALUES_FILE} --namespace ${namespace} ${name} ${path}
}

echo "Deploying the CRD's..."
retry 5 kubectl apply -f ./config/kubermatic/crd/

if [[ "${1}" = "master" ]]; then
    deploy "nginx-ingress-controller" "nginx-ingress-controller" ./config/nginx-ingress-controller/
    deploy "cert-manager" "cert-manager" ./config/cert-manager/
    deploy "certs" "default" ./config/certs/
    deploy "oauth" "oauth" ./config/oauth/
    deploy "iap" "iap" ./config/iap/
fi

deploy "minio" "minio" ./config/minio/
deploy "nodeport-proxy" "nodeport-proxy" ./config/nodeport-proxy/

#Monitoring
deploy "prometheus" "monitoring" ./config/monitoring/prometheus/
deploy "node-exporter" "monitoring" ./config/monitoring/node-exporter/
deploy "kube-state-metrics" "monitoring" ./config/monitoring/kube-state-metrics/
deploy "grafana" "monitoring" ./config/monitoring/grafana/
deploy "alertmanager" "monitoring" ./config/monitoring/alertmanager/

#Logging
deploy "elasticsearch" "logging" ./config/logging/elasticsearch/
deploy "fluentbit" "logging" ./config/logging/fluentbit/
deploy "kibana" "logging" ./config/logging/kibana/

#Kubermatic
deploy "kubermatic" "kubermatic" ./config/kubermatic/
