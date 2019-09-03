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
  TEST_NAME="[Kubermatic] $TEST_NAME"
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

  # subshell, so that we don't change the directory of the caller, no matter how we call or exit this function
  (
    cd "$(mktemp -d)"
    git clone git@github.com:kubermatic/kubermatic-installer.git .

    if [[ "$FOR_BRANCH" == "master" ]]; then
      # just get the tip of dashboard master
      git show-ref refs/heads/master -s

      return
    fi

    cd "$(mktemp -d)"
    git clone git@github.com:kubermatic/kubermatic-installer.git .
    MINOR_VERSION="${FOR_BRANCH##release/}"
    FOUND_TAG="$(git for-each-ref "refs/tags/$MINOR_VERSION*" --sort=-authordate --format='%(refname)' --count=1)"
    if [ -z "$FOUND_TAG" ]; then
      echo "Error, no Dashboard tags contain $MINOR_VERSION" >/dev/stderr
      exit 1
    fi

    echo "${FOUND_TAG##refs/tags/}"
  )
}