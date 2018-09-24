#!/usr/bin/env bash

ver=v0.1

set -euox pipefail

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags="-w -s" github.com/kubermatic/kubermatic/api/cmd/http-prober

docker build --no-cache --pull -t quay.io/kubermatic/http-prober:$ver .
docker push quay.io/kubermatic/http-prober:$ver

rm http-prober
