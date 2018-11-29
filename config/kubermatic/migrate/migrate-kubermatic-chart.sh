#!/usr/bin/env bash

# Modify according to the tiller namespace which contains the release configmaps.
# Its what gets specified with the "--tiller-namespace" flag on Helm. Default is "kube-system"
TILLER_NAMESPACE='kube-system'

# Delete the kubermatic namespace to enable a clean install afterwards
kubectl delete ns kubermatic

# Delete a ClusterRoleBinding - which is the only thing which does not exist in the kubermatic namespace
kubectl delete clusterrolebindings.rbac.authorization.k8s.io kubermatic

for cm in $(kubectl -n kubermatic-installer get configmap -o json | jq -r '.items[].metadata.name' | grep 'kubermatic');do
    echo "Deleting helm release info in ConfigMap: ${TILLER_NAMESPACE}/${cm}"
    kubectl -n ${TILLER_NAMESPACE} delete configmap ${cm}
done

helm upgrade --install --kube-context=europe-west3-c --tiller-namespace=kubermatic-installer \
    --set=kubermatic.controller.image.tag=aee6917710fde8e8b8ca65c73d904f950500af55 \
    --set=kubermatic.api.image.tag=aee6917710fde8e8b8ca65c73d904f950500af55 \
    --set=kubermatic.rbac.image.tag=aee6917710fde8e8b8ca65c73d904f950500af55 \
    --values /home/henrik/go/src/github.com/kubermatic/secrets/seed-clusters/dev.kubermatic.io/values.yaml \
    --namespace kubermatic kubermatic ../kubermatic/
