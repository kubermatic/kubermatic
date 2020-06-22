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

set -exuo pipefail

cd $(dirname $0)

function cleanup {
  set +e
  MAIN_PID=$(jobs -l|grep run-api.sh|awk '{print $2}')
  # There is no `kill job and all its children` :(
  kill $(pgrep -P $MAIN_PID)
  kill $MAIN_PID
}
trap cleanup EXIT

echo "starting api"
./run-api.sh &
echo "finished starting api"

echo "Starting dashboard"
$(go env GOPATH)/src/github.com/kubermatic/dashboard-v2/hack/run-local-dashboard.sh
