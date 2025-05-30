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

### This script is run for every tagged revision and will create
### the appropriate GitHub release and upload source archives. It can be
### tweaked by setting a number of environment variables, most importantly:
###
### * `DRY_RUN=false` - set to `true` to not actually talk to GitHub
### * `GITHUB_TOKEN` - the OAuth token to use
###
### Whenever this is changed, run the optional `pre-kubermatic-simulate-github-release`
### job to (mostly) ensure that the changes will work.

set -euo pipefail

cd $(dirname $0)/../..
source hack/lib.sh

DRY_RUN=${DRY_RUN:-false}

GITHUB_TOKEN="${GITHUB_TOKEN:-$(cat /etc/github/oauth | tr --delete '\n')}"
export GITHUB_AUTH="Authorization: token $GITHUB_TOKEN"

# this stops execution when GIT_TAG is not overridden and
# we are not on a tagged revision
export GIT_TAG="${GIT_TAG:-$(git describe --tags --exact-match --match='v*')}"
export GIT_BRANCH="${GIT_BRANCH:-$(git rev-parse --abbrev-ref HEAD)}"
export GIT_HEAD="${GIT_HEAD:-$(git rev-parse HEAD)}"
export GIT_REPO="${GIT_REPO:-kubermatic/kubermatic}"
export RELEASE_PLATFORMS="${RELEASE_PLATFORMS:-linux-amd64 darwin-amd64 darwin-arm64 windows-amd64}"

# By default, this script is used to released tagged revisions,
# for which a matching tag must exist in the dashboard repository.
export DASHBOARD_GIT_TAG="${DASHBOARD_GIT_TAG:-$GIT_TAG}"

# RELEASE_NAME allows to customize the tag that is used to create the
# Github release for, while the Helm charts and things will still
# point to GIT_TAG
export RELEASE_NAME="${RELEASE_NAME:-$GIT_TAG}"

# utility function setting some curl default values for calling the github API
# first argument is the URL, the rest of the arguments is used as curl
# arguments.
function github_cli {
  local url="$1"
  curl \
    --retry 5 \
    --connect-timeout 10 \
    --header "$GITHUB_AUTH" \
    "${@:2}" "$url"
}

# creates a new github release
function create_release {
  local tag="$1"
  local name="$2"
  local prerelease="$3"
  local body="${4:-}"
  local data

  # using named arguments is a nice way to ensure special
  # characters in the body survive the JSON encoding
  data="$(
    jq --null-input \
      --arg tag_name "$tag" \
      --arg name "$name" \
      --argjson prerelease "$prerelease" \
      --arg body "$body" \
      '{"tag_name":$tag_name,"name":$name,"prerelease":$prerelease,"body":$body}'
  )"

  github_cli \
    "https://api.github.com/repos/$GIT_REPO/releases" \
    --fail --data "$data"
}

# upload an archive from a file
function upload_archive {
  local file="$1"
  res=$(github_cli \
    "https://uploads.github.com/repos/$GIT_REPO/releases/$releaseID/assets?name=$(basename "$file")" \
    --header 'Accept: application/json' \
    --header 'Content-Type: application/gzip' \
    --silent --data-binary "@$file")

  if echo "$res" | jq --exit-status '.' > /dev/null; then
    # if the response contain errors
    if echo "$res" | jq --exit-status '.errors[0]' > /dev/null; then
      for err in $(echo "$res" | jq --raw-output '.errors[0].code'); do
        # if the error code is 'already_exists' do not fail to make this call
        # idempotent. To make it better we should also check that the content
        # match.
        [[ "$err" == "already_exists" ]] && return 0
      done
      echodate "Response contains unexpected errors: $res"
      return 1
    fi
    return 0
  else
    echodate "Response did not contain valid JSON: $res"
    return 1
  fi
}

