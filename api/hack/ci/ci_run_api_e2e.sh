#!/usr/bin/env bash
set -euo pipefail

SDIR=$(dirname $0)
CONTROLLER_IMAGE="quay.io/kubermatic/cluster-exposer:v2.0.0"

function cleanup {
    cat ${SDIR}/../../pkg/test/e2e/api/utils/oidc-proxy-client/_build/oidc-proxy-client-errors

	kubectl delete service -l "prow.k8s.io/id=$PROW_JOB_ID"

	# Kill all descendant processes
	pkill -P $$
}
trap cleanup EXIT

# Step 0: Setup
echo $IMAGE_PULL_SECRET_DATA | base64 -d > /config.json
#TODO stop redirecting stdout and stderr to /dev/null because it makes troubleshooting harder
dockerd > /dev/null 2> /dev/null &

# Step 1: Build kubermatic docker images that will be used by the inner Kube cluster
(
cd ${SDIR}/../../cmd/kubeletdnat-controller
time make build
time docker build --network host -t quay.io/kubermatic/kubeletdnat-controller:latestbuild .
)

(
cd ${SDIR}/../..
export KUBERMATICCOMMIT="latestbuild"
time make build
time docker build --network host -t quay.io/kubermatic/api:latestbuild .
)

# Step 2: create a Kube cluster and deploy Kubermatic
# Note that latestbuild tag comes from running "make docker-build"
# scripts deploy.sh and expose.sh are provided by the docker image
time deploy.sh latestbuild
DOCKER_CONFIG=/ docker run --name controller -d -v /root/.kube/config:/inner -v /etc/kubeconfig/kubeconfig:/outer --network host --privileged ${CONTROLLER_IMAGE} --kubeconfig-inner "/inner" --kubeconfig-outer "/outer" --namespace "default" --build-id "$PROW_JOB_ID"
docker logs -f controller &
time expose.sh

# Step 3: An elegant hack that routes dex.oauth domain to localhost and then down to a dex service inside the inner Kube cluster
# See also expose.sh script
sed 's/localhost/localhost dex.oauth/' < /etc/hosts > /hosts
cat /hosts > /etc/hosts

# Step 3: create and run OIDC proxy client
# TODO: since OIDC_CLIENT_ID and OIDC_CLIENT_SECRET are defined in the docker image
#       they could be exposed as envs by that image
export KUBERMATIC_OIDC_CLIENT_ID="kubermatic"
export KUBERMATIC_OIDC_CLIENT_SECRET="ZXhhbXBsZS1hcHAtc2VjcmV0"
export KUBERMATIC_OIDC_ISSUER="http://dex.oauth:5556"
export KUBERMATIC_OIDC_REDIRECT_URI="http://localhost:8000"
(
cd ${SDIR}/../../pkg/test/e2e/api/utils/oidc-proxy-client
make build
make run > /dev/null 2> ./_build/oidc-proxy-client-errors &
)

# TODO check if oidc-proxy-client is available on port 5556 before running e2e tests

# Step 4: run e2e tests
echo "running the API E2E tests"
more /etc/hosts
go test -tags=create -timeout 20m ${SDIR}/../../pkg/test/e2e/api -v
go test -tags=e2e ${SDIR}/../../pkg/test/e2e/api -v
