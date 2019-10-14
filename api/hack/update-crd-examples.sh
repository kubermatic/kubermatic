#!/usr/bin/env bash

set -euo pipefail

cd $(dirname $0)/..

go run codegen/seed-yaml/main.go > ../docs/seed-cr.generated.yaml
