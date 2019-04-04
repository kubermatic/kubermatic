#!/usr/bin/env bash
set -euo pipefail

CONTROLLER_IMAGE="quay.io/kubermatic/cluster-exposer:v1.0.0"
# TODO: remove once we generate static clients from the tests
export KUBERMATIC_DEX_DEV_E2E_USERNAME="roxy@loodse.com"
export KUBERMATIC_DEX_DEV_E2E_USERNAME_2="roxy2@loodse.com"
export KUBERMATIC_DEX_DEV_E2E_PASSWORD="password"

function cleanup {
	kubectl delete service -l "prow.k8s.io/id=$PROW_JOB_ID"

	# Kill all descendant processes
	pkill -P $$
}
trap cleanup EXIT

# Step 0: Setup
# An elegant hack that routes dex.oauth domain to localhost and then down to a dex service inside the inner Kube cluster
# See also expose.sh script
echo $IMAGE_PULL_SECRET_DATA | base64 -d > /config.json
sed 's/localhost/localhost dex.oauth/' < /etc/hosts > /hosts
cat /hosts > /etc/hosts
dockerd > /dev/null 2> /dev/null &

# Step 1: Build kubermatic docker image that will be used by the inner Kube cluster
cd $(dirname $0)/../..
make build
make docker-build

# Step 2: create a Kube cluster and deploy Kubermatic
# Note that latestbuild tag comes from running "make docker-build"
deploy.sh latestbuild
DOCKER_CONFIG=/ docker run --name controller -d -v /root/.kube/config:/inner -v /etc/kubeconfig/kubeconfig:/outer --network host --privileged ${CONTROLLER_IMAGE} --kubeconfig-inner "/inner" --kubeconfig-outer "/outer" --namespace "default" --build-id "$PROW_JOB_ID"
docker logs -f controller &
expose.sh

# Step3: run e2e tests
# TODO: run api e2e test
