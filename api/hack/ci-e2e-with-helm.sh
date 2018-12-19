#!/usr/bin/env bash

set -euo pipefail

export BUILD_ID=${BUILD_ID:-build-id-undef}
export NAMESPACE="prow-kubermatic-${BUILD_ID}"

function cleanup {
  set +e
  helm delete --purge kubermatic-$BUILD_ID  \
    --tiller-namespace=$NAMESPACE
  kubectl delete clusterrolebinding -l prowjob=$BUILD_ID
  kubectl delete namespace $NAMESPACE
}
trap cleanup EXIT

#echo "Getting secrets from Vault"
#export VAULT_ADDR=https://vault.loodse.com/
#export VAULT_TOKEN=$(vault write \
#  --format=json auth/approle/login \
#  role_id=$VAULT_ROLE_ID secret_id=$VAULT_SECRET_ID \
#  | jq .auth.client_token -r)
vault kv get -field=kubeconfig \
  dev/seed-clusters/dev.kubermatic.io > /tmp/kubeconfig
vault kv get -field=values.yaml \
  dev/seed-clusters/dev.kubermatic.io > /tmp/values.yaml
export KUBECONFIG=/tmp/kubeconfig
export VALUES_FILE=/tmp/values.yaml

echo "Building docker image"
./api/hack/push_image.sh ${PULL_PULL_SHA}

INITIAL_MANIFESTS=$(cat <<EOF
apiVersion: v1
kind: Namespace
metadata:
  name: $NAMESPACE
spec: {}
status: {}
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: tiller
  namespace: $NAMESPACE
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: tiller
  namespace: $NAMESPACE
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: prow-${BUILD_ID}-tiller
  labels:
    prowjob: ${BUILD_ID}
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
subjects:
  - kind: ServiceAccount
    name: tiller
    namespace: $NAMESPACE
---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: prow-${BUILD_ID}-kubermatic
  labels:
    prowjob: ${BUILD_ID}
subjects:
- kind: ServiceAccount
  name: kubermatic
  namespace: $NAMESPACE
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
EOF
)
echo "Creating namespace $NAMESPACE to deploy kubermatic in"
echo "$INITIAL_MANIFESTS"|kubectl apply -f -

echo "Deploying tiller"
helm init --wait --service-account=tiller --tiller-namespace=$NAMESPACE

echo "Installing Kubermatic via Helm"
rm -f config/kubermatic/templates/cluster-role-binding.yaml
helm upgrade --install --wait --timeout 300 \
  --tiller-namespace=$NAMESPACE \
  --set=kubermatic.isMaster=true \
  --set=kubermatic.controller.image.tag=$PULL_PULL_SHA \
  --set=kubermatic.api.image.tag=$PULL_PULL_SHA \
  --set=kubermatic.rbac.image.tag=$PULL_PULL_SHA \
  --set=kubermatic.worker_name=prow-$BUILD_ID \
  --set=kubermatic.project_migrator.dry_run=true \
  --values ${VALUES_FILE} \
  --namespace $NAMESPACE \
  kubermatic-$BUILD_ID ./config/kubermatic/
echo "Finished installing Kubermatic"
