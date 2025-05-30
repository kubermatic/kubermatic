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

# Default version
DEFAULT_VERSION="release-1.14"

if [ "$#" -eq 0 ]; then
  version="$DEFAULT_VERSION"
  echodate "No version specified, using default: $version"
elif [ "$1" == "--help" ]; then
  cat << EOF
Usage: $(basename $0) [version]
where:
    version:    Kyverno policies release tag to fetch, e.g. $DEFAULT_VERSION (default if not specified)
e.g: $(basename $0) $DEFAULT_VERSION
EOF
  exit 0
else
  version="$1"
fi

policies_dir="pkg/ee/default-policy-catalog/policies"
mkdir -p "$policies_dir"

echodate "Cleaning existing policies directory..."
rm -f "$policies_dir"/*.yml "$policies_dir"/*.yaml

echodate "Downloading Kyverno pod-security/enforce policies for $version..."
cd "$policies_dir"
kustomize build "https://github.com/kyverno/policies/pod-security/enforce?ref=$version" |
  yq eval-all --split-exp '.metadata.name + ".yml"' 'select(.kind == "ClusterPolicy")' -

echodate "Policies have been successfully downloaded. To convert them to PolicyTemplates, build and run the application."
