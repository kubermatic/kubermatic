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

promtool=promtool
if ! [ -x "$(command -v $promtool)" ]; then
  version=2.43.1
  url="https://github.com/prometheus/prometheus/releases/download/v$version/prometheus-$version.linux-amd64.tar.gz"
  promtool=/tmp/promtool

  echodate "Downloading promtool v$version..."
  wget -O- "$url" | tar xzOf - prometheus-$version.linux-amd64/promtool > $promtool
  chmod +x $promtool
  echodate "Done!"
fi

cd charts/monitoring/prometheus/rules/

# remove old files
rm -f ./*.yaml

cd src/

for file in */*.yaml; do
  newfile=$(dirname "$file")-$(basename "$file")
  echo "$file => $newfile"

  echo -e "# This file has been generated, DO NOT EDIT.\n" > "../$newfile"
  yq 'del(.groups.[].rules.[].runbook)' "$file" >> "../$newfile"
done

cd ..
$promtool check rules ./*.yaml
