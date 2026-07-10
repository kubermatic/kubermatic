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

### Helpers shared by run-gateway-api-migration-e2e.sh (Case 1) and
### run-gateway-api-migration-case2-e2e.sh (Case 2). All functions assume the
### working directory is the repository root and that hack/lib.sh is sourced.

# Latest patch releases of the two prior KKP minors that are involved in the
# upgrade path. Can be overridden via env vars when a newer patch ships.
: "${KKP_V229_VERSION:=v2.29.8}"
: "${KKP_V230_VERSION:=v2.30.4}"

: "${KUBERMATIC_EDITION:=ee}"
: "${KIND_CLUSTER_NAME:=${SEED_NAME:-kubermatic}}"
: "${KUBERMATIC_DOMAIN:=worker.ci.k8c.io}"

INSTALL_DIR_V229="${INSTALL_DIR_V229:-/tmp/kkp-${KKP_V229_VERSION}}"
INSTALL_DIR_V230="${INSTALL_DIR_V230:-/tmp/kkp-${KKP_V230_VERSION}}"

DEX_PASSWORD_HASH='$2y$10$Lurps56wlfD5Rgelz9u4FuYOMdUw8FZaIKyt5xUyPBwHP0Eo.yLhW'

download_kkp_release() {
  local version="$1"
  local target_dir="$2"

  if [ -x "${target_dir}/kubermatic-installer" ]; then
    echodate "${version} installer already present at ${target_dir}, skipping download."
    return 0
  fi

  echodate "Downloading KKP ${version} (${KUBERMATIC_EDITION}) release to ${target_dir}..."
  mkdir -p "${target_dir}"

  local tarball="${target_dir}/kubermatic-${KUBERMATIC_EDITION}-${version}-linux-amd64.tar.gz"
  local url="https://github.com/kubermatic/kubermatic/releases/download/${version}/kubermatic-${KUBERMATIC_EDITION}-${version}-linux-amd64.tar.gz"

  if ! curl --fail --location --silent --show-error --output "${tarball}" "${url}"; then
    echodate "Failed to download KKP ${version} (${KUBERMATIC_EDITION}) tarball from ${url}. Check the version is published and the URL is reachable."
    return 1
  fi

  if ! tar -xzf "${tarball}" --directory "${target_dir}/"; then
    echodate "Downloaded KKP ${version} tarball is not a valid gzip archive: ${tarball}"
    return 1
  fi

  if [ ! -x "${target_dir}/kubermatic-installer" ]; then
    echodate "Failed to extract installer binary from ${version} release tarball"
    return 1
  fi
}

