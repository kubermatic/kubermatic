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

cd charts/backup/velero/

version=$(velero version --client-only | grep Version | cut -d' ' -f2)
crds=$(velero install --crds-only --dry-run -o json | jq -c '.items[]')

while IFS= read -r crd; do
  name=$(echo "$crd" | jq -r '.spec.names.plural')
  filename="crd/$name.yaml"

  pretty=$(echo "$crd" | yq -P r -)

  echo "# This file has been generated with Velero $version. Do not edit." > $filename
  echo -e "---\n$pretty" >> $filename

  # remove misleading values
  yq delete -i -d'*' $filename 'metadata.creationTimestamp'
done <<< "$crds"
