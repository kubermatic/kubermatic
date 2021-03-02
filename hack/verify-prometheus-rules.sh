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

if ! [ -x "$(command -v promtool)" ]; then
  version=2.19.2

  echodate "Downloading promtool v$version..."

  curl -L https://github.com/prometheus/prometheus/releases/download/v$version/prometheus-$version.linux-amd64.tar.gz | tar -xz
  mv prometheus-$version.linux-amd64/promtool /usr/local/bin/
  rm -rf prometheus-$version.linux-amd64

  echodate "Done!"
fi

echodate "Rebuilding rules..."
./hack/update-prometheus-rules.sh

echodate "Diffing..."
git diff --exit-code
echodate "No changes detected."