write_migration_helm_values() {
  local file="$1"
  local migrate_gateway_api="$2" # "true" or "false"
  local image_tag="${3:-}"       # optional explicit operator image tag

  local image_pull_secret_inline=""
  if [ -n "${IMAGE_PULL_SECRET_DATA:-}" ]; then
    image_pull_secret_inline="$(echo "$IMAGE_PULL_SECRET_DATA" | base64 --decode | jq --compact-output --monochrome-output '.')"
  fi

  local reposuffix=""
  if [ "${KUBERMATIC_EDITION}" = "ee" ]; then
    reposuffix="-ee"
  fi

  cat > "${file}" << EOF
kubermaticOperator:
  image:
    repository: "quay.io/kubermatic/kubermatic${reposuffix}"
EOF
  if [ -n "${image_tag}" ]; then
    cat >> "${file}" << EOF
    tag: "${image_tag}"
EOF
  fi
  cat >> "${file}" << EOF
  imagePullSecret: '${image_pull_secret_inline}'

minio:
  credentials:
    accessKey: test
    secretKey: testtest
EOF

  # migrate_gateway_api values:
  #   "true"   Gateway API enabled, dex.ingress disabled (fresh Gateway API state)
  #   "false"  nginx-ingress mode, dex.ingress enabled (legacy state)
  #   "hybrid" Gateway API enabled AND dex.ingress enabled — models the staged
  #            --skip-ingress-cleanup path where Envoy comes up alongside nginx
  #            and both data planes coexist until DNS is flipped.
  if [ "${migrate_gateway_api}" = "true" ] || [ "${migrate_gateway_api}" = "hybrid" ]; then
    cat >> "${file}" << EOF

migrateGatewayAPI: true

httpRoute:
  gatewayName: kubermatic
  gatewayNamespace: kubermatic
  domain: ${KUBERMATIC_DOMAIN}
  timeout: 3600s

envoyProxy:
  service:
    type: NodePort
    externalTrafficPolicy: Cluster
    patch:
      type: JSONMerge
      value:
        spec:
          type: NodePort
          ports:
            - name: http
              port: 80
              nodePort: 30080
              targetPort: 10080
            - name: https
              port: 443
              nodePort: 30443
              targetPort: 10443
EOF
  fi

  if [ "${migrate_gateway_api}" = "true" ]; then
    cat >> "${file}" << EOF

dex:
  replicaCount: 1
  ingress:
    enabled: false
    hosts: []
    tls: []
EOF
  else
    cat >> "${file}" << EOF

dex:
  replicaCount: 1
  ingress:
    enabled: true
    className: nginx
    hosts:
      - host: ${KUBERMATIC_DOMAIN}
        paths:
          - path: /dex
            pathType: ImplementationSpecific
    tls: []
    annotations:
      cert-manager.io/cluster-issuer: letsencrypt-staging
EOF
  fi

  cat >> "${file}" << EOF

  config:
    issuer: https://${KUBERMATIC_DOMAIN}/dex
    enablePasswordDB: true
    staticPasswords:
      - email: kubermatic@example.com
        hash: ${DEX_PASSWORD_HASH}
        username: admin
    staticClients:
      - id: kubermatic
        name: Kubermatic
        secret: kubermatic-static-client-secret
        RedirectURIs:
          - https://${KUBERMATIC_DOMAIN}
          - https://${KUBERMATIC_DOMAIN}/projects

nginx:
  controller:
    replicaCount: 1

telemetry:
  uuid: "559a1b90-b5d0-40aa-a74d-bd9e808ec10f"
  schedule: "* * * * *"
  reporterArgs:
    - stdout
    - --client-uuid=\$(CLIENT_UUID)
    - --record-dir=\$(RECORD_DIR)
EOF
}

reset_kind_cluster() {
  echodate "Tearing down kind cluster ${KIND_CLUSTER_NAME} between cases..."
  kind delete cluster --name "${KIND_CLUSTER_NAME}" || true
  unset KUBERMATIC_CONFIG \
    HELM_VALUES_FILE_V229_NGINX \
    HELM_VALUES_FILE_V230_NGINX \
    HELM_VALUES_FILE_V230_GATEWAY_API \
    HELM_VALUES_FILE_UNDER_TEST \
    HELM_VALUES_FILE_UNDER_TEST_HYBRID \
    HELM_VALUES_FILE_UNDER_TEST_OVERRIDE
}

build_kkp_under_test() {
  local reposuffix=""
  if [ "${KUBERMATIC_EDITION}" = "ee" ]; then
    reposuffix="-ee"
  fi
  local kkp_image="quay.io/kubermatic/kubermatic${reposuffix}:${KUBERMATIC_VERSION}"
  local addons_image="quay.io/kubermatic/addons:${KUBERMATIC_VERSION}"

  if [ ! -x ./_build/kubermatic-installer ]; then
    TEST_NAME="Build Kubermatic binaries"
    echodate "Building PR binaries for ${KUBERMATIC_VERSION}..."
    local before
    before=$(nowms)
    retry 1 make build
    pushElapsed kubermatic_go_build_duration_milliseconds "${before}"
  else
    echodate "PR installer already built; skipping rebuild."
  fi

  if ! docker image inspect "${kkp_image}" > /dev/null 2>&1; then
    TEST_NAME="Build Kubermatic Docker image"
    echodate "Building ${kkp_image}..."
    retry 5 docker build -t "${kkp_image}" .
  fi
  if ! docker image inspect "${addons_image}" > /dev/null 2>&1; then
    TEST_NAME="Build addons Docker image"
    echodate "Building ${addons_image}..."
    (cd addons && retry 5 docker build -t "${addons_image}" .)
  fi

  TEST_NAME="Load Kubermatic images into kind"
  echodate "Loading PR images into kind cluster ${KIND_CLUSTER_NAME}..."
  retry 5 kind load docker-image "${kkp_image}" --name "${KIND_CLUSTER_NAME}"
  retry 5 kind load docker-image "${addons_image}" --name "${KIND_CLUSTER_NAME}"
  echodate "PR binaries and images ready."
}

