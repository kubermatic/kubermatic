#!/usr/bin/env sh

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

set -e

cd $(dirname $0)/..
. hack/lib.sh

CONTAINERIZE_IMAGE=quay.io/kubermatic/codespell:1.17.1 containerize ./hack/verify-spelling.sh

echodate "Running codespell..."

codespell \
  --skip .git,_build,_dist,vendor,go.mod,go.sum,swagger.json,*.jpg,*.jpeg,*.png,*.woff,*.woff2,*.pem,./charts/cert-manager/crd,./charts/backup/velero/crd \
  --ignore-words .codespell.exclude \
  --check-filenames \
  --check-hidden

echodate "No problems detected."
