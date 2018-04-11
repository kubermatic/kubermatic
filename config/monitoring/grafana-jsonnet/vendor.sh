#!/usr/bin/env bash

mkdir -p vendor

curl -L --output ksonnet-lib.zip https://github.com/ksonnet/ksonnet-lib/archive/master.zip
unzip -d vendor/ ksonnet-lib.zip
mv vendor/ksonnet-lib-master/ vendor/ksonnet-lib/
rm ksonnet-lib.zip

curl -L --output grafonnet-lib.zip https://github.com/grafana/grafonnet-lib/archive/master.zip
unzip -d vendor/ grafonnet-lib.zip
mv vendor/grafonnet-lib-master/ vendor/grafonnet-lib/
rm grafonnet-lib.zip

curl -L --output kubernetes-grafana.zip https://github.com/brancz/kubernetes-grafana/archive/master.zip
unzip -d vendor/ kubernetes-grafana.zip
mv vendor/kubernetes-grafana-master/ vendor/kubernetes-grafana/
rm kubernetes-grafana.zip
