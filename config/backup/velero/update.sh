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

crdfile=templates/customresourcedefinitions.yaml
version=$(velero version --client-only | grep Version | cut -d' ' -f2)

echo "# This file has been generated with Velero $version. Do not edit." > $crdfile

# extract CRDs by calling Velero with dummy values
crds=$(
  velero install --bucket does-not-matter --use-restic --plugins gcp --provider gcp --secret-file Chart.yaml --dry-run -o json | \
  jq -c '.items | map(select(.kind=="CustomResourceDefinition")) | sort_by(.metadata.name) | .[]'
)

while IFS= read -r crd; do
  crd=$(
    echo "$crd" | \
    jq 'del(.metadata.creationTimestamp)' | \
    jq -S '.metadata.annotations["helm.sh/hook"] = "crd-install"' | \
    yq -P r -
  )

  echo -e "---\n$crd" >> $crdfile
done <<< "$crds"
