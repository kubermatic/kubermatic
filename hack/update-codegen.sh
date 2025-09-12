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

cd $(dirname $0)/..
source hack/lib.sh

CONTAINERIZE_IMAGE=quay.io/kubermatic/build:go-1.25-node-22-4 containerize ./hack/update-codegen.sh

sed="sed"
[ "$(command -v gsed)" ] && sed="gsed"

echodate "Generating reconciling helpers"

reconcileHelpers=pkg/resources/reconciling/zz_generated_reconcile.go
go run k8c.io/reconciler/cmd/reconciler-gen --config hack/reconciling.yaml > $reconcileHelpers

currentYear=$(date +%Y)
$sed -i "s/Copyright YEAR/Copyright $currentYear/g" $reconcileHelpers

CRD_DIR=pkg/crd/k8c.io

echodate "Generating openAPI v3 CRDs"

export GODEBUG=gotypesalias=0
# see https://github.com/kubernetes-sigs/controller-tools/issues/1123
# and https://github.com/kubernetes-sigs/controller-tools/pull/1122
#
# To fix this we will need to update controller-tools at least to v0.17.1+ which pull main kubernetes updates
go run sigs.k8s.io/controller-tools/cmd/controller-gen \
  crd \
  object:headerFile=./hack/boilerplate/ce/boilerplate.go.txt \
  paths=./sdk/apis/... \
  output:crd:dir=./$CRD_DIR

annotation="kubermatic.k8c.io/location"
declare -A locationMap=(
  ["applicationdefinitions.apps.kubermatic.k8c.io"]="master,seed"
  ["applicationinstallations.apps.kubermatic.k8c.io"]="usercluster"
  ["addonconfigs.kubermatic.k8c.io"]="master"
  ["addons.kubermatic.k8c.io"]="master,seed"
  ["admissionplugins.kubermatic.k8c.io"]="master"
  ["alertmanagers.kubermatic.k8c.io"]="master,seed"
  ["allowedregistries.kubermatic.k8c.io"]="master"
  ["clusters.kubermatic.k8c.io"]="master,seed"
  ["clustertemplateinstances.kubermatic.k8c.io"]="master,seed"
  ["clustertemplates.kubermatic.k8c.io"]="master,seed"
  ["constraints.kubermatic.k8c.io"]="master,seed"
  ["constrainttemplates.kubermatic.k8c.io"]="master,seed"
  ["customoperatingsystemprofiles.operatingsystemmanager.k8c.io"]="master,seed"
  ["etcdbackupconfigs.kubermatic.k8c.io"]="master,seed"
  ["etcdrestores.kubermatic.k8c.io"]="master,seed"
  ["externalclusters.kubermatic.k8c.io"]="master"
  ["groupprojectbindings.kubermatic.k8c.io"]="master,seed"
  ["ipamallocations.kubermatic.k8c.io"]="master,seed"
  ["ipampools.kubermatic.k8c.io"]="master,seed"
  ["kubermaticconfigurations.kubermatic.k8c.io"]="master,seed"
  ["kubermaticsettings.kubermatic.k8c.io"]="master"
  ["mlaadminsettings.kubermatic.k8c.io"]="master,seed"
  ["presets.kubermatic.k8c.io"]="master,seed"
  ["projects.kubermatic.k8c.io"]="master,seed"
  ["resourcequotas.kubermatic.k8c.io"]="master,seed"
  ["rulegroups.kubermatic.k8c.io"]="master,seed"
  ["seeds.kubermatic.k8c.io"]="master,seed"
  ["userprojectbindings.kubermatic.k8c.io"]="master,seed"
  ["usersshkeys.kubermatic.k8c.io"]="master,seed"
  ["users.kubermatic.k8c.io"]="master,seed"
  ["clusterbackupstoragelocations.kubermatic.k8c.io"]="master,seed"

  ["verticalpodautoscalers.autoscaling.k8s.io"]="seed"
  ["verticalpodautoscalercheckpoints.autoscaling.k8s.io"]="seed"

  # Gateway API
  ["gateways.gateway.networking.k8s.io"]="usercluster"
  ["grpcroutes.gateway.networking.k8s.io"]="usercluster"
  ["httproutes.gateway.networking.k8s.io"]="usercluster"
  ["tcproutes.gateway.networking.k8s.io"]="usercluster"
  ["udproutes.gateway.networking.k8s.io"]="usercluster"
  ["tlsroutes.gateway.networking.k8s.io"]="usercluster"
  ["referencegrants.gateway.networking.k8s.io"]="usercluster"

  ["policytemplates.kubermatic.k8c.io"]="master,seed"
  # PolicyBindings will be deployed on master clusters although they are used on seed cluster namespaces.
  # This is because the KKP API (running on master) sets up caching rules for PolicyBindings.
  ["policybindings.kubermatic.k8c.io"]="master,seed"
)

failure=false
echodate "Annotating CRDs"

cleanup_dir() {
  for filename in $1/*.yaml; do
    crdName="$(yq '.metadata.name' "$filename")"
    location="${locationMap[$crdName]}"

    if [ "$location" == "null" ]; then
      echodate "Error: No location defined for CRD $crdName"
      failure=true
      continue
    fi

    yq --inplace ".metadata.annotations.\"$annotation\" = \"$location\"" "$filename"
  done
}

cleanup_dir "$CRD_DIR"
cleanup_dir pkg/crd/k8s.io

if $failure; then
  exit 1
fi
