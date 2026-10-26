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

### Checks resolve_previous_minor_release from hack/ci/lib-mla-upgrade.sh
### against fixed tag lists. Needs no network.

set -euo pipefail

cd $(dirname $0)/../..
source hack/ci/lib-mla-upgrade.sh

tags="v2.30.0
v2.30.7
v2.30.8
v2.31.0-rc.3
v2.31.0
v2.31.2
v2.31.10
v2.32.0-alpha.1"

failures=0

expect() {
  local base_ref="$1"
  local want="$2"
  local got

  got="$(resolve_previous_minor_release "$base_ref" "$tags")"
  if [ "$got" != "$want" ]; then
    echo "FAIL: $base_ref resolved to $got, expected $want"
    failures=$((failures + 1))
  else
    echo "ok: $base_ref -> $got"
  fi
}

expect main v2.31.10
expect release/v2.31 v2.30.8
expect release/v2.32 v2.31.10

got="$(KKP_PREVIOUS_VERSION=v9.9.9 resolve_previous_minor_release main "$tags")"
if [ "$got" != "v9.9.9" ]; then
  echo "FAIL: KKP_PREVIOUS_VERSION override ignored, got $got"
  failures=$((failures + 1))
fi

exit $failures
