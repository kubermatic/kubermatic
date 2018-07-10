#!/usr/bin/env bash

export REGISTRY=quay.io/kubermatic/s3-exporter
export TAG=v0.2

make -C $(dirname $0)/../ s3-exporter

mv $(dirname $0)/../_build/s3-exporter $(dirname $0)/../cmd/s3-exporter/

cd $(dirname $0)/../cmd/s3-exporter/

docker build -t $REGISTRY:$TAG .
docker push $REGISTRY:$TAG
