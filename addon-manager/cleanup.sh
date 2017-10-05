#!/bin/bash

KUBECTL=${KUBECTL_BIN:-/usr/local/bin/kubectl}
KUBECTL_OPTS=${KUBECTL_OPTS:-}

#Delete old flannel since we moved to canal
${KUBECTL} ${KUBECTL_OPTS} delete ClusterRole flannel
${KUBECTL} ${KUBECTL_OPTS} delete ClusterRoleBinding flannel
${KUBECTL} ${KUBECTL_OPTS} -n kube-system delete ConfigMap kube-flannel-cfg
${KUBECTL} ${KUBECTL_OPTS} -n kube-system delete DaemonSet kube-flannel-ds
