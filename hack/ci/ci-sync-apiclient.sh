#!/bin/sh

set -eu

. $(dirname $0)/../lib.sh

URL="git@github.com:kubermatic/go-kubermatic.git"

commit_and_push() {
    local repodir branch source_sha
    repodir="$1"
    branch="$2"
    source_sha="$3"

    (
        cd "$repodir"

        git config --local user.email "dev@loodse.com"
        git config --local user.name "Prow CI Robot"
        git config --local core.sshCommand 'ssh -o CheckHostIP=no -i /ssh/id_rsa'

        git add .
        if ! git status | grep -q 'nothing to commit'; then
            git commit -m "Syncing client from Kubermatic $source_sha"
            git push origin "$branch"
        fi
    )
}

if [ -z "${PULL_BASE_REF=}" ]; then
    echo "\$PULL_BASE_REF undefined, I don't know which branch to sync."
    exit 1
fi

if [ -n "$(git rev-parse --show-prefix)" ]; then
    echo "You must run this script from repo root"
    exit 1
fi

# Ensure Github's host key is available and disable IP checking.
ensure_github_host_pubkey

# clone the target and pick the right branch
tempdir="$(mktemp -d)"
trap "rm -rf '$tempdir'" EXIT
GIT_SSH_COMMAND="ssh -o CheckHostIP=no -i /ssh/id_rsa" git clone "$URL" "$tempdir"
(
    cd "$tempdir"
    git checkout "$PULL_BASE_REF" || git checkout -b "$PULL_BASE_REF"
)

# rewrite all the import paths
echo "Rewriting import paths"
sed_expression="s#github.com/kubermatic/kubermatic/pkg/test/e2e/api/utils/apiclient#github.com/kubermatic/go-kubermatic#g"
time find pkg/test/e2e/api/utils/apiclient/ -type f -exec sed "$sed_expression" -i {} \;

# sync the files
echo "Synchronizing the files"
rsync --archive --verbose --delete "./pkg/test/e2e/api/utils/apiclient/client/" "$tempdir/client/"
rsync --archive --verbose --delete "./pkg/test/e2e/api/utils/apiclient/models/" "$tempdir/models/"

# commit and push
commit_and_push "$tempdir" "$PULL_BASE_REF" "$(git rev-parse HEAD)"
