retry() {
  # Works only with bash but doesn't fail on other shells
  start_time=$(date +%s)
  set +e
  actual_retry $@
  rc=$?
  set -e
  elapsed_time=$(($(date +%s) - $start_time))
  write_junit "$rc" "$elapsed_time"
  return $rc
}

# We use an extra wrapping to write junit and have a timer
actual_retry() {
  retries=$1 ; shift

  count=0
  delay=1
  until "$@"; do
    rc=$?
    count=$(( count + 1 ))
    if [ $count -lt "$retries" ]; then
      echo "Retry $count/$retries exited $rc, retrying in $delay seconds..." >/dev/stderr
      sleep $delay
    else
      echo "Retry $count/$retries exited $rc, no more retries left." >/dev/stderr
      return $rc
    fi
    delay=$(( delay * 2 ))
  done
  return 0
}

echodate() {
  echo "$(date -Is)" "$@"
}

write_junit() {
  # Doesn't make any sense if we don't know a testname
  if [ -z "${TEST_NAME:-}" ]; then return; fi
  # Only run in CI
  if [ -z "$ARTIFACTS" ]; then return; fi

  rc=$1
  duration=${2:-0}
  errors=0
  failure=""
  if [ "$rc" -ne 0 ]; then
    errors=1
    failure='<failure type="Failure">Step failed</failure>'
  fi
  TEST_NAME="[Kubermatic] ${TEST_NAME#\[Kubermatic\] }"
  cat <<EOF > ${ARTIFACTS}/junit.$(echo $TEST_NAME|sed 's/ /_/g').xml
<?xml version="1.0" ?>
<testsuites>
    <testsuite errors="$errors" failures="$errors" name="$TEST_NAME" tests="1">
        <testcase classname="$TEST_NAME" name="$TEST_NAME" time="$duration">
          $failure
        </testcase>
    </testsuite>
</testsuites>
EOF
}

get_latest_dashboard_tag() {
  FOR_BRANCH="$1"

  git config --global core.sshCommand 'ssh -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -i /ssh/id_rsa'
  local DASHBOARD_URL="git@github.com:kubermatic/dashboard-v2.git"

  MINOR_VERSION="${FOR_BRANCH##release/}"
  # --sort=-version:refname will treat the tag names as semvers and sort by version number.
  # -c versionsort.suffix=-alpha -c versionsort.suffix=-rc tell git that alphas and RCs come before versions wihtout any suffix.
  FOUND_TAG="$(retry 5 git -c versionsort.suffix=-alpha -c versionsort.suffix=-rc ls-remote --sort=-version:refname "$DASHBOARD_URL" "refs/tags/$MINOR_VERSION*" | head -n1 | awk '{print $2}')"
  if [ -z "$FOUND_TAG" ]; then
    echo "Error, no Dashboard tags contain $MINOR_VERSION" >/dev/stderr
    exit 1
  fi

  local TAG="${FOUND_TAG##refs/tags/}"
  echodate "The latest dashboard tag for $FOR_BRANCH is $TAG" >/dev/stderr
  echo "$TAG"
}

get_latest_dashboard_hash() {
  FOR_BRANCH="$1"

  git config --global core.sshCommand 'ssh -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -i /ssh/id_rsa'
  local DASHBOARD_URL="git@github.com:kubermatic/dashboard-v2.git"

  # `local` always sets the rc to 0, so declare as local _before_ doing the substitution
  # which may fail
  local HASH
  HASH="$(retry 5 git ls-remote "$DASHBOARD_URL" "refs/heads/$FOR_BRANCH" | awk '{print $1}')"
  echodate "The latest dashboard hash for $FOR_BRANCH is $HASH" >/dev/stderr
  echo "$HASH"
}

format_dashboard() {
  local filename="$1"
  local tmpfile="$filename.tmp"

  cat "$filename" | \
    jq '(.templating.list[] | select(.type=="query") | .options) = []' | \
    jq '(.templating.list[] | select(.type=="query") | .refresh) = 2' | \
    jq '(.templating.list[] | select(.type=="query") | .current) = {}' | \
    jq '(.templating.list[] | select(.type=="datasource") | .current) = {}' | \
    jq '(.templating.list[] | select(.type=="interval") | .current) = {}' | \
    jq '(.panels[] | select(.scopedVars!=null) | .scopedVars) = {}' | \
    jq '(.templating.list[] | select(.type=="datasource") | .hide) = 2' | \
    jq '(.annotations.list) = []' | \
    jq '(.links) = []' | \
    jq '(.refresh) = "30s"' | \
    jq '(.time.from) = "now-6h"' | \
    jq '(.editable) = true' | \
    jq '(.panels[] | select(.type!="row") | .editable) = true' | \
    jq '(.panels[] | select(.type!="row") | .transparent) = true' | \
    jq '(.panels[] | select(.type!="row") | .timeRegions) = []' | \
    jq '(.hideControls) = false' | \
    jq '(.time.to) = "now"' | \
    jq '(.timezone) = ""' | \
    jq '(.graphTooltip) = 1' | \
    jq 'del(.panels[] | select(.repeatPanelId!=null))' | \
    jq 'del(.id)' | \
    jq 'del(.iteration)' | \
    jq --sort-keys '.' > "$tmpfile"

  mv "$tmpfile" "$filename"
}
