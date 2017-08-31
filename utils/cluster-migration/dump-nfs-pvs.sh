#!/usr/bin/env bash
mkdir -p ./dump

for ns in $(jq -r '.metadata.name' < ./dump/namespaces.json);do
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
        )' >> "./dump/pvc.json"
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
    )' > "./dump/pv.json"
