#!/bin/bash -e

set -o errexit
set -o nounset
set -o pipefail

source vendor/k8s.io/code-generator/hack/lib/codegen.sh

echo Removing old clients
rm -rf "pkg/crd/client"

codegen::generate-groups all github.com/kubermatic/kubermatic/api/pkg/crd/client/seed github.com/kubermatic/kubermatic/api/pkg/crd "etcdoperator:v1beta2"
codegen::generate-groups all github.com/kubermatic/kubermatic/api/pkg/crd/client/master github.com/kubermatic/kubermatic/api/pkg/crd "kubermatic:v1"
