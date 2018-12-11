#!/usr/bin/env bash

TAG=v0.1.0
IMAGE=quay.io/kubermatic/startup-script

set -euox pipefail

docker build --no-cache --pull -t ${IMAGE}:${TAG} .
docker push ${IMAGE}:${TAG}
