#!/usr/bin/env bash

set -euo pipefail

source $(dirname $0)/../lib.sh

cd $(dirname $0)/../../..

TARGET_DIR=docs_sync
REVISION=$(git rev-parse --short HEAD)

# configure Git
git config --global user.email "dev@loodse.com"
git config --global user.name "Prow CI Robot"
git config --global core.sshCommand 'ssh -o CheckHostIP=no -i /ssh/id_rsa'
ensure_github_host_pubkey

# create a fresh clone
git clone git@github.com:kubermatic/docs.git $TARGET_DIR
cd $TARGET_DIR

# copy interesting files over
cp ../docs/zz_generated.seed.yaml data/seed.yaml
cp ../docs/zz_generated.kubermaticConfiguration.yaml data/kubermaticConfiguration.yaml

# re-create Prometheus runbook
make runbook

# update repo
git add .

if ! git diff --cached --stat --exit-code; then
	git commit -m "Syncing with kubermatic/kubermatic@$REVISION"
	git push
fi
