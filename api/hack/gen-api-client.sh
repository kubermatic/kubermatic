#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

API_DIR="$(go env GOPATH)/src/github.com/kubermatic/kubermatic/api"
SWAGGER_FILE="swagger.json"

cd ${API_DIR}/vendor/github.com/go-swagger/go-swagger/cmd/swagger
go install
cd ${API_DIR}/cmd/kubermatic-api/
mkdir -p ../../pkg/test/e2e/api/utils/apiclient/
swagger generate client -q -f ${SWAGGER_FILE} -t ../../pkg/test/e2e/api/utils/apiclient/
