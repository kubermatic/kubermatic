#!/usr/bin/env bash

ver=v0.6

set -euox pipefail

docker build --no-cache --pull -t quay.io/kubermatic/openvpn:$ver .
docker push quay.io/kubermatic/openvpn:$ver