# always tar'ing and the converting to zip as required
# saves us from having to handle the filename munging
# logic twice
function tar_to_zip() {
  local archive="$(realpath "$1")"

  tmpdir="$(mktemp --directory)"
  tar xzf "$archive" --directory "$tmpdir"
  rm -- "$archive"

  archive="$(echo "$archive" | sed 's/.tar.gz/.zip/')"
  (
    cd "$tmpdir"
    zip --recurse-paths --quiet "$archive" .
  )
  rm -rf -- "$tmpdir"

  echo "$archive"
}

function build_installer() {
  make clean kubermatic-installer
  if [ "$GOOS" == "windows" ]; then
    mv _build/kubermatic-installer _build/kubermatic-installer.exe
  fi
}

function ship_archive() {
  local archive="$1"
  local buildTarget="$2"

  if [ "$GOOS" == "windows" ]; then
    echodate "Converting $archive to Zip..."
    archive="$(tar_to_zip "$archive")"
  fi

  if ! $DRY_RUN; then
    echodate "Uploading $buildTarget archive..."
    upload_archive "$archive"
    rm -- "$archive"
  fi
}

# ensure the tag has already been pushed
if ! $DRY_RUN && ! github_cli "https://api.github.com/repos/$GIT_REPO/git/ref/tags/$RELEASE_NAME" --silent --fail > /dev/null; then
  echodate "Tag $RELEASE_NAME has not been pushed to $GIT_REPO yet."
  exit 1
fi

prerelease=false
if [[ "$RELEASE_NAME" =~ "-" ]]; then
  prerelease=true
fi

# create a nice-sounding release name
name=$(echo "$RELEASE_NAME" | sed --regexp-extended 's/-alpha\.([0-9]+)/ (Alpha \1)/')
name=$(echo "$name" | sed --regexp-extended 's/-beta\.([0-9]+)/ (Beta \1)/')
name=$(echo "$name" | sed --regexp-extended 's/-rc\.([0-9]+)/ (Release Candidate \1)/')

echodate "Release name : $name"
echodate "Current tag  : $GIT_TAG ($GIT_BRANCH @ $GIT_HEAD)"

if [ "$RELEASE_NAME" != "$GIT_TAG" ]; then
  echodate "GitHub tag   : $RELEASE_NAME"
fi

echodate "Dashboard tag: $DASHBOARD_GIT_TAG"
echodate "Pre-Release  : $prerelease"

if $DRY_RUN; then
  echodate "This is a dry-run, no actual communication with GitHub happens."
fi

export KUBERMATICDOCKERTAG="$GIT_TAG"
export UIDOCKERTAG="$DASHBOARD_GIT_TAG"

# retrieve release info
if ! $DRY_RUN; then
  echodate "Checking release existence..."
  releasedata="$(github_cli "https://api.github.com/repos/$GIT_REPO/releases/tags/$RELEASE_NAME" --silent --fail || true)"

  if [ -z "$releasedata" ]; then
    echodate "Creating release..."

    create_release "$RELEASE_NAME" "$name" "$prerelease" "${RELEASE_DESCRIPTION:-}"

    releasedata="$(github_cli "https://api.github.com/repos/$GIT_REPO/releases/tags/$RELEASE_NAME" --silent --fail)"
  fi

  releaseID=$(echo "$releasedata" | jq -r '.id')
fi

# prepare source for archiving (prepend "9.9.9" if GIT_TAG is only
# a hash, because Helm requires a semver version)
CHART_TAG="$GIT_TAG"
if [[ "$CHART_TAG" != v* ]]; then
  CHART_TAG="9.9.9-$CHART_TAG"
fi

set_helm_charts_version "$CHART_TAG" "$GIT_TAG"

mkdir -p _dist

# CRDs since KKP 2.21 are not directly put into the charts/ directory
# anymore, but into pkg/ so they can be embedded. In our Github archives
# we still want and need them to be part of the operator chart.
copy_crds_to_chart
set_crds_version_annotation "$GIT_TAG"

