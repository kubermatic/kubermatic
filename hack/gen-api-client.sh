#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

function cleanup() {
    rm -f $TMP_SWAGGER

    cd ${API_DIR}/cmd/kubermatic-api/
    sed -i -e '1,16d' ../../pkg/handler/routes_v1.go
}
trap cleanup EXIT SIGINT SIGTERM

SWAGGER_META="// Kubermatic API.
// Kubermatic API. This describes possible operations which can be made against the Kubermatic API.
//
//     Schemes: https
//     Host: dev.kubermatic.io
//
//     Security:
//     - api_key:
//
//     SecurityDefinitions:
//     api_key:
//          type: apiKey
//          name: Authorization
//          in: header
//
// swagger:meta
"
KUBERMATIC_DIR="$(go env GOPATH)/src/github.com/kubermatic/kubermatic"
SWAGGER_FILE="swagger.json"
TMP_SWAGGER="${SWAGGER_FILE}.tmp"

# install swagger to temp dir
TMP_DIR=$(mktemp -d)
mkdir -p "${TMP_DIR}/bin"

cd ${KUBERMATIC_DIR}/vendor/github.com/go-swagger/go-swagger/cmd/swagger
env "GOBIN=${TMP_DIR}/bin" go install
export PATH="${TMP_DIR}/bin:${PATH}"
cd ${KUBERMATIC_DIR}/cmd/kubermatic-api/

echo "${SWAGGER_META}$(cat ../../pkg/handler/routes_v1.go)" > ../../pkg/handler/routes_v1.go
swagger generate spec --scan-models -o ${TMP_SWAGGER}
rm -r ../../pkg/test/e2e/api/utils/apiclient/
mkdir -p ../../pkg/test/e2e/api/utils/apiclient/
swagger generate client -q --skip-validation -f ${TMP_SWAGGER} -t ../../pkg/test/e2e/api/utils/apiclient/
