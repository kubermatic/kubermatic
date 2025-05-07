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

set -euo pipefail

cd $(dirname $0)/..
source hack/lib.sh

if [ "$#" -lt 1 ] || [ "$1" == "--help" ]; then
  cat << EOF
Usage: $(basename $0) (version)
where:
    version:    Kyverno version used to generate the CRDs, e.g. v1.14.1
e.g: $(basename $0) v1.14.1
EOF
  exit 0
fi

containerize ./hack/update-kyverno-crds.sh $1

# Handle containerized execution
if is_containerized; then
  version="$2"
else
  version="$1"
fi

crd_dir="pkg/ee/kyverno/resources/user-cluster/static/kyverno-crds"
mkdir -p "$crd_dir"
cd "$crd_dir"

# Clean existing CRDs
rm -f *.yaml

echodate "Downloading Kyverno CRDs for version $version..."
url="https://github.com/kyverno/kyverno/releases/download/$version/install.yaml"

# Download and split CRDs into separate files
wget -qO- "$url" |
  yq eval-all --split-exp '.spec.names.plural' 'select(.kind == "CustomResourceDefinition")' - |
  while IFS= read -r file; do
    if [ -f "$file" ]; then
      name=$(basename "$file" .yaml)
      echodate "Processing CRD for $name..."
      echo "# This file has been generated with Kyverno $version. Do not edit." > "${name}.yaml"
      cat "$file" | yq e 'del(.metadata.creationTimestamp)' - >> "${name}.yaml"
      rm "$file"
    fi
  done

echodate "Successfully updated Kyverno CRDs in $crd_dir"
