#!/bin/bash
cd /kubermatic
BAREMETAL="$(grep 'IsBareMetal' config/values.yaml | grep -v '#' | grep -o 'true')"
LOGGING="$(grep 'Logging' config/values.yaml | grep -v '#' | grep -o 'true')"
PROMETHEUS="$(grep 'Prometheus' config/values.yaml | grep -v '#' | grep -o 'true')"

helm init --upgrade
sleep 60
helm upgrade -i k8sniff -f kubermatic-values.yaml -f config/values.yaml k8sniff/
helm upgrade -i kubermatic -f kubermatic-values.yaml -f config/values.yaml kubermatic/
helm upgrade -i storage -f kubermatic-values.yaml -f config/values.yaml storage/
helm upgrade -i nginx-ingress-controller -f kubermatic-values.yaml -f config/values.yaml nginx-ingress-controller/
helm upgrade -i tpr -f kubermatic-values.yaml -f config/values.yaml thirdpartyresources/
helm upgrade -i kube-lego -f kubermatic-values.yaml -f config/values.yaml kube-lego/

# Logging
if $LOGGING = true ; then
  helm upgrade -i efk-logging -f kubermatic-values.yaml -f config/values.yaml efk-logging/
fi
#Monitoring
if $PROMETHEUS = true ; then
  helm upgrade -i prometheus -f kubermatic-values.yaml -f config/values.yaml prometheus/
fi
# Bare metal
if $BAREMETAL = true ; then
  helm upgrade -i coreos-ipxe-server -f kubermatic-values.yaml -f config/values.yaml coreos-ipxe-server/
  helm upgrade -i bare-metal-provider -f kubermatic-values.yaml -f config/values.yaml bare-metal-provider/
fi
