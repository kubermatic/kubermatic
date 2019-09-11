#!/usr/bin/env sh

set -euo pipefail

# This script can be used to get all crd manifests from a running
# openshift cluster


targetFile="$(mktemp)"

for crd in $(kubectl get crd -o name); do
	echo "Getting crd $crd"
	echo -e '\n---\n' >> "$targetFile"
	# We cant use --export because the status has mandatory fields that are not preserved
	kubectl get "$crd" -o json|jq '{metadata: {name: .metadata.name}, apiVersion: .apiVersion, kind: .kind, spec: .spec}' >> "$targetFile"
done

mv "$targetFile" "$(dirname "$0")/crd/crds.yaml"
