#!/usr/bin/env bash

# Copyright 2025 The Kubermatic Kubernetes Platform contributors.
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

# This helper script can be used to mirror upstream system application Helm charts to Kubermatic OCI registry.
# When introducing a new helm chart version for our system applications, make sure it is:
#   - Helm chart is mirrored in Kubermatic OCI registry, use the script hack/mirror-chart.sh <chart-name>

set -euo pipefail

# ─── Default registry and repo ────────────────────────────────────────────────
REGISTRY_HOST="${REGISTRY_HOST:-quay.io}"
REPOSITORY_PREFIX="${REPOSITORY_PREFIX:-kubermatic/helm-charts}"

# ─── Chart-specific configurations ────────────────────────────────────────────
# Format: key = chart name, value = "<URL_TEMPLATE> <DEFAULT_VERSION>"
declare -A CHART_CONFIGS=(
  ["cluster-autoscaler"]="https://github.com/kubernetes/autoscaler/releases/download/cluster-autoscaler-chart-%s/cluster-autoscaler-%s.tgz 9.46.6"
  ["cilium"]="https://helm.cilium.io/cilium-%s.tgz 1.16.9"
  # Add more charts here as needed
)

# ─── Usage ────────────────────────────────────────────────────────────────────
usage() {
  echo "Usage: $0 <chart-name> [version]"
  echo "Supported charts:"
  for key in "${!CHART_CONFIGS[@]}"; do echo "  - $key"; done
  echo
  echo "Environment overrides:"
  echo "  REGISTRY_HOST        OCI registry host (default: quay.io)"
  echo "  REPOSITORY_PREFIX    OCI repo prefix (default: kubermatic/helm-charts)"
  echo
  echo "Example:"
  echo "  $0 cilium"
  echo "  $0 cluster-autoscaler 9.46.6"
  exit 1
}

# ─── Parse and validate input ─────────────────────────────────────────────────
parse_args() {
  if [[ $# -lt 1 ]]; then
    echo "Error: Missing required argument <chart-name>"
    usage
  fi

  CHART_NAME="$1"
  CHART_VERSION="${2:-}"

  if [[ ! -v "CHART_CONFIGS[$CHART_NAME]" ]]; then
    echo "Error: Unsupported chart '$CHART_NAME'"
    usage
  fi
}

# ─── Resolve URL and chart package ────────────────────────────────────────────
resolve_chart_config() {
  local config="${CHART_CONFIGS[$CHART_NAME]}"
  IFS=' ' read -r URL_TEMPLATE DEFAULT_VERSION <<< "$config"
  CHART_VERSION="${CHART_VERSION:-$DEFAULT_VERSION}"
  CHART_SOURCE="${URL_TEMPLATE//%s/$CHART_VERSION}"
  CHART_PACKAGE="${CHART_NAME}-${CHART_VERSION}.tgz"
}

# ─── Authenticate to OCI registry ─────────────────────────────────────────────
login_registry() {
  echo "   → Authenticating to registry..."

  if [ -z "${VAULT_ADDR:-}" ]; then
    export VAULT_ADDR=https://vault.kubermatic.com/
  fi

  REGISTRY_USER="${REGISTRY_USER:-$(vault kv get -field=username dev/kubermatic-quay.io)}"
  REGISTRY_PASSWORD="${REGISTRY_PASSWORD:-$(vault kv get -field=password dev/kubermatic-quay.io)}"

  echo "${REGISTRY_PASSWORD}" | helm registry login "${REGISTRY_HOST}" --username "${REGISTRY_USER}" --password-stdin
}

# ─── Mirror chart ─────────────────────────────────────────────────────────────
mirror_chart() {
  echo "🌐 Mirroring ${CHART_NAME}@${CHART_VERSION} helm chart:"
  echo "   → Destination: oci://${REGISTRY_HOST}/${REPOSITORY_PREFIX}"

  echo "   → Downloading chart..."
  helm pull "${CHART_SOURCE}" --destination ./

  echo "   → Pushing to registry..."
  helm push "${CHART_PACKAGE}" "oci://${REGISTRY_HOST}/${REPOSITORY_PREFIX}"

  echo "   → Cleaning up..."
  rm -f "${CHART_PACKAGE}"

  echo "✅ Successfully mirrored ${CHART_NAME}:${CHART_VERSION}"
}

# ─── Main ─────────────────────────────────────────────────────────────────────
main() {
  parse_args "$@"
  resolve_chart_config
  login_registry
  mirror_chart
}

main "$@"
