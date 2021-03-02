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

### Create a changelog since last release, commit and create a new release tag
###
###     Usage:
###     changelog-gen.sh -r v2.x.x - create changelog, commit and tag new release, using closed PRs release-note

set -e

cd $(dirname $0)/..
CHANGELOG_FILE=CHANGELOG.md

usage() {
  echo "Usage: changelog-gen.sh -r NEW-RELEASE_TAG - create changelog, commit and tag new release, using closed PRs release-note"
  exit 1
}

# Get arguments from cli
while getopts r:help: opts; do
  case ${opts} in
  r)
    NEW_RELEASE=${OPTARG}
    ;;
  ?)
    usage
    ;;
  esac
done

# Check if a version flag is provided.
if [ "$NEW_RELEASE" == "" ]; then
  usage
fi

if [ "$(git rev-parse --abbrev-ref HEAD)" == "master" ]; then
  echo "Error, releases must not be created on master branch!"
  exit 1
fi

if git tag -l | grep -q $NEW_RELEASE; then
  echo "Error: Tag $NEW_RELEASE already exists!"
  exit 1
fi

if ! echo -n $NEW_RELEASE | egrep -q '^v(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$'; then
  echo "Error: Release version \"$NEW_RELEASE\" does not match regex '^v(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$'"
  echo "Valid version must look like this: v1.2.3"
  exit 1
fi

# Install gchl if not installed
if ! [ -x "$(command -v gchl)" ]; then
  echo "gchl not installed!"
  echo "Executing go get on github.com/kubermatic/gchl ..."
  go get github.com/kubermatic/gchl
  echo "Done!"
fi

# Create CHANGELOG.md if not exists in order for the cat not to fail
[ -f $CHANGELOG_FILE ] || touch $CHANGELOG_FILE

# Generate changelog
OUTPUT="$(gchl --for-version $NEW_RELEASE since $(git describe --abbrev=0 --tags) --release-notes)"
echo "${OUTPUT}" | cat - $CHANGELOG_FILE > temp
mv temp $CHANGELOG_FILE

# Commit generated changelog and create a new tag
echo "Creating new commit and tag new release"
git add $CHANGELOG_FILE
git commit -m "Added changelog for new release $NEW_RELEASE"
git tag $NEW_RELEASE

echo "Successfully created new release $NEW_RELEASE, verify via \`git show\`, then do a \`git push origin $(git rev-parse --abbrev-ref HEAD) --tags\`"
