retry() {
  # Works only with bash but doesn't fail on other shells
  start_time=${SECONDS:-0}
  actual_retry $@
  elapsed_time=$((${SECONDS:-0} - $start_time))
  rc=$?
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
      echo "Retry $count/$retries exited $rc, retrying in $delay seconds..."
      sleep $delay
    else
      echo "Retry $count/$retries exited $rc, no more retries left."
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
  if [ -z "$TEST_NAME" ]; then return; fi
  # Only run in CI
  if [ -z "$ARTIFACTS" ]; then return; fi

  rc=$1
  duration=${2:-0}
  errors=0
  if [ "$rc" -ne 0 ]; then errors=1; fi
  TEST_NAME="[Kubermatic] $TEST_NAME"
  cat <<EOF > ${ARTIFACTS}/junit.$(echo $TEST_NAME|sed 's/ /_/g').xml
<?xml version="1.0" ?>
<testsuites>
    <testsuite errors="$errors" failures="$errors" name="$TEST_NAME" tests="1">
        <testcase classname="$TEST_NAME" name="$TEST_NAME" time="$duration">
        </testcase>
    </testsuite>
</testsuites>
EOF
}
