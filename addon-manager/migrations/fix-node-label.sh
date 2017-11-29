#!/bin/bash

KUBECTL=${KUBECTL_BIN:-/usr/local/bin/kubectl}
KUBECTL_OPTS=${KUBECTL_OPTS:-}

NODES=$(${KUBECTL} ${KUBECTL_OPTS} get node -o=custom-columns=:.metadata.name)
for NODE in ${NODES}; do
    NODECLASS=$(${KUBECTL} ${KUBECTL_OPTS} get node ${NODE} -o json | jq -r '.metadata.annotations."node.k8s.io/node-class"')
    if [ -n "${NODECLASS}" ]; then
        ${KUBECTL} ${KUBECTL_OPTS} annotate node ${NODE} 'nodeset.k8s.io/node-class'=${NODECLASS}
    fi
done
