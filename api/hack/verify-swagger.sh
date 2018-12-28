#!/bin/bash

set -euxo pipefail

API_DIR="$(go env GOPATH)/src/github.com/kubermatic/kubermatic/api"
SWAGGER_FILE="swagger.json"
TMP_SWAGGER="${SWAGGER_FILE}.tmp"
trap "rm -f $TMP_SWAGGER" EXIT SIGINT SIGTERM

BINPATH=$(go env GOPATH)/bin/swagger
go build -o ${BINPATH} github.com/kubermatic/kubermatic/api/vendor/github.com/go-swagger/go-swagger/cmd/swagger

cd ${API_DIR}/cmd/kubermatic-api/
${BINPATH} generate spec --scan-models -o ${TMP_SWAGGER}
diff -Naup ${SWAGGER_FILE} ${TMP_SWAGGER}

${BINPATH} validate ${SWAGGER_FILE}
