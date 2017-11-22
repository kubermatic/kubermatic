#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -lt 2 ] || [ "${1}" == "--help" ]; then
  cat <<EOF
Usage: $(basename $0) <path-to-values-file> <path-to-charts>

  <path-to-values-file>                 the path to the values.yaml which should be used
  <path-to-charts>                      the path to the directory where the kubermatic charts are located

  <path-to-override-values>  optional   the path to the values.yaml for overrides

Examples:
  $(basename $0) /kubermatic/values/values.yaml /kubermatic
  $(basename $0) /kubermatic/values/values.yaml /kubermatic /kubermatic/values/override-values.yaml
EOF
  exit 0
fi

HELM_OPTS="--tiller-namespace=kubermatic-installer"
VALUESFILE="$1"
CHARTS_PATH="$2"
OVERRIDE_VALUESFILE=${3:-}

if [ ! -f ${VALUESFILE} ]; then
    echo "${VALUESFILE} does not exist."
    exit 1
fi

if [ ! -d ${CHARTS_PATH} ]; then
    echo "${CHARTS_PATH} does not exist."
    exit 1
fi

if [ ! -z "${OVERRIDE_VALUESFILE}" ]; then
    if [ ! -f ${OVERRIDE_VALUESFILE} ]; then
        echo "${OVERRIDE_VALUESFILE} does not exist."
        exit 1
    fi

    VALUESFILE="${VALUESFILE},${OVERRIDE_VALUESFILE}"
fi

kubectl apply -f ${CHARTS_PATH}/installer/namespace.yaml
kubectl apply -f ${CHARTS_PATH}/installer/tiller-serviceaccount.yaml

# You cannot update clusterrolebindings so we recreate them
kubectl delete -f ${CHARTS_PATH}/installer/tiller-clusterrolebinding.yaml
kubectl create -f ${CHARTS_PATH}/installer/tiller-clusterrolebinding.yaml

helm ${HELM_OPTS} init --service-account tiller --upgrade
until helm ${HELM_OPTS} version
do
   sleep 5
done

############# Kubermatic #############
helm ${HELM_OPTS} upgrade -i storage -f ${VALUESFILE} ${CHARTS_PATH}/storage/
helm ${HELM_OPTS} upgrade -i k8sniff -f ${VALUESFILE} ${CHARTS_PATH}/k8sniff/
helm ${HELM_OPTS} upgrade -i nginx -f ${VALUESFILE} ${CHARTS_PATH}/nginx-ingress-controller/
helm ${HELM_OPTS} upgrade -i oauth -f ${VALUESFILE} ${CHARTS_PATH}/oauth/
helm ${HELM_OPTS} upgrade -i kubermatic -f ${VALUESFILE} ${CHARTS_PATH}/kubermatic/
helm ${HELM_OPTS} upgrade -i cert-manager -f ${VALUESFILE} ${CHARTS_PATH}/cert-manager/
helm ${HELM_OPTS} upgrade -i certs -f ${VALUESFILE} ${CHARTS_PATH}/certs/

############# MONITORING #############
# All monitoring charts require the monitoring ns.
kubectl create namespace monitoring || true
helm ${HELM_OPTS} upgrade -i prometheus-operator -f ${VALUESFILE} ${CHARTS_PATH}/monitoring/prometheus-operator/
helm ${HELM_OPTS} upgrade -i node-exporter -f ${VALUESFILE} ${CHARTS_PATH}/monitoring/node-exporter/
helm ${HELM_OPTS} upgrade -i kube-state-metrics -f ${VALUESFILE} ${CHARTS_PATH}/monitoring/kube-state-metrics/
helm ${HELM_OPTS} upgrade -i grafana -f ${VALUESFILE} ${CHARTS_PATH}/monitoring/grafana/
helm ${HELM_OPTS} upgrade -i alertmanager -f ${VALUESFILE} ${CHARTS_PATH}/monitoring/alertmanager/
helm ${HELM_OPTS} upgrade -i prometheus -f ${VALUESFILE} ${CHARTS_PATH}/monitoring/prometheus/

#TODO: Update
#helm ${HELM_OPTS} upgrade -i efk-logging -f ${VALUESFILE} ${PATH}/efk-logging/

#TODO Update when needed. Needs new implementation anyway
## Bare metal
#if grep -q '\bIsBareMetal\b' ${VALUESFILE}; then
#  helm ${HELM_OPTS} upgrade -i coreos-ipxe-server -f ${VALUESFILE} ${PATH}/coreos-ipxe-server/
#  helm ${HELM_OPTS} upgrade -i bare-metal-provider -f ${VALUESFILE} ${PATH}/bare-metal-provider/
#fi
