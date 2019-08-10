retry() {
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
      write_junit $rc
      return $rc
    fi
    delay=$(( delay * 2 ))
  done
  write_junit 0
  return 0
}

echodate() {
  echo "$(date -Is)" "$@"
}

write_junit() {
  rc=$1
  # Doesn't make any sense if we don't know a testname
  [ -z "$TEST_NAME" ] && return
  # Only run in CI
  [ -z "${ARTIFACTS}" ] && return
  cat <<EOF > ${ARTIFACTS}/$(echo $TEST_NAME|sed 's/ /_/g').xml
<?xml version="1.0" ?>
<testsuites>
    <testsuite errors="$rc" failures="$rc" name="$TEST_NAME" tests="1">
        <testcase classname="$TEST_NAME" name="$TEST_NAME" time="0">
        </testcase>
    </testsuite>
</testsuites>
EOF
}
