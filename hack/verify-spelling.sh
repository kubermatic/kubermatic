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

set -euo pipefail

cd $(dirname $0)/..
source hack/lib.sh

CONTAINERIZE_IMAGE=quay.io/kubermatic/build:go-1.24-node-20-4 containerize ./hack/verify-spelling.sh

echodate "Running codespell..."

skip=(
  .git
  _build
  _dist
  vendor
  go.mod
  go.sum
  .typos.toml
  *.jpg
  *.jpeg
  *.png
  *.woff
  *.woff2
  *.pem
  ./charts/cert-manager/crd
  ./charts/backup/velero/crd
  ./charts/dex/test
  ./charts/oauth/test
  ./addons/multus/crds.yaml
  ./charts/local-kubevirt/crds
  ./pkg/applicationdefinitions/system-applications
  ./pkg/ee/default-application-catalog/applicationdefinitions
  ./pkg/resources/test/fixtures
  ./pkg/ee/kyverno/resources/user-cluster/static
  ./pkg/crd/k8s.io
)

function join {
  local IFS=","
  echo "$*"
}

time codespell \
  --skip "$(join ${skip[@]})" \
  --ignore-words .codespell.exclude \
  --check-filenames \
  --check-hidden

echodate "No problems detected."
