#!/usr/bin/env bash

ver=2.4.8-r1

set -euox pipefail

docker build --build-arg OPENVPN_VERSION=${ver} --no-cache --pull -t quay.io/kubermatic/openvpn:v${ver} .
docker push quay.io/kubermatic/openvpn:v${ver}
