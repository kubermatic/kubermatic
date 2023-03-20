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

if [ -z "${NO_UPDATE:-}" ]; then
  VERSION="${1:-main}"

  echodate "Updating Go dependency to $VERSION (set \$NO_UPDATE=1 to skip)…"

  go get -v "k8c.io/api/v2@$VERSION"
  go mod tidy
fi

apiGitVersion="$(kkp_api_dependency)"

echodate "KKP API is now on $apiGitVersion"
echodate "Fetching CRDs…"

tmpFile=k8c-io-api.tar.gz
curl -fLo "$tmpFile" "https://github.com/kubermatic/api/archive/$apiGitVersion.tar.gz"

echodate "Updating CRDs…"
rm pkg/crd/k8c.io/apps.kubermatic.k8c.io_*.yaml
rm pkg/crd/k8c.io/kubermatic.k8c.io_*.yaml

tar xzf "$tmpFile" --wildcards -C pkg/crd/k8c.io/ --strip-components=3 '*/crd/k8c.io/*'
rm "$tmpFile"
