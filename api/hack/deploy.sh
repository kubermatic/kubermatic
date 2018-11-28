#!/usr/bin/env bash
set -xeuo pipefail

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

HELM_EXTRA_ARGS=${3:-""}
VALUES_FILE=$(realpath ${2})
cd "$(dirname "$0")/../../"

helm upgrade --install --wait --timeout 300 ${HELM_EXTRA_ARGS} --values ${VALUES_FILE} --namespace nginx-ingress-controller nginx-ingress-controller ./config/nginx-ingress-controller/
helm upgrade --install --wait --timeout 300 ${HELM_EXTRA_ARGS} --values ${VALUES_FILE} --namespace cert-manager cert-manager ./config/cert-manager/
helm upgrade --install --wait --timeout 300 ${HELM_EXTRA_ARGS} --values ${VALUES_FILE} --namespace default certs ./config/certs/
helm upgrade --install --wait --timeout 300 ${HELM_EXTRA_ARGS} --values ${VALUES_FILE} --namespace minio minio ./config/minio/
helm upgrade --install --wait --timeout 300 ${HELM_EXTRA_ARGS} --values ${VALUES_FILE} --namespace nodeport-proxy nodeport-proxy ./config/nodeport-proxy/
helm upgrade --install --wait --timeout 300 ${HELM_EXTRA_ARGS} --values ${VALUES_FILE} --namespace iap iap ./config/iap/

helm upgrade --install --wait --timeout 300 ${HELM_EXTRA_ARGS} --values ${VALUES_FILE} --namespace monitoring prometheus ./config/monitoring/prometheus/
helm upgrade --install --wait --timeout 300 ${HELM_EXTRA_ARGS} --values ${VALUES_FILE} --namespace monitoring node-exporter ./config/monitoring/node-exporter/
helm upgrade --install --wait --timeout 300 ${HELM_EXTRA_ARGS} --values ${VALUES_FILE} --namespace monitoring kube-state-metrics ./config/monitoring/kube-state-metrics/
helm upgrade --install --wait --timeout 300 ${HELM_EXTRA_ARGS} --values ${VALUES_FILE} --namespace monitoring grafana ./config/monitoring/grafana/
helm upgrade --install --wait --timeout 300 ${HELM_EXTRA_ARGS} --values ${VALUES_FILE} --namespace monitoring alertmanager ./config/monitoring/alertmanager/

helm upgrade --install --wait --timeout 300 ${HELM_EXTRA_ARGS} --values ${VALUES_FILE} --namespace logging elasticsearch ./config/logging/elasticsearch/
helm upgrade --install --wait --timeout 300 ${HELM_EXTRA_ARGS} --values ${VALUES_FILE} --namespace logging fluentd ./config/logging/fluentd/
helm upgrade --install --wait --timeout 300 ${HELM_EXTRA_ARGS} --values ${VALUES_FILE} --namespace logging kibana ./config/logging/kibana/

helm upgrade --install --wait --timeout 300 ${HELM_EXTRA_ARGS} --values ${VALUES_FILE} --namespace kubermatic kubermatic ./config/kubermatic/

if [[ "${1}" = "master" ]]; then
    helm upgrade --install --wait --timeout 300 ${HELM_EXTRA_ARGS} --values ${VALUES_FILE} --namespace oauth oauth ./config/oauth/
fi
