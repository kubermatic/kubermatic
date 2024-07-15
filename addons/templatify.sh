#!/usr/bin/env bash

# Copyright 2024 The Kubermatic Kubernetes Platform contributors.
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

(
  cd $(dirname $0)/../codegen/addon-templatifier
  go build -v
)

filename="$1"
shift

tmpFile="$filename.tmp"

# pipe input file through the templatifier to templatify it
cat $filename | $(dirname $0)/../codegen/addon-templatifier/addon-templatifier "$@" > "$tmpFile"
mv "$tmpFile" "$filename"

# pipe it through yq to undo yaml.v3's linebreaks, which could confuse Go templates
yq --inplace '.' "$filename"
