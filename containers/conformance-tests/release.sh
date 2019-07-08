#!/usr/bin/env bash

TAG=v0.9.4

set -euox pipefail

docker build --pull -t kubermatic/kubernetes-test-binaries:${TAG} .
#docker push kubermatic/kubernetes-test-binaries:${TAG}
