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

# This script mirrors upstream Helm charts (used by System Applications) to the Kubermatic OCI registry.
# For usage instructions and details on adding new charts or mirroring new versions,
# refer to the accompanying README.

set -euo pipefail

# ─── Default registry and repo ────────────────────────────────────────────────
REGISTRY_HOST="${REGISTRY_HOST:-quay.io}"
REPOSITORY_PREFIX="${REPOSITORY_PREFIX:-kubermatic/helm-charts}"

# ─── Chart-specific configurations ────────────────────────────────────────────
# Format: key = chart name, value = "<URL_TEMPLATE>"
declare -A CHART_URLS=(
  ["cluster-autoscaler"]="https://github.com/kubernetes/autoscaler/releases/download/cluster-autoscaler-chart-%s/cluster-autoscaler-%s.tgz"
  ["cilium"]="https://helm.cilium.io/cilium-%s.tgz"
  # Add more charts here as needed
)

# Default versions for each chart
declare -A CHART_VERSIONS=(
  ["cluster-autoscaler"]="9.46.6"
  ["cilium"]="1.16.9"
  # Add more default versions here as needed
)

# ─── Usage ────────────────────────────────────────────────────────────────────
usage() {
  echo "Usage: $0 <chart-name> [version (optional)]"
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

  if [[ ! -v "CHART_URLS[$CHART_NAME]" ]]; then
    echo "Error: Unsupported chart '$CHART_NAME'"
    usage
  fi
}

# ─── Resolve URL and chart package ────────────────────────────────────────────
resolve_chart_config() {
  # Get the URL template for the specified chart
  URL_TEMPLATE="${CHART_URLS[$CHART_NAME]}"

  # Use the default version if no version is provided
  CHART_VERSION="${CHART_VERSION:-${CHART_VERSIONS[$CHART_NAME]}}"

  # Render the URL_Template and replace %s with the selected version
  CHART_SOURCE="${URL_TEMPLATE//%s/$CHART_VERSION}"
  CHART_PACKAGE="${CHART_NAME}-${CHART_VERSION}.tgz"
}

# ─── Authenticate to OCI registry ─────────────────────────────────────────────
login_registry() {
  echo "🌐 Authenticating to registry..."

  if [ -z "${VAULT_ADDR:-}" ]; then
    export VAULT_ADDR=https://vault.kubermatic.com/
  fi

  REGISTRY_USER="${REGISTRY_USER:-$(vault kv get -field=username dev/kubermatic-quay.io)}"
  REGISTRY_PASSWORD="${REGISTRY_PASSWORD:-$(vault kv get -field=password dev/kubermatic-quay.io)}"

  echo "${REGISTRY_PASSWORD}" | helm registry login "${REGISTRY_HOST}" --username "${REGISTRY_USER}" --password-stdin
}

# ─── Logout from the OCI registry ─────────────────────────────────────────────
logout_registry() {
  echo "🌐 Logging out from registry..."

  helm registry logout ${REGISTRY_HOST}
}

# ─── Check if chart exists in registry ────────────────────────────────────────
chart_exists_in_registry() {
  local oci_repo="oci://${REGISTRY_HOST}/${REPOSITORY_PREFIX}/${CHART_NAME}"

  # Use `helm show chart` to check if the specific version exists
  if helm show chart "$oci_repo" --version "$CHART_VERSION" > /dev/null 2>&1; then
    return 0 # Chart exists
  else
    return 1 # Chart does not exist
  fi
}

# ─── Mirror chart ─────────────────────────────────────────────────────────────
mirror_chart() {
  echo "🌐 Mirroring ${CHART_NAME}@${CHART_VERSION} helm chart:"
  echo "   → Destination: oci://${REGISTRY_HOST}/${REPOSITORY_PREFIX}"

  # Check if the chart already exists in the registry
  if chart_exists_in_registry; then
    echo "   → Chart already exists in the registry. Skipping mirroring."
    return
  fi

  echo "   → Downloading chart..."
  helm pull "${CHART_SOURCE}" --destination ./

  echo "   → Pushing to registry..."
  helm push "./${CHART_PACKAGE}" "oci://${REGISTRY_HOST}/${REPOSITORY_PREFIX}"

  echo "   → Cleaning up..."
  rm -f "${CHART_PACKAGE}"
}

# ─── Main ─────────────────────────────────────────────────────────────────────
main() {
  parse_args "$@"
  resolve_chart_config
  login_registry
  mirror_chart
  logout_registry
  echo "✅ Successfully mirrored ${CHART_NAME}:${CHART_VERSION}"

}

main "$@"
