#!/usr/bin/env bash

set -euo pipefail

cd $(dirname $0)/..
source hack/lib.sh

echodate "Updating static files in Kubermatic Helm chart..."
go run codegen/kubermatic_operator/main.go
echodate "Done."
