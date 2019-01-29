#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail
set -x

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
