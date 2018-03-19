#!/bin/bash

KUBECTL=${KUBECTL_BIN:-/usr/local/bin/kubectl}
KUBECTL_OPTS=${KUBECTL_OPTS:-}

${KUBECTL} ${KUBECTL_OPTS} -n kube-system delete deployment apiserver-server
