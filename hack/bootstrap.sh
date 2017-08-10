#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

mkdir -p $GOPATH/src/github.com/kubermatic
cd $GOPATH/src/github.com/kubermatic

git clone git@github.com:kubermatic/api
git clone git@github.com:kubermatic/config
git clone git@github.com:kubermatic/secrets

cd api

# Link cloud-init
mkdir -p template/coreos &&
pushd template/coreos &&
ln -s $GOPATH/src/github.com/kubermatic/config/kubermatic/static/nodes/coreos/cloud-init.yaml cloud-init.yaml &&
popd

# Install linters
go get -u github.com/golang/lint/golint
go get -u github.com/client9/misspell/cmd/misspell
go get -u github.com/kisielk/errcheck
go get -u github.com/Masterminds/glide
go get -u github.com/opennota/check/cmd/varcheck
go get -u github.com/opennota/check/cmd/structcheck
go get -u honnef.co/go/tools/cmd/unused
go get -u honnef.co/go/tools/cmd/gosimple

# Install dependencies
make bootstrap
