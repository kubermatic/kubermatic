#!/usr/bin/env bash

TAG=v0.10.0

set -euox pipefail

buildah build-using-dockerfile --squash --tag docker.io/kubermatic/kubernetes-test-binaries:${TAG} .
buildah push kubermatic/kubernetes-test-binaries:${TAG}
