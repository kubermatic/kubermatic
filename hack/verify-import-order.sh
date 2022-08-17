#!/usr/bin/env bash

# Copyright 2022 The Kubermatic Kubernetes Platform contributors.
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

ARCH=$(uname -m)
case $ARCH in
armv6*) ARCH="armv6" ;;
aarch64) ARCH="arm64" ;;
x86) ARCH="386" ;;
x86_64) ARCH="amd64" ;;
i686) ARCH="386" ;;
i386) ARCH="386" ;;
esac

version=0.5.0
# Remove gimps if already installed but not with the right version
if [ -x "$(command -v gimps)" ]; then
  iversion="$(gimps --version | grep version)"
  if [ ${iversion##version: } != $version ]; then
    echodate "gimps version is not the expected one... removing it before re-install with the right version"
    rm /usr/local/bin/gimps
  fi
fi

if ! [ -x "$(command -v gimps)" ]; then

  echodate "Downloading gimps v$version..."

  curl -LO https://github.com/xrstf/gimps/releases/download/v$version/gimps_${version}_${OS}_${ARCH}.zip
  unzip gimps_${version}_${OS}_${ARCH}.zip gimps
  mv gimps /usr/local/bin/

  echodate "Done!"
fi

echodate "Sorting import statements..."
gimps .

echodate "Diffing..."
if ! git diff --exit-code; then
  echodate "Some import statements are not properly grouped. Please run https://github.com/xrstf/gimps or sort them manually."
  exit 1
fi

echodate "Your Go import statements are in order :-)"
