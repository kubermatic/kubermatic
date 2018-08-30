#!/usr/bin/env bash

name=user-cluster-controller-manager

ver=v0.1.0-dev1
image=quay.io/kubermatic/$name

set -euox pipefail

cd `dirname $0`

make -C ../../api $name

cp -v ../../api/_build/$name .
docker build --no-cache --pull -t $image:$ver .
docker push $image:$ver
rm -v ./$name
