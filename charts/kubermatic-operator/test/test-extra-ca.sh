#!/usr/bin/env bash

# Copyright 2026 The Kubermatic Kubernetes Platform contributors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

### Tests the CA bundle assembly that the golden master tests cannot cover:
### PEM files dropped into static/extra-ca are picked up, and invalid input
### aborts the rendering instead of installing a broken CA bundle.

set -euo pipefail

cd $(dirname "$0")/..
source ../../hack/lib.sh

EXTRA_CA_DIR=static/extra-ca
exitCode=0

cleanup() {
  rm -f $EXTRA_CA_DIR/test-*.pem
}
trap cleanup EXIT

countCertificates() {
  helm template . "$@" --show-only templates/ca-bundle.yaml | grep --count 'BEGIN CERTIFICATE' || true
}

expectCount() {
  local title="$1"
  local expected="$2"
  shift 2

  local actual
  actual=$(countCertificates "$@")

  if [ "$actual" == "$expected" ]; then
    echodate "  PASS $title ($actual certificates)"
  else
    echodate "  FAIL $title (expected $expected certificates, got $actual)"
    exitCode=1
  fi
}

expectFailure() {
  local title="$1"
  shift

  if helm template . "$@" --show-only templates/ca-bundle.yaml > /dev/null 2>&1; then
    echodate "  FAIL $title (rendering succeeded, but should have failed)"
    exitCode=1
  else
    echodate "  PASS $title"
  fi
}

echodate "(extra-ca) Testing kubermatic-operator CA bundle..."
cleanup

shipped=$(countCertificates)
echodate "The CA bundle shipped with the chart contains $shipped certificates."

openssl req -x509 -newkey rsa:2048 -nodes -keyout /dev/null \
  -out $EXTRA_CA_DIR/test-first.pem -days 1 -subj "/CN=kkp-chart-test-first" 2> /dev/null
expectCount "a dropped-in certificate is appended" $((shipped + 1))

openssl req -x509 -newkey rsa:2048 -nodes -keyout /dev/null \
  -out $EXTRA_CA_DIR/test-second.pem -days 1 -subj "/CN=kkp-chart-test-second" 2> /dev/null
expectCount "multiple dropped-in certificates are appended" $((shipped + 2))

# custom-ca.yaml replaces the shipped bundle with 1 certificate and appends 1 more,
# on top of the 2 certificates dropped in above.
expectCount "drop-ins are appended to a replaced bundle" 4 --values test/custom-ca.yaml

echo "this is not a certificate" > $EXTRA_CA_DIR/test-broken.pem
expectFailure "an invalid drop-in aborts the rendering"
rm -f $EXTRA_CA_DIR/test-broken.pem

cleanup
expectCount "the default bundle is restored" $shipped
expectFailure "an invalid caBundle.certificates aborts the rendering" --set caBundle.certificates="not a certificate"
expectFailure "an invalid caBundle.additionalCertificates aborts the rendering" --set caBundle.additionalCertificates="not a certificate"
expectCount "caBundle can be set to null" $shipped --set caBundle=null

exit $exitCode
