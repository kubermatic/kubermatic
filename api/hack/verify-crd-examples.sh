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

echodate "Recreating example CRs..."

cp -ar docs $tmpdir
rm -f docs/*.generated.yaml
./api/hack/update-crd-examples.sh
diff -rdu docs $tmpdir

echodate "Example CRs are in-sync with Go structs."
