#!/usr/bin/env bash
pushd /kubermatic > /dev/null

helm init --service-account=kubermatic-installer --upgrade
sleep 10
helm upgrade -i k8sniff -f kubermatic-values.yaml -f config/values.yaml k8sniff/
helm upgrade -i kubermatic -f kubermatic-values.yaml -f config/values.yaml kubermatic/
helm upgrade -i storage -f kubermatic-values.yaml -f config/values.yaml storage/
helm upgrade -i nginx-ingress-controller -f kubermatic-values.yaml -f config/values.yaml nginx-ingress-controller/
helm upgrade -i tpr -f kubermatic-values.yaml -f config/values.yaml thirdpartyresources/
helm upgrade -i kube-lego -f kubermatic-values.yaml -f config/values.yaml kube-lego/

# Logging
if grep -q '\bLogging\b' config/values.yaml; then
  helm upgrade -i efk-logging -f kubermatic-values.yaml -f config/values.yaml efk-logging/
fi
#Monitoring
if grep -q '\bPrometheus\b' config/values.yaml; then
  helm upgrade -i prometheus -f kubermatic-values.yaml -f config/values.yaml prometheus/
fi
# Bare metal
if grep -q '\bIsBareMetal\b' config/values.yaml; then
  helm upgrade -i coreos-ipxe-server -f kubermatic-values.yaml -f config/values.yaml coreos-ipxe-server/
  helm upgrade -i bare-metal-provider -f kubermatic-values.yaml -f config/values.yaml bare-metal-provider/
fi
