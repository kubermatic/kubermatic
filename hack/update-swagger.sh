#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

KUBERMATIC_DIR="$(go env GOPATH)/src/github.com/kubermatic/kubermatic"
SWAGGER_FILE="swagger.json"

cd ${KUBERMATIC_DIR}/vendor/github.com/go-swagger/go-swagger/cmd/swagger
go install

cd ${KUBERMATIC_DIR}/cmd/kubermatic-api/
swagger generate spec --scan-models -o ${SWAGGER_FILE}
