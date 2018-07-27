#!/bin/bash
# Create a changelog since last release, commit and create a new release tag
#
# Usage:
# changelog-gen.sh -r v2.x.x - create changelog, commit and tag new release, using closed PRs release-note

set -eo pipefail

CHANGELOG_FILE="CHANGELOG.md"

# Get arguments from cli
while getopts r:help: opts; do
   case ${opts} in
      r) NEW_RELEASE=${OPTARG} ;;
   esac
done

# Show usage
if [ "$1" == "help" ] ; then
    echo "Usage: changelog-gen.sh -r v2.x.x - create changelog, commit and tag new release, using closed PRs release-note"
    exit 0
fi

# Check if a version flag is provided.
if [ "$NEW_RELEASE" == "" ]; then
	echo "Usage: changelog-gen.sh -r v2.x.x - create changelog, commit and tag new release, using closed PRs release-note"
	exit 1
fi

# Install gchl if not installed
if ! [ -x "$(command -v gchl)" ]; then
	echo "Error: gchl not installed!" >&2
	echo "Executing go get on github.com/kubermatic/gchl ..."
	go get github.com/kubermatic/gchl
	echo "Done!"
fi

# Create CHANGELOG.md if not exists
if [ ! -f $CHANGELOG_FILE ]; then
	touch $CHANGELOG_FILE
fi

# Generate changelog
OUTPUT="$(gchl --for-version $NEW_RELEASE since $(git tag -l | tail -n 1) --release-notes)"; if [ $? -eq 1 ]; then exit 1; fi
echo "${OUTPUT}"https:// | cat - $CHANGELOG_FILE > temp
mv temp $CHANGELOG_FILE

# Commit generated changelog and create a new tag
echo "Creating new commit and tag new release"
git add $CHANGELOG_FILE
git commit -m "Added changelog for new release $NEW_RELEASE"
git tag $NEW_RELEASE
