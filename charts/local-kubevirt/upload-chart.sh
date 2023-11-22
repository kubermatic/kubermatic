#!/usr/bin/env bash

# Copyright 2023 The Kubermatic Kubernetes Platform contributors.
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

REGISTRY_HOST="${REGISTRY_HOST:-quay.io}"
REPOSITORY_PREFIX="${REPOSITORY_PREFIX:-kubermatic/helm-charts}"
CHART_PACKAGE="${CHART_PACKAGE:-kubevirt}"

echo "Uploading KubeVirt chart to OCI registry ${REGISTRY_HOST}/${REPOSITORY_PREFIX}"

if [ -z "${VAULT_ADDR:-}" ]; then
  export VAULT_ADDR=https://vault.kubermatic.com/
fi
REGISTRY_USER="${REGISTRY_USER:-$(vault kv get -field=username dev/kubermatic-quay.io)}"
REGISTRY_PASSWORD="${REGISTRY_PASSWORD:-$(vault kv get -field=password dev/kubermatic-quay.io)}"

echo ${REGISTRY_PASSWORD} | helm registry login ${REGISTRY_HOST} --username ${REGISTRY_USER} --password-stdin

cd -- "$( dirname -- "${BASH_SOURCE[0]}" )"
CHART_PACKAGE=$(helm package . | sed 's|.*\(kubevirt-v[0-9.]*\.tgz\)|\1|')
helm push ${CHART_PACKAGE} oci://${REGISTRY_HOST}/${REPOSITORY_PREFIX}

helm registry logout ${REGISTRY_HOST}
