#!/usr/bin/env bash

# Copyright 2026 The Kubermatic Kubernetes Platform contributors.
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

### Helpers for hack/ci/run-mla-upgrade-e2e.sh. Source hack/lib.sh first.

# resolve_previous_minor_release prints the newest stable patch release of the
# minor before the one under test. On main that is the newest released minor,
# on release/vX.Y it is X.(Y-1). KKP_PREVIOUS_VERSION overrides the result.
# The tag list comes from the second argument (one tag per line) or from the
# kubermatic GitHub repository.
resolve_previous_minor_release() {
  local base_ref="${1:-${PULL_BASE_REF:-main}}"
  local tags="${2:-}"

  if [ -n "${KKP_PREVIOUS_VERSION:-}" ]; then
    echo "$KKP_PREVIOUS_VERSION"
    return 0
  fi

  if [ -z "$tags" ]; then
    tags="$(git ls-remote --tags --refs https://github.com/kubermatic/kubermatic.git 'v*' | awk -F/ '{print $3}')"
  fi

  local stable
  stable="$(echo "$tags" | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' | sort -V)"

  local minor
  if [[ $base_ref =~ ^release/v([0-9]+)\.([0-9]+)$ ]]; then
    minor="v${BASH_REMATCH[1]}.$((BASH_REMATCH[2] - 1))"
  else
    minor="$(echo "$stable" | tail -n 1 | cut -d. -f1-2)"
  fi

  local version
  version="$(echo "$stable" | grep -F "${minor}." | tail -n 1)"

  if [ -z "$version" ]; then
    echo "no stable release found for ${minor}" >&2
    return 1
  fi

  echo "$version"
}

# write_operator_values writes Helm values that point the kubermatic-operator at
# the given image tag.
write_operator_values() {
  local file="$1"
  local tag="$2"

  local reposuffix=""
  if [ "${KUBERMATIC_EDITION:-ee}" = "ee" ]; then
    reposuffix="-ee"
  fi

  local pull_secret=""
  if [ -n "${IMAGE_PULL_SECRET_DATA:-}" ]; then
    pull_secret="$(echo "$IMAGE_PULL_SECRET_DATA" | base64 --decode | jq --compact-output --monochrome-output '.')"
  fi

  cat > "$file" << VALUES
kubermaticOperator:
  image:
    repository: "quay.io/kubermatic/kubermatic${reposuffix}"
    tag: "${tag}"
  imagePullSecret: '${pull_secret}'
VALUES
}

# render_kubermatic_config writes the MLA KubermaticConfiguration with the
# CI secrets filled in. The file stays in mktemp, never in ARTIFACTS.
render_kubermatic_config() {
  local file="$1"

  local pull_secret=""
  if [ -n "${IMAGE_PULL_SECRET_DATA:-}" ]; then
    pull_secret="$(echo "$IMAGE_PULL_SECRET_DATA" | base64 --decode | jq --compact-output --monochrome-output '.')"
  fi

  cp hack/ci/testdata/kubermatic_mla.yaml "$file"
  sed -i "s;__SERVICE_ACCOUNT_KEY__;${SERVICE_ACCOUNT_KEY};g" "$file"
  sed -i "s;__IMAGE_PULL_SECRET__;${pull_secret};g" "$file"
  sed -i "s;__KUBERMATIC_DOMAIN__;${KUBERMATIC_DOMAIN:-ci.kubermatic.io};g" "$file"
}

# apply_seed creates the Seed that points at the kind cluster itself.
apply_seed() {
  local manifest
  manifest="$(mktemp)"
  local seed_kubeconfig
  seed_kubeconfig="$(sed 's/127.0.0.1.*/kubernetes.default.svc.cluster.local./' "$KUBECONFIG" | base64 -w0)"

  cp hack/ci/testdata/seed.yaml "$manifest"
  sed -i "s/__SEED_NAME__/$SEED_NAME/g" "$manifest"
  sed -i "s/__BUILD_ID__/$BUILD_ID/g" "$manifest"
  sed -i "s/__KUBECONFIG__/$seed_kubeconfig/g" "$manifest"

  retry 8 kubectl apply --filename "$manifest"
  retry 8 check_seed_ready kubermatic "$SEED_NAME"
}

# deploy_mla_stacks runs one installer for KKP, the Seed and both MLA stacks.
# Arguments: step label (used in JUnit file names, so no slashes), installer
# path, charts directory, operator values file, then extra installer flags
# (for example --helm-binary helm3).
deploy_mla_stacks() {
  local label="$1"
  local installer="$2"
  local charts="$3"
  local operator_values="$4"
  shift 4

  TEST_NAME="Deploy KKP (${label})"
  "$installer" deploy --disable-telemetry --charts-directory "$charts" --storageclass copy-default --config "$KUBERMATIC_CONFIG" --helm-values "$operator_values" "$@"
  retry 10 check_all_deployments_ready kubermatic

  TEST_NAME="Apply Seed (${label})"
  apply_seed
  retry 9 check_all_deployments_ready kubermatic

  TEST_NAME="Deploy seed-mla (${label})"
  "$installer" deploy seed-mla --charts-directory "$charts" --config "$KUBERMATIC_CONFIG" --helm-values hack/ci/testdata/seed-mla-values.yaml "$@"
  retry 8 check_all_deployments_ready monitoring

  TEST_NAME="Deploy usercluster-mla (${label})"
  "$installer" deploy usercluster-mla --charts-directory "$charts" --config "$KUBERMATIC_CONFIG" --helm-values hack/ci/testdata/usercluster-mla-upgrade-values.yaml --helm-timeout 1500s "$@"
  retry 8 check_all_deployments_ready mla
}
