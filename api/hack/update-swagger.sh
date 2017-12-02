#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

API_DIR="${GOPATH}/src/github.com/kubermatic/kubermatic/api"
SWAGGER_FILE="swagger.json"

cd ${API_DIR}/vendor/github.com/go-swagger/go-swagger/cmd/swagger
go install 
cd ${API_DIR}/cmd/kubermatic-api/
swagger generate spec -o ${SWAGGER_FILE}
