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

CONTAINERIZE_IMAGE=quay.io/kubermatic/build:go-1.20-node-18-4 containerize ./hack/update-codegen.sh

sed="sed"
[ "$(command -v gsed)" ] && sed="gsed"

echodate "Generating reconciling helpers"

reconcileHelpers=pkg/resources/reconciling/zz_generated_reconcile.go
go run k8c.io/reconciler/cmd/reconciler-gen --config hack/reconciling.yaml > $reconcileHelpers

currentYear=$(date +%Y)
$sed -i "s/Copyright YEAR/Copyright $currentYear/g" $reconcileHelpers
