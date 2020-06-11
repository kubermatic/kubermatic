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
swagger generate spec --scan-models -o ${TMP_SWAGGER}
diff -Naup ${SWAGGER_FILE} ${TMP_SWAGGER}

swagger validate ${SWAGGER_FILE}
