#!/usr/bin/env bash

crdfile=templates/customresourcedefinitions.yaml
version=$(velero version --client-only | grep Version | cut -d' ' -f2)

echo "# This file has been generated with Velero $version. Do not edit." > $crdfile

# extract CRDs by calling Velero with dummy values
crds=$(
  velero install --bucket does-not-matter --use-restic --provider gcp --secret-file Chart.yaml --dry-run -o json | \
  jq -c '.items | map(select(.kind=="CustomResourceDefinition")) | sort_by(.metadata.name) | .[]'
)

while IFS= read -r crd; do
  crd=$(
    echo "$crd" | \
    jq 'del(.metadata.creationTimestamp)' | \
    jq -S '.metadata.annotations["helm.sh/hook"] = "crd-install"' | \
    yq r -
  )

  echo -e "---\n$crd" >> $crdfile
done <<< "$crds"
