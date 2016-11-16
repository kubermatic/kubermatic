#!/usr/bin/env bash

# Copyright 2016 The Kubernetes Authors All rights reserved.
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

exit_code=0

if ! hash gometalinter 2>/dev/null ; then
  go get github.com/alecthomas/gometalinter
  gometalinter --install
fi

echo
echo "==> Running static validations <=="
# Run linters that should return errors
gometalinter \
  --disable-all \
  --enable deadcode \
  --severity deadcode:error \
  --enable gofmt \
  --enable gosimple \
  --enable ineffassign \
  --enable misspell \
  --enable vet \
  --tests \
  --vendor \
  ./... || exit_code=1

echo
echo "==> Running linters <=="
# Run linters that should return warnings
gometalinter \
  --disable-all \
  --enable golint \
  --vendor \
  --skip proto \
  ./... || :

exit $exit_code
