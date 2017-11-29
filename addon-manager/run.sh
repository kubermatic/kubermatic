#!/bin/bash

KUBECTL=${KUBECTL_BIN:-/usr/local/bin/kubectl}
KUBECTL_OPTS=${KUBECTL_OPTS:-}

until ${KUBECTL} ${KUBECTL_OPTS} get ns
do
    echo "apiserver not available"
    sleep 5
done
echo "apiserver available!"

/opt/migrate.sh
/opt/kube-addons.sh
