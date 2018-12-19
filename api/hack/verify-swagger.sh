#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

function cleanup() {
    rm -f $TMP_SWAGGER

    if [[ -n "${TMP_DIR:-}" ]]; then
        rm -rf "${TMP_DIR}"
    fi
}
trap cleanup EXIT SIGINT SIGTERM

API_DIR="$(go env GOPATH)/src/github.com/kubermatic/kubermatic/api"
SWAGGER_FILE="swagger.json"
TMP_SWAGGER="${SWAGGER_FILE}.tmp"

# install swagger to temp dir
TMP_DIR=$(mktemp -d)
mkdir -p "${TMP_DIR}/bin"

cd ${API_DIR}/vendor/github.com/go-swagger/go-swagger/cmd/swagger
env "GOBIN=${TMP_DIR}/bin" go install
export PATH="${TMP_DIR}/bin:${PATH}"

cd ${API_DIR}/cmd/kubermatic-api/
swaggger generate spec --scan-models -o ${TMP_SWAGGER}
diff -Naup ${SWAGGER_FILE} ${TMP_SWAGGER}

swaggger validate ${SWAGGER_FILE}