setup_kkp_migration_environment() {
  TEST_NAME="Pre-warm Go build cache"
  echodate "Attempting to pre-warm Go build cache"
  local before
  before=$(nowms)
  make download-gocache
  pushElapsed gocache_download_duration_milliseconds "${before}"

  export KIND_CLUSTER_NAME
  export KUBERMATIC_YAML=hack/ci/testdata/kubermatic_nginx.yaml

  echodate "Bringing up kind cluster ${KIND_CLUSTER_NAME}..."
  source hack/ci/setup-kind-cluster.sh

  if [ -n "${IMAGE_PULL_SECRET_DATA:-}" ]; then
    echo "$IMAGE_PULL_SECRET_DATA" | base64 -d > /config.json
  fi

  protokol --kubeconfig "$KUBECONFIG" --flat --output "$ARTIFACTS/logs/cluster-control-plane" --namespace 'cluster-*' > /dev/null 2>&1 &
  protokol --kubeconfig "$KUBECONFIG" --flat --output "$ARTIFACTS/logs/kubermatic" --namespace kubermatic > /dev/null 2>&1 &
  protokol --kubeconfig "$KUBECONFIG" --flat --output "$ARTIFACTS/logs/nginx-ingress" --namespace nginx-ingress-controller > /dev/null 2>&1 &

  export KUBERMATIC_VERSION="${KUBERMATIC_VERSION:-$(git rev-parse HEAD)}"
  build_kkp_under_test

  download_kkp_release "${KKP_V229_VERSION}" "${INSTALL_DIR_V229}"
  download_kkp_release "${KKP_V230_VERSION}" "${INSTALL_DIR_V230}"

  # KUBERMATIC_CONFIG gets the substituted serviceAccountKey and dockerconfig
  # imagePullSecret. Render it to /tmp (mktemp) so it is NOT uploaded with the
  # prow ARTIFACTS bucket — that bucket is world-readable for CI debugging.
  export KUBERMATIC_CONFIG="$(mktemp)"
  cp hack/ci/testdata/kubermatic_nginx.yaml "${KUBERMATIC_CONFIG}"

  # The KubermaticConfiguration template contains placeholders that must be filled
  # in before any installer can validate it; see setup-kubermatic-in-kind.sh for the
  # canonical substitution. SERVICE_ACCOUNT_KEY and IMAGE_PULL_SECRET_DATA are wired
  # in by the Prow preset-e2e-ci secret; for local runs supply them via env vars.
  if [ -z "${SERVICE_ACCOUNT_KEY:-}" ]; then
    echodate "FAIL: SERVICE_ACCOUNT_KEY is not set; KubermaticConfiguration cannot be rendered."
    return 1
  fi
  local image_pull_secret_inline=""
  if [ -n "${IMAGE_PULL_SECRET_DATA:-}" ]; then
    image_pull_secret_inline="$(echo "$IMAGE_PULL_SECRET_DATA" | base64 --decode | jq --compact-output --monochrome-output '.')"
  fi

  sed -i "s;__SERVICE_ACCOUNT_KEY__;${SERVICE_ACCOUNT_KEY};g" "${KUBERMATIC_CONFIG}"
  sed -i "s;__IMAGE_PULL_SECRET__;${image_pull_secret_inline};g" "${KUBERMATIC_CONFIG}"
  sed -i "s;__KUBERMATIC_DOMAIN__;${KUBERMATIC_DOMAIN};g" "${KUBERMATIC_CONFIG}"

  # Helm values inline the decoded dockerconfigjson for the image pull secret.
  # Keep them in /tmp (mktemp) so they are NOT uploaded to the prow ARTIFACTS
  # bucket — that bucket is world-readable for CI debugging.
  export HELM_VALUES_FILE_V229_NGINX="$(mktemp)"
  export HELM_VALUES_FILE_V230_NGINX="$(mktemp)"
  export HELM_VALUES_FILE_V230_GATEWAY_API="$(mktemp)"
  export HELM_VALUES_FILE_UNDER_TEST="$(mktemp)"
  # Hybrid values: Gateway API config + dex.ingress.enabled: true. Used by
  # Case 3 during the --skip-ingress-cleanup phase so the Dex chart doesn't
  # wipe the legacy Dex Ingress via helm reconciliation while Envoy stands up.
  export HELM_VALUES_FILE_UNDER_TEST_HYBRID="$(mktemp)"
  write_migration_helm_values "${HELM_VALUES_FILE_V229_NGINX}" "false" "${KKP_V229_VERSION}"
  write_migration_helm_values "${HELM_VALUES_FILE_V230_NGINX}" "false" "${KKP_V230_VERSION}"
  write_migration_helm_values "${HELM_VALUES_FILE_V230_GATEWAY_API}" "true" "${KKP_V230_VERSION}"
  write_migration_helm_values "${HELM_VALUES_FILE_UNDER_TEST}" "true" "${KUBERMATIC_VERSION}"
  write_migration_helm_values "${HELM_VALUES_FILE_UNDER_TEST_HYBRID}" "hybrid" "${KUBERMATIC_VERSION}"
}

