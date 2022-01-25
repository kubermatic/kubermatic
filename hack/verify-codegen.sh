#!/usr/bin/env bash

# Copyright 2020 The Kubermatic Kubernetes Platform contributors.
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

set -euo pipefail

cd $(dirname $0)/..
source hack/lib.sh

TEMPDIR=_tmp
PKG_DIFFROOT=pkg
TMP_PKG_DIFFROOT="$TEMPDIR/pkg"
CHARTS_DIFFROOT=charts
TMP_CHARTS_DIFFROOT="$TEMPDIR/charts"

cleanup() {
  rm -rf "$TEMPDIR"
}
trap "cleanup" EXIT SIGINT

cleanup

mkdir -p "${TMP_PKG_DIFFROOT}" "${TMP_CHARTS_DIFFROOT}"
cp -a "${PKG_DIFFROOT}"/* "${TMP_PKG_DIFFROOT}"
cp -a "${CHARTS_DIFFROOT}"/* "${TMP_CHARTS_DIFFROOT}"

# This will update both generated code and the CRDs for *.k8c.io,
# but until those new CRDs are live, we only diff pkg/. We have
# to restore the CRDs afterwards, so that other verify-* scripts
# do not get confused when they `git diff`.
./hack/update-codegen.sh

# restore CRDs
cp -a "${TMP_CHARTS_DIFFROOT}"/* "${CHARTS_DIFFROOT}"

echodate "Diffing ${PKG_DIFFROOT} against freshly generated codegen"
ret=0
diff -Naupr "${PKG_DIFFROOT}" "${TMP_PKG_DIFFROOT}" || ret=$?

# restore pkg
cp -a "${TMP_PKG_DIFFROOT}"/* "${PKG_DIFFROOT}"

if [[ $ret -eq 0 ]]; then
  echodate "${PKG_DIFFROOT} up to date."
else
  echodate "${PKG_DIFFROOT} is out of date. Please run hack/update-codegen.sh"
  exit 1
fi
