#!/usr/bin/env bash

set -euo pipefail

TAG=v2.2.1
export TAG
make docker
docker push quay.io/kubermatic/nodeport-proxy:${TAG}
