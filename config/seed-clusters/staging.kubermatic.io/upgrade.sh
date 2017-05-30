#!/usr/bin/env bash
helm upgrade -i k8sniff -f seed-clusters/staging.kubermatic.io/values.yaml k8sniff/
helm upgrade -i kubermatic -f seed-clusters/staging.kubermatic.io/values.yaml kubermatic/
helm upgrade -i storage -f seed-clusters/staging.kubermatic.io/values.yaml storage/
helm upgrade -i nginx-ingress-controller -f seed-clusters/staging.kubermatic.io/values.yaml nginx-ingress-controller/
helm upgrade -i tpr -f seed-clusters/staging.kubermatic.io/values.yaml thirdpartyresources/
helm upgrade -i efk-logging -f seed-clusters/staging.kubermatic.io/values.yaml efk-logging/
helm upgrade -i kube-lego -f seed-clusters/staging.kubermatic.io/values.yaml kube-lego/
helm upgrade -i prometheus -f seed-clusters/staging.kubermatic.io/values.yaml prometheus/
