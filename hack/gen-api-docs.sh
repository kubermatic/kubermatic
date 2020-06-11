#!/bin/bash

cd $(go env GOPATH)/src/github.com/kubermatic/kubermatic/api
swagger -apiPackage="github.com/kubermatic/kubermatic/api"  \
  -mainApiFile="github.com/kubermatic/kubermatic/cmd/kubermatic-api/main.go" \
  -format="swagger" \
  -output="$(go env GOPATH)/src/github.com/kubermatic/kubermatic/api/handler/swagger/"
