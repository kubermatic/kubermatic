#!/usr/bin/env bash

ver=v0.5

set -euox pipefail

docker build --no-cache --pull -t quay.io/kubermatic/openvpn:$ver .
#docker push quay.io/kubermatic/openvpn:$ver
