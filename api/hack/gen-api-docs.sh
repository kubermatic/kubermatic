#!/bin/bash

cd ${GOPATH}/src/github.com/kubermatic/kubermatic/api
swagger -apiPackage="github.com/kubermatic/kubermatic/api"  \
  -mainApiFile="github.com/kubermatic/kubermatic/api/cmd/kubermatic-api/main.go" \
  -format="swagger" \
  -output="${GOPATH}/src/github.com/kubermatic/kubermatic/api/handler/swagger/"