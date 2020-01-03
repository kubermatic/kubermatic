#!/usr/bin/env bash

set -euo pipefail

export KUBERMATICDOCKERTAG="${KUBERMATICDOCKERTAG:-}"
export UIDOCKERTAG="${UIDOCKERTAG:-}"
export NAMESPACE="${NAMESPACE:-}"

export KUBERMATIC_WORKERNAME=${KUBERMATIC_WORKERNAME:-$(uname -n)}
KUBERMATIC_WORKERNAME=$(tr -cd '[:alnum:]' <<< $KUBERMATIC_WORKERNAME | tr '[:upper:]' '[:lower:]')
KUBERMATIC_DEBUG=${KUBERMATIC_DEBUG:-true}

if [ -z "$NAMESPACE" ]; then
  echo "You must specify a NAMESPACE environment variable to run the operator in."
  echo "Do not use a namespace where another operator is already running."
  echo "Example: NAMESPACE=$KUBERMATIC_WORKERNAME ./hack/run-operator.sh"
  exit 2
fi

if [ -z "$KUBERMATICDOCKERTAG" ]; then
  KUBERMATICDOCKERTAG=$(git rev-parse master)
fi

if [ -z "$UIDOCKERTAG" ]; then
  echo "Finding latest dashboard master revision..."
  UIDOCKERTAG=$(git ls-remote git@github.com:kubermatic/dashboard-v2.git refs/heads/master | awk '{print $1}')
  echo
fi

echo "Versions"
echo "--------"
echo
echo "  Kubermatic: $KUBERMATICDOCKERTAG (KUBERMATICDOCKERTAG variable)"
echo "  Dashboard : $UIDOCKERTAG (UIDOCKERTAG variable)"
echo

cd $(go env GOPATH)/src/github.com/kubermatic/kubermatic/api
make kubermatic-operator
echo

set -x
./_build/kubermatic-operator \
  -kubeconfig=../../secrets/seed-clusters/dev.kubermatic.io/kubeconfig \
  -namespace="$NAMESPACE" \
  -worker-name="$KUBERMATIC_WORKERNAME" \
  -log-debug=$KUBERMATIC_DEBUG \
  -log-format=Console \
  -logtostderr \
  -v=4 # Log-level for the Kube dependencies. Increase up to 9 to get request-level logs.
