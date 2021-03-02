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
### Works around the fact that `go test -coverprofile` currently does not work
### with multiple packages, see https://code.google.com/p/go/issues/detail?id=6909
###
###     Usage: cover.sh [--html]
###
###     --html      Additionally create HTML report and open it in browser

set -e

workdir=.cover
profile="$workdir/cover.out"
mode="set"

generate_cover_data() {
  rm -rf "$workdir"
  mkdir "$workdir"

  for pkg in "$@"; do
    f="$workdir/$(echo "$pkg" | tr / -).cover"
    go test -covermode="$mode" -coverprofile="$f" "$pkg"
  done

  echo "mode: $mode" > "$profile"
  grep -h -v "^mode:" "$workdir"/*.cover >> "$profile"
}

show_cover_report() {
  echo "Generating $1 report"
  go tool cover -"$1"="$profile" "$2"
}

generate_cover_data "$(go list ./cmd/... ./pkg/...)"
show_cover_report func

case "$1" in
"") ;;
--html)
  show_cover_report html "-o $workdir/cover.html"
  ;;
*)
  echo >&2 "error: invalid option: $1"
  exit 1
  ;;
esac
