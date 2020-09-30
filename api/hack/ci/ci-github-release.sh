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

cd $(dirname $0)/../../..
source api/hack/lib.sh

GITHUB_TOKEN="${GITHUB_TOKEN:-$(cat /etc/github/oauth | tr -d '\n')}"

# err can be used to print logs to stderr
err(){
  echo "E: $*" >>/dev/stderr
}

# utility function setting some curl default values for calling the github API
# first argument is the URL, the rest of the arguments is used as curl
# arguments.
function github_cli {
  local url=${1}
  curl \
    --retry 5 \
    --connect-timeout 10 \
    -H "Authorization: token ${GITHUB_TOKEN}" \
    "${@:2}" "${url}"
}

# creates a new github release
function create_release {
  local tag="${1}"
  local name="${2}"
  local prerelease="${3}"
  data=$(cat << EOF
{
  "tag_name": "$tag",
  "name": "$name",
  "prerelease": $prerelease
}
EOF
)
  github_cli \
    "https://api.github.com/repos/${repo}/releases" \
    -f --data "${data}"
}

# upload an archive from a file
function upload_archive {
  local file="${1}"
  res=$(github_cli \
    "https://uploads.github.com/repos/$repo/releases/$releaseID/assets?name=${file}" \
    -H "Accept: application/json" \
    -H 'Content-Type: application/gzip' \
    -s --data-binary "@${file}")
  if echo "${res}" | jq -e '.'; then
    # it the response contain errors
    if echo "${res}" | jq -e '.errors[0]'; then
      for err in $(echo "${res}" | jq -r '.errors[0].code'); do
        # if the error code is 'already_exists' do not fail to make this call
        # idempotent. To make it better we should alse check that the content
        # match.
        [[ "${err}" == "already_exists" ]] && return 0
      done
      err "Response contains unexpected errors: ${res}"
      return 1
    fi
    return 0
  else
    err "Response did not contain valid JSON: ${res}"
    return 1
  fi
}


# this stops execution when we are not on a tagged revision
tag="$(git describe --tags --exact-match)"
branch=$(git rev-parse --abbrev-ref HEAD)
head="$(git rev-parse HEAD)"
repo="${repo:-kubermatic/kubermatic}"
auth="Authorization: token $GITHUB_TOKEN"

# ensure the tag has already been pushed
if [ -z "$(github_cli "https://api.github.com/repos/$repo/tags" -s | jq ".[] | select(.name==\"$tag\")")" ]; then
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
releasedata="$(github_cli "https://api.github.com/repos/$repo/releases/tags/$tag" -sf || true)"

if [ -z "$releasedata" ]; then
  echodate "Creating release..."

  create_release "$tag" "$name" "$prerelease"

  releasedata="$(github_cli "https://api.github.com/repos/$repo/releases/tags/$tag" -sf)"
fi

releaseID=$(echo "$releasedata" | jq -r '.id')

# prepare source for archiving
sed -i "s/__DASHBOARD_TAG__/$tag/g" config/*/*.yaml
sed -i "s/__KUBERMATIC_TAG__/$tag/g" config/*/*.yaml

echodate "Uploading kubermatic CE archive..."

archive="kubermatic-ce-$tag.tar.gz"
# Gnu tar is required
tar czf "$archive" \
  --transform='flags=r;s|config/values.example.ce.yaml|examples/values.example.yaml|' \
  --transform='flags=r;s|config/test/|examples/|' \
  --transform='flags=r;s|config/|charts/|' \
  config/backup \
  config/cert-manager \
  config/iap \
  config/kubermatic-operator \
  config/kubermatic/crd \
  config/kubernetes-dashboard \
  config/logging/loki \
  config/logging/promtail \
  config/minio \
  config/monitoring \
  config/nginx-ingress-controller \
  config/nodeport-proxy \
  config/oauth \
  config/s3-exporter \
  config/values.example.ce.yaml \
  config/test/kubermatic.example.ce.yaml \
  config/test/seed.example.yaml \
  LICENSE \
  CHANGELOG.md

upload_archive "$archive"
rm -- "${archive}"

echodate "Uploading kubermatic EE archive..."

yq w -i config/kubermatic-operator/values.yaml 'kubermaticOperator.image.repository' 'quay.io/kubermatic/kubermatic-ee'

archive="kubermatic-ee-$tag.tar.gz"
# Gnu tar is required
tar czf "$archive" \
  --transform='flags=r;s|config/values.example.ee.yaml|examples/values.example.yaml|' \
  --transform='flags=r;s|config/test/|examples/|' \
  --transform='flags=r;s|config/|charts/|' \
  --transform='flags=r;s|api/pkg/ee/LICENSE|LICENSE.ee|' \
  config/backup \
  config/cert-manager \
  config/iap \
  config/kubermatic-operator \
  config/kubermatic \
  config/kubernetes-dashboard \
  config/logging \
  config/minio \
  config/monitoring \
  config/nginx-ingress-controller \
  config/nodeport-proxy \
  config/oauth \
  config/s3-exporter \
  config/values.example.ee.yaml \
  config/test/kubermatic.example.ee.yaml \
  config/test/seed.example.yaml \
  LICENSE \
  api/pkg/ee/LICENSE \
  CHANGELOG.md

git checkout -- config

upload_archive "$archive"
rm -- "${archive}"

echodate "Done."
