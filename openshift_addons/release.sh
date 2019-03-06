#!/usr/bin/env bash

set -ex
export TAG=v0.4

docker build -t quay.io/kubermatic/openshift-addons:${TAG} .
docker push quay.io/kubermatic/openshift-addons:${TAG}
