#!/usr/bin/env bash

crds=templates/customresourcedefinitions.yaml
version=$(velero version --client-only | grep Version | cut -d' ' -f2)

echo "# This file has been generated with Velero $version. Do not edit." > $crds

# extract CRDs by calling Velero with dummy values
velero install --bucket does-not-matter --use-restic --provider gcp --secret-file Chart.yaml --dry-run -o json | \
  jq '.items=(.items | map(select(.kind=="CustomResourceDefinition")) | sort_by(.metadata.name))' | \
  jq 'del(.items[].metadata.creationTimestamp)' | \
  jq -S '.items[].metadata.annotations["helm.sh/hook"] = "crd-install"' | \
  yq r - \
  >> $crds
