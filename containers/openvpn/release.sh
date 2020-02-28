#!/usr/bin/env bash

ver=v2.4.8

set -euox pipefail

docker build --no-cache --pull -t quay.io/kubermatic/openvpn:$ver .
docker push quay.io/kubermatic/openvpn:$ver
