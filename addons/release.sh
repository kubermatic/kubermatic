#!/usr/bin/env bash

set -ex
export TAG=v0.1.8-rc

docker build -t quay.io/kubermatic/addons:${TAG} .
docker push quay.io/kubermatic/addons:${TAG}
