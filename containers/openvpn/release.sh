#!/usr/bin/env bash

ver=v0.4

set -euox pipefail

docker build --no-cache --pull -t kubermatic/openvpn:$ver .
docker push kubermatic/openvpn:$ver
