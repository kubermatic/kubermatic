#!/usr/bin/env bash

ver=v0.2.0-rc1
image=quay.io/kubermatic/vpnsidecar-dnat-controller

set -euox pipefail

cd `dirname $0`

make -C ../../api kubeletdnat-controller

cp -v ../../api/_build/kubeletdnat-controller .

docker build --no-cache --pull -t $image:$ver .
docker push $image:$ver
