#!/bin/bash -e

set -o errexit
set -o nounset
set -o pipefail

source vendor/k8s.io/code-generator/hack/lib/codegen.sh

echo Removing old client
rm -rf "pkg/crd/client"

codegen::generate-groups all github.com/kubermatic/kubermatic/api/pkg/crd/client github.com/kubermatic/kubermatic/api/pkg/crd "kubermatic:v1 etcdoperator:v1beta2"
