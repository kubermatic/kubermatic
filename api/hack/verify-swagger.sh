#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

trap 'rm -f $TMP_SWAGGER' EXIT SIGINT SIGTERM

API_DIR="$(realpath "$(dirname "$0")"/..)"
SWAGGER_FILE="swagger.json"
TMP_SWAGGER="${SWAGGER_FILE}.tmp"
SWAGGER_BIN="go run github.com/go-swagger/go-swagger/cmd/swagger"

cd "${API_DIR}"/cmd/kubermatic-api
$SWAGGER_BIN generate spec --scan-models -o ${TMP_SWAGGER}
diff -Naup ${SWAGGER_FILE} ${TMP_SWAGGER}

$SWAGGER_BIN validate ${SWAGGER_FILE}
