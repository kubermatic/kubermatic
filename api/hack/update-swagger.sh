#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

API_DIR="$(realpath "$(dirname "$0")"/..)"
SWAGGER_FILE="swagger.json"
SWAGGER_BIN="go run github.com/go-swagger/go-swagger/cmd/swagger"

cd "${API_DIR}"/cmd/kubermatic-api
$SWAGGER_BIN generate spec --scan-models -o ${SWAGGER_FILE}
