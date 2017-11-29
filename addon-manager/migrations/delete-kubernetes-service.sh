#!/bin/bash

KUBECTL=${KUBECTL_BIN:-/usr/local/bin/kubectl}
KUBECTL_OPTS=${KUBECTL_OPTS:-}

${KUBECTL} ${KUBECTL_OPTS} -n default delete service kubernetes
