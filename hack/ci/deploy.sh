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
    retry 5 helm upgrade \
      --namespace "$namespace" \
      --create-namespace \
      --install \
      --atomic \
      --timeout "$timeout" \
      --values "$VALUES_FILE" \
      "$name" "$path"
  fi

  unset TEST_NAME
}

# silence complaints by Helm
chmod 600 "$KUBECONFIG"

set_helm_charts_version "9.9.9-${GIT_HEAD_HASH}" "${GIT_HEAD_HASH}"
copy_crds_to_chart
set_crds_version_annotation

echodate "Deploying ${DEPLOY_STACK} stack..."
case "${DEPLOY_STACK}" in
usercluster-mla)
  echodate "Running Kubermatic Installer for UserCluster MLA..."
  # deploy iap only when it is set in values file
  if $(yq '.iap.deployments != null' ${VALUES_FILE}); then
    ./_build/kubermatic-installer deploy usercluster-mla \
      --config "$KUBERMATIC_CONFIG" \
      --helm-values "$VALUES_FILE" \
      --mla-include-iap \
      --helm-timeout=30m
  else
    ./_build/kubermatic-installer deploy usercluster-mla \
      --config "$KUBERMATIC_CONFIG" \
      --helm-values "$VALUES_FILE" \
      --helm-timeout=30m
  fi
  ;;

seed-mla)
  echodate "Running Kubermatic Installer for Master/Seed MLA..."
  # deploy iap only when it is set in values file
  if $(yq '.iap.deployments != null' ${VALUES_FILE}); then
    ./_build/kubermatic-installer deploy seed-mla \
      --config "$KUBERMATIC_CONFIG" \
      --helm-values "$VALUES_FILE" \
      --mla-include-iap \
      --helm-timeout=30m
  else
    ./_build/kubermatic-installer deploy seed-mla \
      --config "$KUBERMATIC_CONFIG" \
      --helm-values "$VALUES_FILE" \
      --helm-timeout=30m
  fi
  ;;

kubermatic)
  if [ -n "${IMAGE_PULL_SECRET:-}" ]; then
    export IMAGE_PULL_SECRET_CONTENT="$(cat $IMAGE_PULL_SECRET)"
    yq --inplace ".kubermaticOperator.imagePullSecret = strenv(IMAGE_PULL_SECRET_CONTENT)" charts/kubermatic-operator/values.yaml
  fi

  # Kubermatic
  clusterType="$1"

  echodate "Running Kubermatic Installer..."

  # --force must be given because the Chart versions have not necessarily changed.
  ./_build/kubermatic-installer deploy "kubermatic-$clusterType" \
    --storageclass copy-default \
    --config "$KUBERMATIC_CONFIG" \
    --helm-values "$VALUES_FILE" \
    --migrate-upstream-cert-manager \
    --force

  ;;
esac
