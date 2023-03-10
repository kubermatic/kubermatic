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

VERSION=""
END=""

# Get arguments from cli
while getopts v:e: opts; do
  case ${opts} in
  v)
    VERSION=${OPTARG}
    ;;
  e)
    END=${OPTARG}
    ;;
  esac
done

if [ -z "$VERSION" ]; then
  echo "Usage: ./changelog-gen.sh -v VERSION [-e END_GIT_HASH]"
  echo ""
  echo "       Use -e to perform incremental changelogs: Once the"
  echo "       beta changelog has been generated, -e can be used"
  echo "       to only include changes since the beta when updating"
  echo "       the changelog for the final release."
  exit 1
fi

# Install gchl if not installed
if ! [ -x "$(command -v gchl)" ]; then
  echo "Installing k8c.io/gchl…"
  go install k8c.io/gchl@latest
fi

FILENAME="$VERSION-$(date +%s).md"

echo "Generating changelog in $FILENAME…"
gchl \
  --organization kubermatic \
  --repository kubermatic \
  --end "$END" \
  --for-version "$VERSION" > "$FILENAME"