deploy_kkp_v229() {
  echodate "Step 1: deploying KKP ${KKP_V229_VERSION} (nginx-ingress era)..."
  TEST_NAME="Install KKP ${KKP_V229_VERSION}"
  "${INSTALL_DIR_V229}/kubermatic-installer" deploy kubermatic-master \
    --charts-directory "${INSTALL_DIR_V229}/charts" \
    --storageclass copy-default \
    --config "${KUBERMATIC_CONFIG}" \
    --helm-values "${HELM_VALUES_FILE_V229_NGINX}"

  retry 10 check_all_deployments_ready kubermatic
  retry 10 check_all_deployments_ready nginx-ingress-controller
  echodate "KKP ${KKP_V229_VERSION} installed and nginx-ingress is healthy."
}

deploy_kkp_v230_with_gateway_api_flag() {
  echodate "Step 2: upgrading to KKP ${KKP_V230_VERSION} with --migrate-gateway-api..."
  TEST_NAME="Upgrade to KKP ${KKP_V230_VERSION} (Gateway API enabled)"
  "${INSTALL_DIR_V230}/kubermatic-installer" deploy kubermatic-master \
    --charts-directory "${INSTALL_DIR_V230}/charts" \
    --storageclass copy-default \
    --config "${KUBERMATIC_CONFIG}" \
    --helm-values "${HELM_VALUES_FILE_V230_GATEWAY_API}" \
    --migrate-gateway-api

  retry 10 check_all_deployments_ready kubermatic
  retry 10 check_all_deployments_ready envoy-gateway-controller
  echodate "KKP ${KKP_V230_VERSION} installed; Gateway API and nginx-ingress coexist."
}

deploy_kkp_v230_without_gateway_api_flag() {
  echodate "Step 2: upgrading to KKP ${KKP_V230_VERSION} WITHOUT --migrate-gateway-api (stays on nginx)..."
  TEST_NAME="Upgrade to KKP ${KKP_V230_VERSION} (nginx mode)"
  "${INSTALL_DIR_V230}/kubermatic-installer" deploy kubermatic-master \
    --charts-directory "${INSTALL_DIR_V230}/charts" \
    --storageclass copy-default \
    --config "${KUBERMATIC_CONFIG}" \
    --helm-values "${HELM_VALUES_FILE_V230_NGINX}"

  retry 10 check_all_deployments_ready kubermatic
  retry 10 check_all_deployments_ready nginx-ingress-controller
  echodate "KKP ${KKP_V230_VERSION} installed; remains on nginx-ingress."
}

