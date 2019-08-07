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
      return $rc
    fi
    delay=$(( delay * 2 ))
  done
  return 0
}

echodate() {
  echo "$(date)" "$@"
}
