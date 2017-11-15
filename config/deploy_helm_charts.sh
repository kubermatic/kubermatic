#!/usr/bin/env bash
set -euo pipefail

KUBECONFIG=/kubermatic/kubeconfig
VALUESFILE=/kubermatic/values.yaml
HELM_OPTS="--tiller-namespace=kubermatic-installer"

if [ ! -f ${VALUESFILE} ]; then
    echo "${VALUESFILE} does not exist."
    exit 1
fi

if [ ! -f ${KUBECONFIG} ]; then
    echo "${KUBECONFIG} does not exist."
    exit 1
fi

kubectl apply -f installer-assets/installer-ns.yaml
kubectl apply -f installer-assets/tiller-serviceaccount.yaml

# You cannot update clusterrolebindings so we recreate them
kubectl delete -f installer-assets/tiller-clusterrolebinding.yaml
kubectl create -f installer-assets/tiller-clusterrolebinding.yaml

helm ${HELM_OPTS} init --service-account tiller --upgrade
until helm ${HELM_OPTS} version
do
   sleep 5
done

############# Kubermatic #############
helm ${HELM_OPTS} upgrade -i storage -f ${VALUESFILE} storage/
helm ${HELM_OPTS} upgrade -i k8sniff -f ${VALUESFILE} k8sniff/
helm ${HELM_OPTS} upgrade -i nginx -f ${VALUESFILE} nginx-ingress-controller/
helm ${HELM_OPTS} upgrade -i oauth -f ${VALUESFILE} oauth/
helm ${HELM_OPTS} upgrade -i kubermatic -f ${VALUESFILE} kubermatic/
helm ${HELM_OPTS} upgrade -i cert-manager -f ${VALUESFILE} cert-manager/
helm ${HELM_OPTS} upgrade -i certs -f ${VALUESFILE} certs/

############# PROMETHEUS #############
# All monitoring charts require the monitoring ns.
kubectl create namespace monitoring || true
helm ${HELM_OPTS} upgrade -i prometheus-operator -f ${VALUESFILE} monitoring/prometheus-operator/
helm ${HELM_OPTS} upgrade -i node-exporter -f ${VALUESFILE} monitoring/node-exporter/
helm ${HELM_OPTS} upgrade -i kube-state-metrics -f ${VALUESFILE} monitoring/kube-state-metrics/
helm ${HELM_OPTS} upgrade -i grafana -f ${VALUESFILE} monitoring/grafana/
helm ${HELM_OPTS} upgrade -i alertmanager -f ${VALUESFILE} monitoring/alertmanager/
helm ${HELM_OPTS} upgrade -i prometheus -f ${VALUESFILE} monitoring/prometheus/

#TODO: Update
#helm ${HELM_OPTS} upgrade -i efk-logging -f ${VALUESFILE} efk-logging/

#TODO Update when needed. Needs new implementation anyway
## Bare metal
#if grep -q '\bIsBareMetal\b' ${VALUESFILE}; then
#  helm ${HELM_OPTS} upgrade -i coreos-ipxe-server -f ${VALUESFILE} coreos-ipxe-server/
#  helm ${HELM_OPTS} upgrade -i bare-metal-provider -f ${VALUESFILE} bare-metal-provider/
#fi
