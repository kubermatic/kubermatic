#!/usr/bin/env bash

echo -e "\033[1m>> Compiling jsonnet files\033[0m"

jsonnet \
    -J /go/src/github.com/ksonnet/ksonnet-lib \
    -J /go/src/github.com/grafana/grafonnet-lib \
    -J /go/src/github.com/brancz/kubernetes-grafana/src/kubernetes-jsonnet \
    grafana.jsonnet | gojsontoyaml > grafana.yaml
