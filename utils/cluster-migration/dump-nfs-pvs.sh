#!/usr/bin/env bash
set -euo pipefail

dumpdir="$(mktemp -d)"

kubectl get --export -o=json ns | \
jq '.items[] |
	select(.metadata.name!="kube-system") |
	select(.metadata.name!="default") |
	del(.status,
        .metadata.uid,
        .metadata.selfLink,
        .metadata.resourceVersion,
        .metadata.creationTimestamp,
        .metadata.generation
    )' > $dumpdir/namespaces.json

for ns in $(jq -r '.metadata.name' < $dumpdir/namespaces.json);do
    echo "Namespace: $ns"
    kubectl --namespace="${ns}" get --export -o=json pvc | \
    jq '.items[] |
        del(
            .spec.clusterIP,
            .metadata.uid,
            .metadata.selfLink,
            .metadata.resourceVersion,
            .metadata.creationTimestamp,
            .metadata.generation,
            .metadata.annotation,
            .status
        )' >> "$dumpdir/pvc.json"
done

kubectl get --export -o=json pv | \
jq '.items[] |
    del(
        .metadata.uid,
        .metadata.selfLink,
        .metadata.resourceVersion,
        .metadata.creationTimestamp,
        .metadata.generation,
        .metadata.annotation,
        .status,
        .spec.claimRef
    )' > "$dumpdir/pv.json"

printf "backups stored in \n$dumpdir/pv.json\n$dumpdir/pvc.json\n"
