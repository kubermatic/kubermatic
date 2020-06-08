#!/usr/bin/env bash

set -euo pipefail

cd $(dirname $0)/../..
source api/hack/lib.sh

cd config/backup/velero/

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
