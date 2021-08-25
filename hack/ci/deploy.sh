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
DEPLOY_NODEPORT_PROXY=${DEPLOY_NODEPORT_PROXY:-true}
DEPLOY_ALERTMANAGER=${DEPLOY_ALERTMANAGER:-true}
DEPLOY_MINIO=${DEPLOY_MINIO:-true}
DEPLOY_STACK=${DEPLOY_STACK:-kubermatic}
TILLER_NAMESPACE=${TILLER_NAMESPACE:-kubermatic}
HELM_INIT_ARGS=${HELM_INIT_ARGS:-""}

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
  local name=$1
  local namespace=$2
  local path=$3
  local timeout=${4:-300}

  TEST_NAME="[Helm] Deploy chart ${name}"

  if deployment_disabled "kube-system" "${name}"; then
    echodate "Deployment has been manually disabled on this cluster. Skipping this chart."
    unset TEST_NAME
    return 0
  fi

  inital_revision="$(retry 5 helm list --tiller-namespace=${TILLER_NAMESPACE} ${name} --output=json | jq '.Releases[0].Revision')"

  echodate "Upgrading ${name}..."
  retry 5 helm --tiller-namespace ${TILLER_NAMESPACE} upgrade --install --force --atomic --timeout $timeout --values ${VALUES_FILE} --namespace ${namespace} ${name} ${path}

  unset TEST_NAME
}

function deploy3 {
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

function initTiller() {
  TEST_NAME="[Helm] Init Tiller"
  echodate "Initializing Tiller in namespace ${TILLER_NAMESPACE}"
  helm version --client
  kubectl create serviceaccount -n ${TILLER_NAMESPACE} tiller-sa --dry-run -oyaml | kubectl apply -f -
  kubectl create clusterrolebinding tiller-cluster-role --clusterrole=cluster-admin --serviceaccount=${TILLER_NAMESPACE}:tiller-sa --dry-run -oyaml | kubectl apply -f -
  retry 5 helm --tiller-namespace ${TILLER_NAMESPACE} init --service-account tiller-sa --replicas 3 --history-max 10 --upgrade --force-upgrade --wait ${HELM_INIT_ARGS}
  echodate "Tiller initialized successfully"
  unset TEST_NAME
}

echodate "Deploying ${DEPLOY_STACK} stack..."
case "${DEPLOY_STACK}" in
monitoring)
  deploy3 "node-exporter" "monitoring" charts/monitoring/node-exporter/
  deploy3 "kube-state-metrics" "monitoring" charts/monitoring/kube-state-metrics/
  deploy3 "grafana" "monitoring" charts/monitoring/grafana/
  deploy3 "helm-exporter" "monitoring" charts/monitoring/helm-exporter/
  deploy3 "alertmanager" "monitoring" charts/monitoring/alertmanager/

  if [[ "${1}" = "master" ]]; then
    deploy3 "karma" "monitoring" charts/monitoring/karma/
  fi

  # Prometheus can take a long time to become ready, depending on the WAL size.
  # We try to accommodate by waiting for 15 instead of 5 minutes.
  deploy3 "prometheus" "monitoring" charts/monitoring/prometheus/ 15m
  ;;

logging)
  deploy3 "loki" "logging" charts/logging/loki/
  deploy3 "promtail" "logging" charts/logging/promtail/
  ;;

kubermatic)
  initTiller

  echodate "Deploying the Kubermatic CRDs..."
  retry 5 kubectl apply -f charts/kubermatic-operator/crd/

  # PULL_BASE_REF is the name of the current branch in case of a post-submit
  # or the name of the base branch in case of a PR.
  LATEST_DASHBOARD="$(get_latest_dashboard_hash "${PULL_BASE_REF}")"

  sed -i "s/__DASHBOARD_TAG__/$LATEST_DASHBOARD/g" charts/kubermatic/*.yaml
  sed -i "s/__KUBERMATIC_TAG__/${GIT_HEAD_HASH}/g" charts/kubermatic/*.yaml
  sed -i "s/__KUBERMATIC_TAG__/${GIT_HEAD_HASH}/g" charts/kubermatic-operator/*.yaml
  sed -i "s/__KUBERMATIC_TAG__/${GIT_HEAD_HASH}/g" charts/nodeport-proxy/*.yaml

  if [[ "${1}" = "master" ]]; then
    echodate "Deploying the cert-manager CRDs..."
    retry 5 kubectl apply -f charts/cert-manager/crd/

    deploy "nginx-ingress-controller" "nginx-ingress-controller" charts/nginx-ingress-controller/
    deploy "oauth" "oauth" charts/oauth/
    deploy "cert-manager" "cert-manager" charts/cert-manager/

    # We might have not configured IAP which results in nothing being deployed. This triggers https://github.com/helm/helm/issues/4295 and marks this as failed
    # We hack around this by grepping for a string that is mandatory in the values file of IAP
    # to determine if its configured, because am empty chart leads to Helm doing weird things
    if grep -q discovery_url ${VALUES_FILE}; then
      deploy "iap" "iap" charts/iap/
    else
      echodate "Skipping IAP deployment because discovery_url is unset in values file"
    fi
  fi

  # CI has its own Minio deployment as a proxy for GCS, so we do not install the default Helm chart here.
  if [[ "${DEPLOY_MINIO}" = true ]]; then
    deploy "minio" "minio" charts/minio/
    deploy "s3-exporter" "kube-system" charts/s3-exporter/
  fi

  # The NodePort proxy is only relevant in cloud environments (Where LB services can be used)
  if [[ "${DEPLOY_NODEPORT_PROXY}" = true ]]; then
    deploy "nodeport-proxy" "nodeport-proxy" charts/nodeport-proxy/
  fi

  # Kubermatic
  if [[ "${1}" = "master" ]]; then
    echodate "Deploying Kubermatic Operator..."

    retry 3 helm upgrade --install --force --wait --timeout 300 \
      --set-file "kubermaticOperator.imagePullSecret=$DOCKER_CONFIG" \
      --set "kubermaticOperator.image.repository=quay.io/kubermatic/kubermatic-ee" \
      --namespace kubermatic \
      --values ${VALUES_FILE} \
      kubermatic-operator \
      charts/kubermatic-operator/

    # only deploy KubermaticConfigurations on masters, on seed clusters
    # the relevant Seed CR is copied by Kubermatic itself
    if [ -n "${KUBERMATIC_CONFIG:-}" ]; then
      echodate "Deploying KubermaticConfiguration..."
      retry 3 kubectl apply -f $KUBERMATIC_CONFIG
    fi
  else
    echodate "Not deploying Kubermatic, as this is not a master cluster."
  fi
  ;;
esac
