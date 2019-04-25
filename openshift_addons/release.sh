#!/usr/bin/env bash

set -ex
export TAG=v0.9-dev1

docker build -t quay.io/kubermatic/openshift-addons:${TAG} .
docker push quay.io/kubermatic/openshift-addons:${TAG}
