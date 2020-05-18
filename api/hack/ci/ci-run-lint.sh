#!/usr/bin/env bash

set -euo pipefail

lint_output="$(mktemp)"

# run with concurrency=1 to lower memory usage a bit
golangci-lint run -v --build-tags "$KUBERMATIC_EDITION" --concurrency 1 --print-resources-usage 2>&1|tee $lint_output

if egrep -q 'compilation errors|Packages that do not compile' $lint_output; then
  echo "compilation error during linting"
  exit 1
fi
