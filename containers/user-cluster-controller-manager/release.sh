#!/usr/bin/env bash

name=user-cluster-controller-manager

ver=v0.1.0-dev
image=quay.io/kubermatic/$name

if ! grep -q "user-cluster-controller-manager:$ver" ../../api/pkg/resources/userclustercontrollermanager/deployment.go; then
	echo "version mismatch of release with deployment:"
	grep -Hn "user-cluster-controller-manager:" ../../api/pkg/resources/userclustercontrollermanager/deployment.go
	echo "release version is: $ver"
	exit 1
fi

set -euox pipefail

cd `dirname $0`

make -C ../../api $name

cp -v ../../api/_build/$name .
docker build --no-cache --pull -t $image:$ver .
docker push $image:$ver
rm -v ./$name
