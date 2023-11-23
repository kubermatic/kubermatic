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

if [ "$#" -lt 2 ] || [ "$1" == "--help" ]; then
  cat << EOF
Usage: $(basename $0) (chart|embed) (version) 
where:
    chart:      update the CRDs for the Velero chart deployed on seed clusters.
    embed:      update the CRDs for the Velero integration into user clusters.
    version:    Velero version used to generate the CRDs, e.g. v1.10.1
e.g: $(basename $0) chart v1.10.1
EOF
  exit 0
fi

containerize ./hack/update-velero-crds.sh $1 $2

# positional args refer to the args of containerize() function, not the script. So, we skip one
# echo $*
# ./hack/update-velero-crds.sh chart v1.10.1
if is_containerized; then
  location="$2"
  version="$3"
fi
velero=velero

if ! [ -x "$(command -v $velero)" ]; then
  url="https://github.com/vmware-tanzu/velero/releases/download/$version/velero-$version-linux-amd64.tar.gz"
  velero=/tmp/velero

  echodate "Downloading Velero $version..."
  wget -O- "$url" | tar xzOf - velero-$version-linux-amd64/velero > $velero
  chmod +x $velero
  echodate "Done!"
fi

crd_dir=""
if [ "$location" = "chart" ]; then
  crd_dir="charts/backup/velero/crd"
  mkdir -p "$crd_dir"
else
  crd_dir="pkg/ee/cluster-backup/resources/user-cluster/static"
fi

cd "$crd_dir"

version=$($velero version --client-only | grep Version | cut -d' ' -f2)
crds=$($velero install --crds-only --dry-run -o json | jq -c '.items[]')

while IFS= read -r crd; do
  name=$(echo "$crd" | jq -r '.spec.names.plural')
  filename="$name.yaml"

  pretty=$(echo "$crd" | yq --prettyPrint)

  echo "# This file has been generated with Velero $version. Do not edit." > $filename
  echo -e "---\n$pretty" >> $filename

  # remove misleading values
  yq --inplace 'del(.metadata.creationTimestamp)' $filename
done <<< "$crds"
