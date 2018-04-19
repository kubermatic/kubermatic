#!/usr/bin/env sh

echo -e "\033[1m>> Generating Kubernetes manifests for Grafana\033[0m"

docker build -t kubermatic/grafana-jsonnet .

docker run --rm -u=$UID:$(id -g $USER) -it -v \
    `pwd`:/go/src/github.com/kubermatic/kubermatic/config/monitoring/grafana-jsonnet \
    kubermatic/grafana-jsonnet /bin/sh compile.sh

# Escape {{ }} in grafana templating with {{ "{{ }}" }} for helm first
sed -i -e 's#{{#{{ "{{#g' grafana.yaml
sed -i -e 's#}}#}}" }}#g' grafana.yaml

cp grafana.yaml ../grafana/templates/configmaps.yaml

rm grafana.yaml
