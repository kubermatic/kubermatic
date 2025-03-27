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

line() {
  printf "| %-30s | %-30s |\n" "$1" "$2"
}

TARGET_DIR=docs_sync
REVISION=$(git rev-parse --short HEAD)

# figure out what release we're updating the docs for (main branch or a fixed release branch?)
GIT_BRANCH="${PULL_BASE_REF:-main}"

export KKP_RELEASE="$GIT_BRANCH"
if [[ "$GIT_BRANCH" =~ release/v[0-9]+.* ]]; then
  # turn "release/v2.21" into "v2.21"
  KKP_RELEASE="${GIT_BRANCH#release/}"
fi

echodate "Updating documentation for KKP."
echodate "GIt branch: $GIT_BRANCH"
echodate "KKP version directory: $KKP_RELEASE"
echo

# configure Git
git config --global user.email "dev@kubermatic.com"
git config --global user.name "Kubermatic Bot"
git config --global core.sshCommand 'ssh -o CheckHostIP=no -i /ssh/id_rsa'
ensure_github_host_pubkey

# create the addon resource overview (addonresources.json)
go run codegen/addon-resources/main.go

# create a fresh clone
git clone git@github.com:kubermatic/docs.git $TARGET_DIR
cd $TARGET_DIR

# copy interesting files over
mkdir -p "data/kubermatic/$KKP_RELEASE"
mkdir -p "content/kubermatic/$KKP_RELEASE/data"

for resource in seed kubermaticConfiguration applicationDefinition applicationInstallation; do
  for edition in ce ee; do
    cp ../docs/zz_generated.$resource.$edition.yaml "content/kubermatic/$KKP_RELEASE/data/$resource.$edition.yaml"
  done

  # for backwards compatibility with the scripting in the docs repository
  cp ../docs/zz_generated.$resource.ce.yaml "content/kubermatic/$KKP_RELEASE/data/$resource.yaml"
done

cp ../docs/zz_generated.addondata.go.txt "content/kubermatic/$KKP_RELEASE/data/addondata.go"
cp ../docs/zz_generated.applicationdata.go.txt "content/kubermatic/$KKP_RELEASE/data/applicationdata.go"
cp ../docs/zz_generated.prometheusdata.go.txt "content/kubermatic/$KKP_RELEASE/data/prometheusdata.go"
cp ../addonresources.json "content/kubermatic/$KKP_RELEASE/data/addonresources.json"

# re-create Prometheus runbook
make runbook

# update CRDs reference
hack/render-crds.sh

# update components page
components_file=content/kubermatic/$KKP_RELEASE/architecture/compatibility/kkp-components-versioning/_index.en.md
cat > ${components_file} << EOT
+++
title = "KKP Components"
date = 2021-04-13T20:07:15+02:00
weight = 2

+++

## Kubermatic Kubernetes Platform Components

The following list is only applicable for the KKP version that is currently available. Kubermatic has a strong emphasis on security and reliability
of provided software and therefore releases updates regularly that also include component updates.

| KKP Components                 | Version                        |
| ------------------------------ | ------------------------------ |
EOT

# iterate over all charts to extract version information
for filepath in $(find ../charts -name Chart.yaml | sort); do
  # extract chart name by removing a "../charts/" prefix from the directory path
  chart_name=$(dirname ${filepath} | sed -e 's/^\.\.\/charts\///g')
  # read appVersion from Chart.yaml and normalize version format by removing a "v" prefix
  app_version=$(yq '.appVersion' ${filepath} | sed -e 's/^v//g')
  # append information to components markdown file
  line ${chart_name} ${app_version} >> ${components_file}
done

# update repo
git add .

if ! git diff --cached --stat --exit-code; then
  git commit -m "Syncing with KKP $KKP_RELEASE ($REVISION)"
  git push
fi
