#!/usr/bin/env bash

# Copyright 2026 The Kubermatic Kubernetes Platform contributors.
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

# Stages the release-archive layout the installer expects: an examples/
# directory beside the installer binary, plus the freshly built installer
# itself. Removes the manual mkdir/cp step from every run.
#
# Usage (from the repository root):
#   hack/dev/stage.sh TARGET_DIR
#
# Example:
#   hack/dev/stage.sh /tmp/mytest
#   /tmp/mytest/kubermatic-installer local kind --name my-kkp ...

set -euo pipefail

DIR="${1:?usage: stage.sh TARGET_DIR}"

mkdir -p "${DIR}/examples"
cp charts/kubermatic.example.ce.yaml "${DIR}/examples/kubermatic.example.yaml"
cp charts/values.example.ce.yaml "${DIR}/examples/values.example.yaml"

go build -o "${DIR}/kubermatic-installer" ./cmd/kubermatic-installer

echo "> staged ${DIR}/kubermatic-installer + examples/"
echo "> run it from the repository root so ./charts resolves"
