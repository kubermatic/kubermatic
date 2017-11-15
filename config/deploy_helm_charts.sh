#!/usr/bin/env bash
export  KUBECONFIG=/kubermatic/kubeconfig
pushd /kubermatic > /dev/null

kubectl create namespace kubermatic-installer
kubectl create serviceaccount tiller --namespace kubermatic-installer
kubectl create -f role-tiller.yaml
kubectl create -f rolebinding-tiller.yaml
helm init --service-account tiller --tiller-namespace kubermatic-installer --upgrade
sleep 10
helm upgrade -i storage -f kubermatic-values.yaml -f config/values.yaml storage/
helm upgrade -i k8sniff -f kubermatic-values.yaml -f config/values.yaml k8sniff/
helm upgrade -i nginx-ingress-controller -f kubermatic-values.yaml -f config/values.yaml nginx-ingress-controller/
helm upgrade -i oauth -f kubermatic-values.yaml -f config/values.yaml oauth/
helm upgrade -i kubermatic -f kubermatic-values.yaml -f config/values.yaml kubermatic/
helm upgrade -i --namespace=cert-manager cert-manager -f kubermatic-values.yaml -f config/values.yaml cert-manager/
helm upgrade -i certs -f kubermatic-values.yaml -f config/values.yaml certs/

# Logging
if grep -q '\bLogging\b' config/values.yaml; then
  helm upgrade -i efk-logging -f kubermatic-values.yaml -f config/values.yaml efk-logging/
fi
#Monitoring
if grep -q '\bPrometheus\b' config/values.yaml; then
  helm upgrade -i prometheus -f kubermatic-values.yaml -f config/values.yaml monitoring/prometheus/
fi
# Bare metal
if grep -q '\bIsBareMetal\b' config/values.yaml; then
  helm upgrade -i coreos-ipxe-server -f kubermatic-values.yaml -f config/values.yaml coreos-ipxe-server/
  helm upgrade -i bare-metal-provider -f kubermatic-values.yaml -f config/values.yaml bare-metal-provider/
fi
