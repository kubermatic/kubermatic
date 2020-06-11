#!/usr/bin/env bash

set -euo pipefail

cd $(dirname $0)/../..
source ./api/hack/lib.sh

tmpdir=docs-old

cleanup() {
  rm -rf "$tmpdir"
}
trap "cleanup" EXIT SIGINT

cleanup

echodate "Recreating documentation..."

cp -ar docs $tmpdir
rm -f docs/zz_generated.*.yaml
./api/hack/update-docs.sh
diff -rdu docs $tmpdir

echodate "Documentation is in-sync with Go code."
