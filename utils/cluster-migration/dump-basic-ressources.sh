#!/usr/bin/env bash
mkdir -p ./dump

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
    )' > ./dump/namespaces.json

for ns in $(jq -r '.metadata.name' < ./dump/namespaces.json);do
    echo "Namespace: $ns"
    kubectl --namespace="${ns}" get --export -o=json deployment,ingress,daemonset,secrets,configmap,service,serviceaccount,statefulsets | \
    jq '.items[] |
        select(.type!="kubernetes.io/service-account-token") |
        del(
            .spec.clusterIP,
            .metadata.uid,
            .metadata.selfLink,
            .metadata.resourceVersion,
            .metadata.creationTimestamp,
            .metadata.generation,
            .status,
            .spec.template.spec.securityContext,
            .spec.template.spec.dnsPolicy,
            .spec.template.spec.terminationGracePeriodSeconds,
            .spec.template.spec.restartPolicy,
            .spec.volumeName
        )' >> "./dump/basic-resources.json"
done
