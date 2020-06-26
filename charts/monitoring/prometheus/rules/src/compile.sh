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

cd $(dirname $0)

# Install yq if not installed
if ! [ -x "$(command -v yq)" ]; then
	echo "yq not installed / vailable in PATH!"
	echo "Executing go get on github.com/mikefarah/yq ..."
	go get github.com/mikefarah/yq
	echo "Done!"
fi

comment="# This file has been generated, do not edit."

for file in */*.yaml; do
  newfile=$(dirname $file)-$(basename $file)
  echo "$file => $newfile"
  yq r $file --tojson | jq 'del(.groups[].rules[].runbook)' | (echo "$comment"; yq r --prettyPrint -) > ../$newfile
done
