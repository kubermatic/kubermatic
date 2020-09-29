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

### Runs golangci-lint to validate that no issues have been introduced
### into the codebase.

set -euo pipefail

cd $(dirname $0)/../..

lint_output="$(mktemp)"
golangci-lint run -v --build-tags "$KUBERMATIC_EDITION" --print-resources-usage ./pkg/... ./cmd/... ./codegen/... 2>&1 | tee $lint_output

if egrep -q 'compilation errors|Packages that do not compile' $lint_output; then
  echo "compilation error during linting"
  exit 1
fi
