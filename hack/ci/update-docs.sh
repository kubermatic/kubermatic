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

### Updates the docs repository by copying over a couple of generated
### files, like CRD examples and the Prometheus runbook.

set -euo pipefail

cd $(dirname $0)/../..
source hack/lib.sh

TARGET_DIR=docs_sync
REVISION=$(git rev-parse --short HEAD)

# create the addon resource overview (addonresources.json)
go run codegen/addon-resources/main.go

# configure Git
git config --global user.email "dev@kubermatic.com"
git config --global user.name "Prow CI Robot"
git config --global core.sshCommand 'ssh -o CheckHostIP=no -i /ssh/id_rsa'
ensure_github_host_pubkey

# create a fresh clone
git clone git@github.com:kubermatic/docs.git $TARGET_DIR
cd $TARGET_DIR

# copy interesting files over
mkdir -p data/kubermatic/main
mkdir -p content/kubermatic/main/data

for resource in seed kubermaticConfiguration applicationDefinition applicationInstallation; do
  for edition in ce ee; do
    cp ../docs/zz_generated.$resource.$edition.yaml content/kubermatic/main/data/$resource.$edition.yaml
  done

  # for backwards compatibility with the scripting in the docs repository
  cp ../docs/zz_generated.$resource.ce.yaml content/kubermatic/main/data/$resource.yaml
done

cp ../docs/zz_generated.addondata.go.txt content/kubermatic/main/data/addondata.go
cp ../docs/zz_generated.prometheusdata.go.txt content/kubermatic/main/data/prometheusdata.go
cp ../addonresources.json content/kubermatic/main/data/addonresources.json

# re-create Prometheus runbook
make runbook

# update CRDs reference
hack/render-crds.sh

# update components page
components_file=content/kubermatic/main/architecture/compatibility/KKP-components-versioning/_index.en.md
echo > ${components_file} <<EOT
+++
title = "KKP Components"
date = 2021-04-13T20:07:15+02:00
weight = 2

+++

## Kubermatic Kubernetes Platform Components

The following list is only eligible for the version that is currently available. Kubermatic has a strong emphasis on the security and reliability of the provided software and releases updates regularly that also update components.

| KKP Components                | Version                      |
| ----------------------------- | ---------------------------- |
EOT

for filepath in $(find charts -name Chart.yaml | sort); do echo "| $(echo ${filepath} | sed -e 's/^charts\///g' -e 's/\/Chart\.yaml$//g') | $(yq '.appVersion' ${filepath} | sed -e 's/^v//g') |" >> ${components_file}; done

# update repo
git add .

if ! git diff --cached --stat --exit-code; then
  git commit -m "Syncing with kubermatic/kubermatic@$REVISION"
  git push
fi
