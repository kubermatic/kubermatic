#!/usr/bin/env bash

cd $(dirname $0)/..

export REGISTRY=quay.io/kubermatic/s3-exporter
export TAG=v0.4

GOOS=linux GOARCH=amd64 make s3-exporter

mv _build/s3-exporter cmd/s3-exporter/
cd cmd/s3-exporter/

docker build -t $REGISTRY:$TAG .
docker push $REGISTRY:$TAG
