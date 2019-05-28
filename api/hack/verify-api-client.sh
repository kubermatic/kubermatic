#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

function cleanup() {

    if [[ -n "${TMP_DIR:-}" ]]; then
        rm -rf "${TMP_DIR}"
    fi
}
trap cleanup EXIT SIGINT SIGTERM

API_DIR="$(go env GOPATH)/src/github.com/kubermatic/kubermatic/api"
API_CLIENT_DIR="$(go env GOPATH)/src/github.com/kubermatic/kubermatic/api/pkg/test/e2e/api/utils/apiclient"
SWAGGER_FILE="swagger.json"


# install swagger to temp dir
TMP_DIR=$(mktemp -d)
mkdir -p "${TMP_DIR}/bin"
mkdir -p "${TMP_DIR}/src"

cd ${API_DIR}/vendor/github.com/go-swagger/go-swagger/cmd/swagger

env "GOBIN=${TMP_DIR}/bin" go install
export PATH="${TMP_DIR}/bin:${PATH}"
export GOPATH=${TMP_DIR}

cd ${API_DIR}/cmd/kubermatic-api/

swagger generate client -q -f ${SWAGGER_FILE} -t ${TMP_DIR}/src
diff -Naupr ${API_CLIENT_DIR}/models ${TMP_DIR}/src/models

