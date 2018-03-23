#!/usr/bin/env bash

jsonnet \
    -J vendor/ksonnet-lib \
    -J vendor/grafonnet-lib \
    -J vendor/kubernetes-grafana/src/kubernetes-jsonnet/grafana/configs \
    grafana.jsonnet | gojsontoyaml > ../grafana/templates/configmaps.yaml

# Escape {{ }} in grafana templating with {{ "{{ }}" }} for helm first
sed -i -e 's#{{#{{ "{{#g' ../grafana/templates/configmaps.yaml
sed -i -e 's#}}#}}" }}#g' ../grafana/templates/configmaps.yaml
