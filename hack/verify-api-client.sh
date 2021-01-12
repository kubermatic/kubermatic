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

function cleanup() {
  if [[ -n "${TMP_DIFFROOT:-}" ]]; then
    rm -rf "${TMP_DIFFROOT}"
  fi
}
trap cleanup EXIT SIGINT SIGTERM

ROOT_DIR="$(realpath .)"
DIFFROOT="${ROOT_DIR}/pkg/test/e2e/utils/apiclient"

TMP_DIFFROOT=$(mktemp -d)

cp -a "${DIFFROOT}"/* "${TMP_DIFFROOT}"

echodate "Generating API client"
./hack/gen-api-client.sh

echodate "Diffing ${DIFFROOT} against freshly generated api client"
ret=0
diff -Naupr "${DIFFROOT}" "${TMP_DIFFROOT}" || ret=$?
cp -a "${TMP_DIFFROOT}"/client "${DIFFROOT}"
cp -a "${TMP_DIFFROOT}"/models "${DIFFROOT}"

if [[ $ret -eq 0 ]]; then
  echodate "${DIFFROOT} up to date."
else
  echodate "${DIFFROOT} is out of date. Please run hack/gen-api-client.sh"
  exit 1
fi
