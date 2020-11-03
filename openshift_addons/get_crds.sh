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

# This script can be used to get all crd manifests from a running
# openshift cluster

set -euo pipefail

cd $(dirname $0)

targetFile="$(mktemp)"

echo "# This file has been generated, DO NOT EDIT." > $targetFile

for crd in $(kubectl get crd -o name); do
	echo "Getting crd $crd"
	echo -e '\n---\n' >> $targetFile
	# We can't use --export because the status has mandatory fields that are not preserved
	kubectl get $crd -o json|jq '{metadata: {name: .metadata.name}, apiVersion: .apiVersion, kind: .kind, spec: .spec}' >> $targetFile
done

mv $targetFile openshift_addons/crd/crds.yaml
