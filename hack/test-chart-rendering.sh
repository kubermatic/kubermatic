#!/usr/bin/env bash

# Copyright 2021 The Kubermatic Kubernetes Platform contributors.
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

### Used for "golden master" testing.
### Symlink this script to a directory called "test" in the root directory of a chart.
### Prepared test fixtures should end with .yaml, results will end with .yaml.out.
### Script exits with 1 if the output of rendering is different than what is stored the .yaml.out file

# root directory of a chart
BASEDIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/.."
# real directory where script resides
REALDIR="$(cd "$(dirname $(readlink -f "${BASH_SOURCE[0]}"))" && pwd)"
source ${REALDIR}/lib.sh

cd ${BASEDIR}
chartname=$(yq '.name' Chart.yaml)
echodate "(Golden master) Testing ${chartname}..."
exitCode=0

set +o errexit
helm version
echodate "Fetching dependencies..."
i=0
for url in $(yq '.dependencies.[].repository' Chart.yaml); do
  # Remove quotes from the URL
  url=${url//\"/}
  # Skip OCI repositories as they don't need to be added to helm repos
  if [[ $url == oci://* ]]; then
    echo "Skipping OCI repository: ${url}"
    continue
  fi
  i=$((i + 1))
  helm repo add ${chartname}-dep-${i} ${url}
done

helm dependency build .

echodate "Testing template commands"
for valuesFile in test/*.yaml; do
  echodate "- evaluating ${valuesFile}... "
  helm template . --values ${valuesFile} > ${valuesFile}.out
  helmOut=$?

  if ! git diff --quiet ${valuesFile}.out || [[ $helmOut -ne 0 ]]; then
    exitCode=1
    git --no-pager diff ${valuesFile}.out
    echodate "  FAIL"
  else
    echodate "  PASS"
  fi
done
echodate "Cleaning up dependencies..."
while [[ $i -gt 0 ]]; do
  helm repo remove ${chartname}-dep-${i}
  i=$((i - 1))
done

exit ${exitCode}
