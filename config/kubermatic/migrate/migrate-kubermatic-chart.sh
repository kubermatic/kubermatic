#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -lt 1 ] || [ "${1}" == "--help" ]; then
  cat <<EOF
Usage: $(basename $0) path/to/values.yaml tiller-namespace
EOF
  exit 0
fi

if [[ ! -f ${1} ]]; then
    echo "File not found!"
    exit 1
fi

VALUES_FILE=$(realpath ${1})
TILLER_NAMESPACE=${2:-'kube-system'}
cd "$(dirname "$0")/../../"


# Delete the kubermatic namespace to enable a clean install afterwards
kubectl delete --ignore-not-found=true ns kubermatic

# Delete a ClusterRoleBinding - which is the only thing which does not exist in the kubermatic namespace
kubectl delete --ignore-not-found=true clusterrolebindings.rbac.authorization.k8s.io kubermatic

for cm in $(kubectl -n kubermatic-installer get configmap -o json | jq -r '.items[].metadata.name' | grep 'kubermatic');do
    echo "Deleting helm release info in ConfigMap: ${TILLER_NAMESPACE}/${cm}"
    kubectl -n ${TILLER_NAMESPACE} delete configmap ${cm}
done

helm upgrade --install --tiller-namespace=${TILLER_NAMESPACE} \
    --values ${VALUES_FILE} \
    --namespace kubermatic kubermatic ./kubermatic/ --dry-run --debug
