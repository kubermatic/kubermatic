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

# This script is run for every tagged revision and will create
# the appropriate GitHub release and upload source archives.

set -euo pipefail

cd $(dirname $0)/../..
source hack/lib.sh

GITHUB_TOKEN="${GITHUB_TOKEN:-$(cat /etc/github/oauth | tr -d '\n')}"

# this stops execution when we are not on a tagged revision
tag="$(git describe --tags --exact-match)"
branch=$(git rev-parse --abbrev-ref HEAD)
head="$(git rev-parse HEAD)"
repo="kubermatic/kubermatic"
auth="Authorization: token $GITHUB_TOKEN"

# ensure the tag has already been pushed
if [ -z "$(curl -s -H "$auth" "https://api.github.com/repos/$repo/tags" | jq ".[] | select(.name==\"$tag\")")" ]; then
  echodate "Tag $tag has not been pushed to $repo yet."
  exit 1
fi

prerelease=false
if [[ "$tag" =~ "-" ]]; then
  prerelease=true
fi

# create a nice-sounding release name
name=$(echo "$tag" | sed -E 's/-beta\.([0-9]+)/ (Beta \1)/')
name=$(echo "$name" | sed -E 's/-rc\.([0-9]+)/ (Release Candidate \1)/')

echodate "Release name: $name"
echodate "Current tag : $tag ($branch @ $head)"
echodate "Pre-Release : $prerelease"

# retrieve release info
echodate "Checking release existence..."
releasedata="$(curl -sf -H "$auth" "https://api.github.com/repos/$repo/releases/tags/$tag" || true)"

if [ -z "$releasedata" ]; then
  echodate "Creating release..."

  curl -s -H "$auth" "https://api.github.com/repos/$repo/releases" --data @- > /dev/null << EOF
{
  "tag_name": "$tag",
  "name": "$name",
  "prerelease": $prerelease
}
EOF

  releasedata="$(curl -sf -H "$auth" "https://api.github.com/repos/$repo/releases/tags/$tag")"
fi

releaseID=$(echo "$releasedata" | jq -r '.id')

upload() {
  curl -s -H "$auth" -H 'Content-Type: application/gzip' --data-binary "@$1" \
       "https://uploads.github.com/repos/$repo/releases/$releaseID/assets?name=$1" > /dev/null
  rm -- "$1"
}

# prepare source for archiving
sed -i "s/__DASHBOARD_TAG__/$tag/g" charts/kubermatic/*.yaml
sed -i "s/__KUBERMATIC_TAG__/$tag/g" charts/kubermatic/*.yaml
sed -i "s/__KUBERMATIC_TAG__/$tag/g" charts/kubermatic-operator/*.yaml
sed -i "s/__KUBERMATIC_TAG__/$tag/g" charts/nodeport-proxy/*.yaml

echodate "Uploading kubermatic CE main archive..."

archive="kubermatic-ce-$tag.tar.gz"
tar czf "$archive" \
  charts/cert-manager \
  charts/kubermatic/crd \
  charts/kubermatic-operator \
  charts/minio \
  charts/nginx-ingress-controller \
  charts/nodeport-proxy \
  charts/oauth \
  LICENSE \
  README.md \
  CHANGELOG.md

upload "$archive"

echodate "Uploading kubermatic EE main archive..."

archive="kubermatic-ee-$tag.tar.gz"
tar czf "$archive" \
  charts/cert-manager \
  charts/kubermatic \
  charts/kubermatic-operator \
  charts/minio \
  charts/nginx-ingress-controller \
  charts/nodeport-proxy \
  charts/oauth \
  LICENSE \
  README.md \
  CHANGELOG.md

upload "$archive"

echodate "Uploading monitoring stack archive..."

archive="monitoring-stack-$tag.tar.gz"
tar czf "$archive" \
  charts/cert-manager \
  charts/iap \
  charts/minio \
  charts/monitoring \
  charts/oauth \
  LICENSE \
  README.md \
  CHANGELOG.md

upload "$archive"

echodate "Uploading logging stack archive..."

archive="logging-stack-$tag.tar.gz"
tar czf "$archive" \
  charts/cert-manager \
  charts/iap \
  charts/minio \
  charts/logging \
  charts/monitoring/grafana \
  charts/oauth \
  LICENSE \
  README.md \
  CHANGELOG.md

upload "$archive"

echodate "Uploading backup stack archive..."

archive="backup-stack-$tag.tar.gz"
tar czf "$archive" \
  charts/backup \
  LICENSE \
  README.md \
  CHANGELOG.md

upload "$archive"

echodate "Done."
