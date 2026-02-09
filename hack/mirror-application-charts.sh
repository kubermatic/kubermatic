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

set -eo pipefail

# ─── Default registry and repo ────────────────────────────────────────────────
REGISTRY_HOST="${REGISTRY_HOST:-quay.io}"
REPOSITORY_PREFIX="${REPOSITORY_PREFIX:-kubermatic-mirror/helm-charts}"

# ─── Chart-specific configurations ────────────────────────────────────────────
# Format: key = chart name, value = "<URL_TEMPLATE>"
declare -A CHART_URLS=(
  ["cluster-autoscaler"]="https://github.com/kubernetes/autoscaler/releases/download/cluster-autoscaler-chart-%s/cluster-autoscaler-%s.tgz"
  ["cilium"]="https://helm.cilium.io/cilium-%s.tgz"
  # Add more charts here as needed
  ["aikit"]="https://sozercan.github.io/aikit/charts/aikit-%s.tgz"
  ["argo-cd"]="https://github.com/argoproj/argo-helm/releases/download/argo-cd-%s/argo-cd-%s.tgz"
  ["cert-manager"]="https://charts.jetstack.io/charts/cert-manager-%s.tgz"
  ["envoy-gateway"]="oci://docker.io/envoyproxy/gateway-helm"
  ["falco"]="https://github.com/falcosecurity/charts/releases/download/falco-%s/falco-%s.tgz"
  ["flux2"]="https://github.com/fluxcd-community/helm-charts/releases/download/flux2-%s/flux2-%s.tgz"
  ["k8sgpt-operator"]="https://charts.k8sgpt.ai/k8sgpt-operator-%s.tgz"
  ["kube-vip"]="https://github.com/kube-vip/helm-charts/releases/download/kube-vip-%s/kube-vip-%s.tgz"
  ["metallb"]="https://github.com/metallb/metallb/releases/download/metallb-chart-%s/metallb-%s.tgz"
  ["ingress-nginx"]="https://github.com/kubernetes/ingress-nginx/releases/download/helm-chart-%s/ingress-nginx-%s.tgz"
  ["gpu-operator"]="https://github.com/NVIDIA/gpu-operator/charts/gpu-operator-%s.tgz"
  ["trivy"]="https://github.com/aquasecurity/helm-charts/releases/download/trivy-%s/trivy-%s.tgz"
  ["trivy-operator"]="https://github.com/aquasecurity/helm-charts/releases/download/trivy-operator-%s/trivy-operator-%s.tgz"
  ["local-ai"]="https://github.com/go-skynet/helm-charts/releases/download/local-ai-%s/local-ai-%s.tgz"
  ["kueue"]="oci://registry.k8s.io/kueue/charts/kueue"
  ["mcp-server-kubernetes"]="oci://ghcr.io/flux159/mcp-server-kubernetes"
)

# Default versions for each chart
declare -A CHART_VERSIONS=(
  ["cluster-autoscaler"]="9.46.6"
  ["cilium"]="1.18.6"
  # Add more default versions here as needed
  ["aikit"]="0.18.0"
  ["argo-cd"]="8.0.0"
  ["cert-manager"]="v1.17.2"
  ["envoy-gateway"]="v1.7.0"
  ["falco"]="4.21.2"
  ["flux2"]="2.15.0"
  ["k8sgpt-operator"]="0.2.17"
  ["kube-vip"]="0.6.6"
  ["metallb"]="0.14.9"
  ["ingress-nginx"]="4.14.3"
  ["gpu-operator"]="v25.3.0"
  ["trivy"]="0.14.1"
  ["trivy-operator"]="0.28.0"
  ["local-ai"]="3.4.2"
  ["kueue"]="0.13.4"
  ["mcp-server-kubernetes"]="2.9.9"
)

# Re-enable unset variable checking after array declarations
set -u

# ─── Usage ────────────────────────────────────────────────────────────────────
usage() {
  echo "Usage: $0 <chart-name> [version (optional)]"
  echo "Supported charts:"
  for key in "${!CHART_URLS[@]}"; do echo "  - $key"; done
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

  if [[ -z "${CHART_URLS[$CHART_NAME]:-}" ]]; then
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

  REGISTRY_USER="${REGISTRY_USER:-$(vault kv get -field=username dev/kubermatic-mirror-quay.io)}"
  REGISTRY_PASSWORD="${REGISTRY_PASSWORD:-$(vault kv get -field=password dev/kubermatic-mirror-quay.io)}"

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
  if [[ "${CHART_SOURCE}" == oci://* ]]; then
    # For OCI registries, use version-specific pull
    helm pull "${CHART_SOURCE}" --version "${CHART_VERSION}" --destination ./
  else
    # For HTTP/HTTPS URLs, use direct URL
    helm pull "${CHART_SOURCE}" --destination ./
  fi

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