for buildTarget in $RELEASE_PLATFORMS; do
  rm -rf _build

  export GOOS="$(echo "$buildTarget" | cut -d- -f1)"
  export GOARCH="$(echo "$buildTarget" | cut -d- -f2)"

  echodate "Compiling CE installer ($buildTarget)..."
  KUBERMATIC_EDITION=ce build_installer

  echodate "Creating CE archive..."

  # switch Docker repository used by the operator to the CE repository
  yq --inplace '.kubermaticOperator.image.repository = "quay.io/kubermatic/kubermatic"' charts/kubermatic-operator/values.yaml

  archive="_dist/kubermatic-ce-$RELEASE_NAME-$buildTarget.tar.gz"
  # GNU tar is required
  tar czf "$archive" \
    --transform='flags=r;s|_build/||' \
    --transform='flags=r;s|charts/values.example.ce.yaml|examples/values.example.yaml|' \
    --transform='flags=r;s|charts/values.example.mla.yaml|examples/values.example.mla.yaml|' \
    --transform='flags=r;s|charts/kubermatic.example.ce.yaml|examples/kubermatic.example.yaml|' \
    --transform='flags=r;s|charts/seed.example.yaml|examples/seed.example.yaml|' \
    --exclude='*/test' \
    _build/kubermatic-installer* \
    charts/backup \
    charts/cert-manager \
    charts/dex \
    charts/gitops \
    charts/iap \
    charts/kubermatic-operator \
    charts/local-kubevirt \
    charts/logging \
    charts/minio \
    charts/mla \
    charts/monitoring \
    charts/nginx-ingress-controller \
    charts/oauth \
    charts/s3-exporter \
    charts/telemetry \
    charts/values.example.ce.yaml \
    charts/values.example.mla.yaml \
    charts/kubermatic.example.ce.yaml \
    charts/seed.example.yaml \
    LICENSE \
    CHANGELOG.md

  ship_archive "$archive" "$buildTarget"

  echodate "Compiling EE installer ($buildTarget)..."
  KUBERMATIC_EDITION=ee build_installer

  echodate "Creating EE archive..."

  # switch Docker repository used by the operator to the EE repository
  yq --inplace '.kubermaticOperator.image.repository = "quay.io/kubermatic/kubermatic-ee"' charts/kubermatic-operator/values.yaml

  archive="_dist/kubermatic-ee-$RELEASE_NAME-$buildTarget.tar.gz"
  # GNU tar is required
  tar czf "$archive" \
    --transform='flags=r;s|_build/||' \
    --transform='flags=r;s|charts/values.example.ee.yaml|examples/values.example.yaml|' \
    --transform='flags=r;s|charts/values.example.mla.yaml|examples/values.example.mla.yaml|' \
    --transform='flags=r;s|charts/kubermatic.example.ee.yaml|examples/kubermatic.example.yaml|' \
    --transform='flags=r;s|charts/seed.example.yaml|examples/seed.example.yaml|' \
    --transform='flags=r;s|pkg/ee/LICENSE|LICENSE.ee|' \
    --exclude='*/test' \
    _build/kubermatic-installer* \
    charts/backup \
    charts/cert-manager \
    charts/dex \
    charts/gitops \
    charts/iap \
    charts/kubermatic-operator \
    charts/local-kubevirt \
    charts/logging \
    charts/minio \
    charts/mla \
    charts/monitoring \
    charts/nginx-ingress-controller \
    charts/oauth \
    charts/s3-exporter \
    charts/telemetry \
    charts/values.example.ee.yaml \
    charts/values.example.mla.yaml \
    charts/kubermatic.example.ee.yaml \
    charts/seed.example.yaml \
    LICENSE \
    pkg/ee/LICENSE \
    CHANGELOG.md

  ship_archive "$archive" "$buildTarget"
done

echodate "Done."
