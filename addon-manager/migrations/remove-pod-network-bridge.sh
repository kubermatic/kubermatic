#!/bin/bash

KUBECTL=${KUBECTL_BIN:-/usr/local/bin/kubectl}
KUBECTL_OPTS=${KUBECTL_OPTS:-}

#Delete old flannel since we moved to canal
${KUBECTL} ${KUBECTL_OPTS} -n kube-system delete deployment pod-network-bridge
