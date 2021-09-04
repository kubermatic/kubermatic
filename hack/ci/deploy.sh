#!/usr/bin/env bash

# Copyright 2020 The Kubermatic Kubernetes Platform contributors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

### This script is used by all other `deploy-*` scripts to handle the
### common deployment logic. It can setup the main, monitoring and logging
### stacks by updating the Helm charts on the target cluster.

set -euo pipefail

if [ "$#" -lt 1 ] || [ "$1" == "--help" ]; then
  cat << EOF
Usage: $(basename $0) (master|seed) path/to/${VALUES_FILE}
EOF
  exit 0
fi

if [[ ! -f "$2" ]]; then
  echo "File not found!"
  exit 1
fi

VALUES_FILE="$(realpath "$2")"
DEPLOY_MINIO=${DEPLOY_MINIO:-true}
DEPLOY_STACK=${DEPLOY_STACK:-kubermatic}

cd $(dirname "$0")/../..
source hack/lib.sh

deployment_disabled() {
  local namespace="$1"
  local release="$2"
  local name="prow-disable-$release-chart"

  # retrieve a dummy configmap
  cm=$(kubectl --namespace "$namespace" get configmap "$name" --output json --ignore-not-found)
  if [ -z "$cm" ]; then
    return 1
  fi

  # if the ConfigMap is older than a week, assume someone forgot it;
  # fail the deployment loudly so someone can fix it
  age=$(echo "$cm" | jq -r 'now - (.metadata.creationTimestamp | fromdateiso8601)')
  age="${age%.*}"

  if [ "$age" -gt "604800" ]; then
    echo "ConfigMap $name exists but is older than 7 days. Either remove the ConfigMap or re-create it to get another 7 days of paused deployments."
    exit 1
  fi

  return 0
}

function deploy {
  local name="$1"
  local namespace="$2"
  local path="$3"
  local timeout="${4:-10m}"

  TEST_NAME="[Helm] Deploy chart $name"

  if deployment_disabled "$namespace" "$name"; then
    echodate "Deployment has been manually disabled on this cluster. Skipping this chart."
  else
    echodate "Upgrading $name..."

    # Do not set --force, as this will cause problems when upgrading a Helm release.
    # See Helm issue #7956 and the upstream k8s bug #91459, which was fixed in 1.22+.
    retry 5 helm3 --namespace "$namespace" upgrade --install --atomic --timeout "$timeout" --values "$VALUES_FILE" "$name" "$path"
  fi

  unset TEST_NAME
}

# silence complaints by Helm
chmod 600 "$KUBECONFIG"

echodate "Deploying ${DEPLOY_STACK} stack..."
case "${DEPLOY_STACK}" in
monitoring)
  deploy "node-exporter" "monitoring" charts/monitoring/node-exporter/
  deploy "kube-state-metrics" "monitoring" charts/monitoring/kube-state-metrics/
  deploy "grafana" "monitoring" charts/monitoring/grafana/
  deploy "helm-exporter" "monitoring" charts/monitoring/helm-exporter/
  deploy "alertmanager" "monitoring" charts/monitoring/alertmanager/

  if [[ "${1}" = "master" ]]; then
    deploy "karma" "monitoring" charts/monitoring/karma/
  fi

  # Prometheus can take a long time to become ready, depending on the WAL size.
  # We try to accommodate by waiting for 15 instead of 5 minutes.
  deploy "prometheus" "monitoring" charts/monitoring/prometheus/ 15m
  ;;

logging)
  deploy "loki" "logging" charts/logging/loki/
  deploy "promtail" "logging" charts/logging/promtail/
  ;;

kubermatic)
  sed -i "s/__KUBERMATIC_TAG__/${GIT_HEAD_HASH}/g" charts/kubermatic-operator/*.yaml

  yq write --inplace charts/kubermatic-operator/values.yaml 'kubermaticOperator.imagePullSecret' "$(cat $DOCKER_CONFIG)"

  # Kubermatic
  if [[ "${1}" = "master" ]]; then
    echodate "Running Kubermatic Installer..."

    ./_build/kubermatic-installer deploy \
      --storageclass copy-default \
      --config "$KUBERMATIC_CONFIG" \
      --helm-values "$VALUES_FILE" \
      --helm-binary "helm3"

    # We might have not configured IAP which results in nothing being deployed. This triggers https://github.com/helm/helm/issues/4295 and marks this as failed
    # We hack around this by grepping for a string that is mandatory in the values file of IAP
    # to determine if its configured, because am empty chart leads to Helm doing weird things
    if grep -q discovery_url ${VALUES_FILE}; then
      deploy "iap" "iap" charts/iap/
    else
      echodate "Skipping IAP deployment because discovery_url is unset in values file"
    fi
  else
    echodate "Installing Kubermatic CRDs into seed cluster..."
    retry 3 kubectl apply --filename charts/kubermatic-operator/crd/
  fi

  if [[ "${DEPLOY_MINIO}" = true ]]; then
    deploy "minio" "minio" charts/minio/
    deploy "s3-exporter" "kube-system" charts/s3-exporter/
  fi
  ;;
esac
