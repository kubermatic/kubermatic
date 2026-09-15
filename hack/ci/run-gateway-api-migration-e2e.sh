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

### Upgrade-path E2E that exercises --clean-nginx-lb after a real 2.29 -> 2.30 -> 2.31
### upgrade sequence. Two cases run sequentially against fresh kind clusters:
###
###   Case 1 (2.30 already on Gateway API):
###     1. Install KKP v2.29.x (nginx-ingress era).
###     2. Upgrade to KKP v2.30.x with --migrate-gateway-api.
###     3. Re-run the installer without --migrate-gateway-api (no-op flag).
###     4. Re-run the installer with --clean-nginx-lb to tear down nginx.
###
###   Case 2 (2.30 stays on nginx, late migrator):
###     1. Install KKP v2.29.x.
###     2. Upgrade to KKP v2.30.x WITHOUT --migrate-gateway-api (stay on nginx).
###     3. Run the installer without --migrate-gateway-api.
###     4. Run the installer without --migrate-gateway-api again (idempotency).
###     5. Run the installer with --clean-nginx-lb.
###
###   Case 3 (late DNS flipper, --skip-ingress-cleanup staging):
###     1. Install KKP v2.30.x with nginx-ingress (no --migrate-gateway-api).
###     2. Run PR installer with --skip-ingress-cleanup — Gateway API stands up alongside nginx and legacy Ingresses stay.
###     3. Verify legacy Ingress objects are still present.
###     4. Verify PR installer rejects --skip-ingress-cleanup + --clean-nginx-lb combined.
###     5. Run PR installer without flags — Ingresses removed.
###     6. Run PR installer with --clean-nginx-lb — nginx-ingress-controller release gone.
###
### After each case the script verifies the nginx-ingress-controller release, namespace,
### and legacy Ingress objects are gone, and that Gateway API resources are healthy.

set -euo pipefail

cd $(dirname $0)/../..
source hack/lib.sh
source hack/ci/lib-gateway-api-migration.sh

run_case_1() {
  echodate "================================================================"
  echodate " Case 1: 2.29 -> 2.30 (Gateway API enabled) -> 2.31 cleanup"
  echodate "================================================================"

  setup_kkp_migration_environment

  deploy_kkp_v229
  run_pre_migration_tests "case1-v229"

  deploy_kkp_v230_with_gateway_api_flag

  echodate "Case 1 / Step 3: re-running PR installer without --migrate-gateway-api (flag is a no-op)..."
  deploy_kkp_under_test

  echodate "Case 1 / Step 4: re-running PR installer with --clean-nginx-lb..."
  deploy_kkp_under_test --clean-nginx-lb

  verify_cleanup_state
}

run_case_2() {
  echodate "================================================================"
  echodate " Case 2: 2.29 -> 2.30 (still on nginx) -> 2.31 cleanup"
  echodate "================================================================"

  setup_kkp_migration_environment

  deploy_kkp_v229
  run_pre_migration_tests "case2-v229"

  deploy_kkp_v230_without_gateway_api_flag
  run_pre_migration_tests "case2-v230-nginx"

  echodate "Case 2 / Step 3: running installer without --migrate-gateway-api (first PR-installer run)..."
  deploy_kkp_under_test

  echodate "Case 2 / Step 4: running installer without --migrate-gateway-api again (idempotency)..."
  deploy_kkp_under_test

  echodate "Case 2 / Step 5: running installer with --clean-nginx-lb..."
  deploy_kkp_under_test --clean-nginx-lb

  verify_cleanup_state
}

run_case_3() {
  echodate "================================================================"
  echodate " Case 3: 2.30 (nginx) -> 2.31 --skip-ingress-cleanup staging"
  echodate "================================================================"

  setup_kkp_migration_environment

  deploy_kkp_v230_without_gateway_api_flag
  run_pre_migration_tests "case3-v230-nginx"

  # Steps 2-4 run with hybrid values (Gateway API config + dex.ingress.enabled: true)
  # so the Dex helm chart does not remove the legacy Dex Ingress via reconciliation
  # while nginx is still expected to serve traffic.
  echodate "Case 3 / Step 2: PR installer with --skip-ingress-cleanup (hybrid values keep dex.ingress.enabled: true)..."
  HELM_VALUES_FILE_UNDER_TEST_OVERRIDE="${HELM_VALUES_FILE_UNDER_TEST_HYBRID}" \
    deploy_kkp_under_test --skip-ingress-cleanup

  echodate "Case 3 / Step 3: verify legacy Ingress objects are still present..."
  verify_legacy_ingresses_still_present "case3-after-skip"

  echodate "Case 3 / Step 4: verify PR installer rejects --skip-ingress-cleanup + --clean-nginx-lb..."
  HELM_VALUES_FILE_UNDER_TEST_OVERRIDE="${HELM_VALUES_FILE_UNDER_TEST_HYBRID}" \
    expect_installer_rejects_skip_and_clean_combo

  echodate "Case 3 / Step 5: PR installer default (Ingresses now removed)..."
  deploy_kkp_under_test

  echodate "Case 3 / Step 6: PR installer with --clean-nginx-lb..."
  deploy_kkp_under_test --clean-nginx-lb

  verify_cleanup_state
}

run_case_1
reset_kind_cluster
run_case_2
reset_kind_cluster
run_case_3

echodate "All three upgrade-path cases completed successfully!"
