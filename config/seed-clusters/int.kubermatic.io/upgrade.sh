#!/usr/bin/env bash
helm upgrade -i kubermatic -f seed-clusters/int.kubermatic.io/values.yaml kubermatic/
helm upgrade -i storage -f seed-clusters/int.kubermatic.io/values.yaml storage/
helm upgrade -i nginx-ingress-controller -f seed-clusters/int.kubermatic.io/values.yaml nginx-ingress-controller/
helm upgrade -i bare-metal-provider -f seed-clusters/int.kubermatic.io/values.yaml bare-metal-provider/
helm upgrade -i tpr -f seed-clusters/int.kubermatic.io/values.yaml thirdpartyresources/
helm upgrade -i coreos-ipxe-server -f seed-clusters/int.kubermatic.io/values.yaml coreos-ipxe-server/
helm upgrade -i kube-lego -f seed-clusters/int.kubermatic.io/values.yaml kube-lego/
helm upgrade -i prometheus -f seed-clusters/int.kubermatic.io/values.yaml prometheus/
# Disabled until we have faster storage
# helm upgrade -i efk-logging -f seed-clusters/int.kubermatic.io/values.yaml efk-logging/
