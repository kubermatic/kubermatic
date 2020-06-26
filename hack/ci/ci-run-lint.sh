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

lint_output="$(mktemp)"

# run with concurrency=1 to lower memory usage a bit
golangci-lint run -v --build-tags "$KUBERMATIC_EDITION" --concurrency 1 --print-resources-usage 2>&1|tee $lint_output

if egrep -q 'compilation errors|Packages that do not compile' $lint_output; then
  echo "compilation error during linting"
  exit 1
fi
