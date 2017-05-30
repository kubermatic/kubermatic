#!/usr/bin/env bash
helm upgrade -i k8sniff -f seed-clusters/kubermatic.arvato.com/values.yaml k8sniff/
helm upgrade -i kubermatic -f seed-clusters/kubermatic.arvato.com/values.yaml kubermatic/
helm upgrade -i storage -f seed-clusters/kubermatic.arvato.com/values.yaml storage/
helm upgrade -i nginx-ingress-controller -f seed-clusters/kubermatic.arvato.com/values.yaml nginx-ingress-controller/
helm upgrade -i tpr -f seed-clusters/kubermatic.arvato.com/values.yaml thirdpartyresources/
helm upgrade -i efk-logging -f seed-clusters/kubermatic.arvato.com/values.yaml efk-logging/
helm upgrade -i kube-lego -f seed-clusters/kubermatic.arvato.com/values.yaml kube-lego/
helm upgrade -i prometheus -f seed-clusters/kubermatic.arvato.com/values.yaml prometheus/