deploy_kkp_under_test() {
  local extra_flags=("$@")
  # When the caller sets HELM_VALUES_FILE_UNDER_TEST_OVERRIDE, use that values
  # file instead of the default under-test one. This lets Case 3 keep the
  # existing 2.30 nginx values (dex.ingress.enabled: true, etc.) while running
  # the PR installer with --skip-ingress-cleanup, mirroring what an admin would
  # do in production during a staged DNS-flip migration.
  local values_file="${HELM_VALUES_FILE_UNDER_TEST_OVERRIDE:-${HELM_VALUES_FILE_UNDER_TEST}}"
  echodate "Running KKP installer under test (this PR) with flags: ${extra_flags[*]:-<none>}, values: ${values_file}..."
  TEST_NAME="KKP installer under test (${extra_flags[*]:-default})"
  ./_build/kubermatic-installer deploy kubermatic-master \
    --storageclass copy-default \
    --config "${KUBERMATIC_CONFIG}" \
    --helm-values "${values_file}" \
    --skip-seed-validation=kubermatic \
    --verbose \
    "${extra_flags[@]}"

  retry 10 check_all_deployments_ready kubermatic
  retry 10 check_all_deployments_ready envoy-gateway-controller
}

verify_legacy_ingresses_still_present() {
  local phase_label="$1"
  echodate "Verifying legacy Ingress objects remain after --skip-ingress-cleanup (${phase_label})..."

  local expected=("kubermatic/kubermatic" "dex/dex")
  local missing=()
  for entry in "${expected[@]}"; do
    local ns="${entry%%/*}"
    local name="${entry##*/}"
    if ! kubectl -n "${ns}" get ingress "${name}" > /dev/null 2>&1; then
      missing+=("${entry}")
    fi
  done

  if [ "${#missing[@]}" -gt 0 ]; then
    echodate "FAIL: expected these legacy Ingresses to remain after --skip-ingress-cleanup, but they were deleted: ${missing[*]}"
    return 1
  fi
  echodate "Legacy Ingress objects are still in place, as expected."
}

expect_installer_rejects_skip_and_clean_combo() {
  echodate "Verifying installer rejects --skip-ingress-cleanup + --clean-nginx-lb combination..."
  TEST_NAME="Installer rejects --skip-ingress-cleanup + --clean-nginx-lb"

  local values_file="${HELM_VALUES_FILE_UNDER_TEST_OVERRIDE:-${HELM_VALUES_FILE_UNDER_TEST}}"

  set +e
  ./_build/kubermatic-installer deploy kubermatic-master \
    --storageclass copy-default \
    --config "${KUBERMATIC_CONFIG}" \
    --helm-values "${values_file}" \
    --skip-seed-validation=kubermatic \
    --skip-ingress-cleanup \
    --clean-nginx-lb \
    --verbose > /tmp/installer-reject.log 2>&1
  local rc=$?
  set -e

  if [ "${rc}" -eq 0 ]; then
    echodate "FAIL: installer exited 0 with --skip-ingress-cleanup + --clean-nginx-lb; expected non-zero"
    cat /tmp/installer-reject.log
    return 1
  fi
  if ! grep -q "skip-ingress-cleanup cannot be combined with --clean-nginx-lb" /tmp/installer-reject.log; then
    echodate "FAIL: installer failed but did not emit the expected rejection message"
    cat /tmp/installer-reject.log
    return 1
  fi
  echodate "Installer correctly rejected the flag combination."
}

run_pre_migration_tests() {
  local phase_label="$1"
  echodate "Running pre-migration tests (Ingress mode) — ${phase_label}..."
  TEST_NAME="Pre-migration tests (${phase_label})"
  go_test "gateway_api_premigration_${phase_label}" -timeout 1h -tags e2e -v ./pkg/test/e2e/gateway-api \
    -test.run "TestGatewayAPIPreMigration"
  echodate "Pre-migration tests passed (${phase_label})."
}

verify_cleanup_state() {
  echodate "Verifying cleanup state..."

  if helm status -n nginx-ingress-controller nginx-ingress-controller > /dev/null 2>&1; then
    echodate "FAIL: nginx-ingress-controller Helm release still exists after --clean-nginx-lb"
    return 1
  fi
  echodate "nginx-ingress-controller Helm release is gone."

  protokol --kubeconfig "$KUBECONFIG" --flat --output "$ARTIFACTS/logs/envoy-gateway" --namespace envoy-gateway-controller > /dev/null 2>&1 &

  echodate "Running cleanup verification tests..."
  go_test gateway_api_migration_cleanup -timeout 1h -tags e2e -v ./pkg/test/e2e/gateway-api \
    -test.run "TestGatewayAPIPostMigration|TestNginxIngressControllerCleanedUp|TestLegacyIngressesCleanedUp"

  echodate "Migration cleanup verification passed."
}
