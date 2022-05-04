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

### Generate test coverage statistics for Go packages.
###
###     Usage: coverage.sh [--html]
###
###     --html      Additionally create HTML report and open it in browser

set -e

workdir=.cover
profile="$workdir/cover.out"
covermode="set"

create_cover_report() {
  local kind="$1"
  shift

  echo "Generating $kind report…"
  go tool cover "-$kind=$profile" $@
}

rm -rf "$workdir"
mkdir "$workdir"

echo "Generating coverage data…"
go test -covermode="$covermode" -coverprofile="$profile" ./cmd/... ./pkg/...

create_cover_report func

case "$1" in
"") ;;
--html)
  create_cover_report html -o $workdir/cover.html
  ;;
*)
  echo >&2 "error: invalid option: $1"
  exit 1
  ;;
esac
