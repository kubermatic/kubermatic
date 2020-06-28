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

cd charts/monitoring/prometheus/rules/

# remove old files
rm -f *.yaml

cd src/

for file in */*.yaml; do
  newfile=$(dirname $file)-$(basename $file)
  echo "$file => $newfile"

  echo -e "# This file has been generated, DO NOT EDIT.\n" > ../$newfile
  yq d $file 'groups.*.rules.*.runbook' >> ../$newfile
done

cd ..
promtool check rules *.yaml
