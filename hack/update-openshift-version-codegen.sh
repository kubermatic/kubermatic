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

# This file can be used to update the generated image names for Openshift.
# The desired versions msut be configured first in
# codegen/openshift_versions/main.go and a const for each version must be
# added to pkg/controller/openshift/resources/const.go
#
# Also, executing this script requires access to the ocp quay repo.

set -euo pipefail

cd $(dirname $0)/..
source hack/lib.sh

go generate pkg/controller/openshift/resources/const.go
