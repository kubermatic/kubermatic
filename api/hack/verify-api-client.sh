#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

function cleanup() {
	if [[ -n "${TMP_DIFFROOT:-}" ]]; then
		rm -rf "${TMP_DIFFROOT}"
	fi
}
trap cleanup EXIT SIGINT SIGTERM

SCRIPT_ROOT=$(dirname "${BASH_SOURCE}")/..
API_DIR="$(go env GOPATH)/src/github.com/kubermatic/kubermatic/api"
DIFFROOT="${API_DIR}/pkg/test/e2e/api/utils/apiclient"

TMP_DIFFROOT=$(mktemp -d)

cp -a "${DIFFROOT}"/* "${TMP_DIFFROOT}"

"${SCRIPT_ROOT}/hack/gen-api-client.sh" &>/dev/null

echo "diffing ${DIFFROOT} against freshly generated api client"
ret=0
diff -Naupr "${DIFFROOT}" "${TMP_DIFFROOT}" || ret=$?
cp -a "${TMP_DIFFROOT}"/client "${DIFFROOT}"
cp -a "${TMP_DIFFROOT}"/models "${DIFFROOT}"

if [[ $ret -eq 0 ]]; then
	echo "${DIFFROOT} up to date."
else
	echo "${DIFFROOT} is out of date. Please run hack/gen-api-client.sh"
	exit 1
fi
