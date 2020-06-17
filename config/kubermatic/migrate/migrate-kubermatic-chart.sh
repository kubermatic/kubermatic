#!/usr/bin/env bash

# Copyright 2020 The Kubermatic Kubernetes Platform contributors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -euo pipefail

if [ "$#" -lt 2 ] || [ "${1}" == "--help" ]; then
  cat <<EOF
Usage: $(basename $0) path/to/values.yaml kubermatic-namespace
EOF
  exit 0
fi

if [[ ! -f ${1} ]]; then
    echo "File not found!"
    exit 1
fi

VALUES_FILE=$(realpath ${1})
KUBERMATIC_NAMESPACE=${2:-'kubermatic'}

TILLER_NAMESPACE=""
RELEASE_NAME=""
# Figure out the namespace tiller is saving the release configmaps in.
# This is needed to figure out the correct release name + content
# This script will reuse the old release name
for NS in $(kubectl get ns -o json | jq -r '.items[].metadata.name');do
  CONFIGMAPS=$(kubectl -n ${NS} get ConfigMap -l "OWNER=TILLER" -o json | jq -r '.items[].metadata.name')
  # Check all revisions
  for CM in ${CONFIGMAPS};do
    # Get chart name
    CURRENT_RELEASE_NAME=$(kubectl -n ${NS} get ConfigMap ${CM} -o json | jq -r '.metadata.labels.NAME')
    REVISION=$(kubectl -n ${NS} get ConfigMap ${CM} -o json | jq -r '.metadata.labels.VERSION')
    RELEASE_CONTENT=$(helm --tiller-namespace=${NS} get ${CURRENT_RELEASE_NAME} --revision ${REVISION})
    if [[ ${RELEASE_CONTENT} == *"kubermatic-0.1.0"* ]]; then
      TILLER_NAMESPACE=${NS}
      RELEASE_NAME=${CURRENT_RELEASE_NAME}
      break
    fi
  done

  # End outer loop if we found it
  if [[ -n "${TILLER_NAMESPACE}" ]] && [[ -n "${RELEASE_NAME}" ]];
  then
    break
  fi
done

# End outer loop if we found it
if [[ -z "${TILLER_NAMESPACE}" ]] || [[ -z "${RELEASE_NAME}" ]];
then
  echo "could not get tiller namespace + release name of the kubermatic chart"
  exit 1
fi

echo "========================================"
echo "================ WARNING ==============="
echo "========================================"
echo "This script will reinstall the existing kubermatic installation"
echo ""
echo "Waiting 30 seconds..."
sleep 30

# Delete the kubermatic namespace to enable a clean install afterwards
kubectl delete --ignore-not-found=true ns ${KUBERMATIC_NAMESPACE}

# Delete a ClusterRoleBinding - which is the only thing which does not exist in the kubermatic namespace
kubectl delete --ignore-not-found=true clusterrolebindings.rbac.authorization.k8s.io kubermatic
kubectl -n ${TILLER_NAMESPACE} delete configmap -l NAME=${RELEASE_NAME}

cd "$(dirname "$0")/../../"
helm upgrade --install --tiller-namespace=${TILLER_NAMESPACE} \
    --values ${VALUES_FILE} \
    --namespace ${KUBERMATIC_NAMESPACE} ${RELEASE_NAME} \
    --set kubermatic.checks.crd.disable=true \
    ./kubermatic/

echo "Successfully reinstalled kubermatic. From now on, the Kubermatic chart does not contain any CustomResourceDefinitions."
echo "This enables users to purge the chart without deleting all clusters as consequence."
echo "From now on, new CRD's will be placed in the /crd folder inside the kubermatic chart."
