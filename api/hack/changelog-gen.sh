#!/bin/bash
# Create a changelog since last release, commit and create a new release tag
#
# Usage:
# changelog-gen.sh v2.x.x - create changelog, commit and tag new release, using closed PRs release-note
# changelog-gen.sh v2.x.x github-title - create changelog, commit and tag new release, using closed Githubs PRs title

set -eo pipefail

CHANGELOG_FILE="CHANGELOG.md"

gen_changelog() {
	OUTPUT="$(gchl --for-version $1 since $(git tag -l | tail -n 1) $2)"; if [ $? -eq 1 ]; then exit 1; fi
	echo "${OUTPUT}" | cat - $CHANGELOG_FILE > temp
	mv temp $CHANGELOG_FILE
}

new_release() {
	echo "Creating new commit and tag new release"
	git add $CHANGELOG_FILE
	git commit -m "Added changelog for new release $1"
	git tag $1
}

# Check wether gchl is installed or not
# If ghcl is not installed, execute go get and install latest version
if ! [ -x "$(command -v gchl)" ]; then
	echo "Error: gchl not installed!" >&2
	echo "Executing go get on https://github.com/kubermatic/gchl"
	go get -v https://github.com/kubermatic/gchl
fi

# Check if a version flag is provided.
# Version flag is required for commit message and tag creation
if [ "$1" == "" ]; then
	echo "Please provide a version tag in format v0.0.0 -- [example: changelog-gen.sh v2.x.x]"
	echo "Add release-notes flag to use closed PRs `release-notes` annotation content instead of PR title [example: changelog-gen.sh v2.x.x release-notes]"
	exit 1
fi

# Create CHANGELOG.md if not exists
if [ ! -f $CHANGELOG_FILE ]; then
	touch $CHANGELOG_FILE
fi

# Use use closed PR title instead of release-notes field
# or use closed PR title instead
if [ "$2" == "github-title" ]; then
	gen_changelog $1
	new_release $1
else
	gen_changelog $1 --release-notes
	new_release $1
fi
