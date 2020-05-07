#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

function cleanup() {
    rm -f "$TMP_SWAGGER"

    cd "${API_DIR}"/cmd/kubermatic-api/
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
API_DIR="$(realpath "$(dirname "$0")"/..)"
SWAGGER_FILE="swagger.json"
TMP_SWAGGER="${SWAGGER_FILE}.tmp"
SWAGGER_BIN="go run github.com/go-swagger/go-swagger/cmd/swagger"

echo "${SWAGGER_META}$(cat ../../pkg/handler/routes_v1.go)" >../../pkg/handler/routes_v1.go
$SWAGGER_BIN generate spec --scan-models -o ${TMP_SWAGGER}
mkdir -p ../../pkg/test/e2e/api/utils/apiclient/
$SWAGGER_BIN generate client -q --skip-validation -f ${TMP_SWAGGER} -t ../../pkg/test/e2e/api/utils/apiclient/
