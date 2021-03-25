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

### Contains commonly used functions for the other scripts.

# Required for signal propagation to work so
# the cleanup trap gets executed when a script
# receives a SIGINT
set -o monitor

worker_name() {
  echo "${KUBERMATIC_WORKERNAME:-$(uname -n)}" | tr -cd '[:alnum:]' | tr '[:upper:]' '[:lower:]'
}

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
  retries=$1
  shift

  count=0
  delay=1
  until "$@"; do
    rc=$?
    count=$((count + 1))
    if [ $count -lt "$retries" ]; then
      echo "Retry $count/$retries exited $rc, retrying in $delay seconds..." > /dev/stderr
      sleep $delay
    else
      echo "Retry $count/$retries exited $rc, no more retries left." > /dev/stderr
      return $rc
    fi
    delay=$((delay * 2))
  done
  return 0
}

echodate() {
  # do not use -Is to keep this compatible with macOS
  echo "[$(date +%Y-%m-%dT%H:%M:%S%:z)]" "$@"
}

write_junit() {
  # Doesn't make any sense if we don't know a testname
  if [ -z "${TEST_NAME:-}" ]; then return; fi
  # Only run in CI
  if [ -z "${ARTIFACTS:-}" ]; then return; fi

  rc=$1
  duration=${2:-0}
  errors=0
  failure=""
  if [ "$rc" -ne 0 ]; then
    errors=1
    failure='<failure type="Failure">Step failed</failure>'
  fi
  TEST_NAME="[Kubermatic] ${TEST_NAME#\[Kubermatic\] }"
  cat << EOF > ${ARTIFACTS}/junit.$(echo $TEST_NAME | sed 's/ /_/g').xml
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

containerize() {
  local cmd="$1"
  local image="${CONTAINERIZE_IMAGE:-quay.io/kubermatic/util:1.5.0}"
  local gocache="${CONTAINERIZE_GOCACHE:-/tmp/.gocache}"

  if ! [ -f /.dockerenv ]; then
    echodate "Running $cmd in a Docker container using $image..."

    exec docker run \
      -v $PWD:/go/src/k8c.io/kubermatic \
      -w /go/src/k8c.io/kubermatic \
      -e "GOCACHE=$gocache" \
      -u "$(id -u):$(id -g)" \
      --entrypoint="$cmd" \
      --rm \
      -it \
      $image $@

    exit $?
  fi
}

ensure_github_host_pubkey() {
  # check whether we already have a known_hosts entry for Github
  if ssh-keygen -F github.com > /dev/null 2>&1; then
    echo " [*] Github's SSH host key already present" > /dev/stderr
  else
    local github_rsa_key
    # https://help.github.com/en/github/authenticating-to-github/githubs-ssh-key-fingerprints
    github_rsa_key="github.com ssh-rsa AAAAB3NzaC1yc2EAAAABIwAAAQEAq2A7hRGmdnm9tUDbO9IDSwBK6TbQa+PXYPCPy6rbTrTtw7PHkccKrpp0yVhp5HdEIcKr6pLlVDBfOLX9QUsyCOV0wzfjIJNlGEYsdlLJizHhbn2mUjvSAHQqZETYP81eFzLQNnPHt4EVVUh7VfDESU84KezmD5QlWpXLmvU31/yMf+Se8xhHTvKSCZIFImWwoG6mbUoWf9nzpIoaSjB+weqqUUmpaaasXVal72J+UX2B+2RPW3RcT0eOzQgqlJL3RKrTJvdsjE3JEAvGq3lGHSZXy28G3skua2SmVi/w4yCE6gbODqnTWlg7+wC604ydGXA8VJiS5ap43JXiUFFAaQ=="

    echo " [*] Adding Github's SSH host key to known hosts" > /dev/stderr
    mkdir -p "$HOME/.ssh"
    chmod 700 "$HOME/.ssh"
    echo "$github_rsa_key" >> "$HOME/.ssh/known_hosts"
    chmod 600 "$HOME/.ssh/known_hosts"
  fi
}

get_latest_dashboard_hash() {
  local FOR_BRANCH="$1"

  ensure_github_host_pubkey
  git config --global core.sshCommand 'ssh -o CheckHostIP=no -i /ssh/id_rsa'
  local DASHBOARD_URL="git@github.com:kubermatic/dashboard.git"

  # `local` always sets the rc to 0, so declare as local _before_ doing the substitution
  # which may fail
  local HASH
  HASH="$(retry 5 git ls-remote "$DASHBOARD_URL" "refs/heads/$FOR_BRANCH" | awk '{print $1}')"
  echodate "The latest dashboard hash for $FOR_BRANCH is $HASH" > /dev/stderr
  echo "$HASH"
}

# This should only be used for release branches, as it only fetches the last 25 commits
# for checking for tags.
get_latest_dashboard_tag() {
  local FOR_BRANCH="$1"
  local DEPTH="${2:-25}"

  ensure_github_host_pubkey
  git config --global core.sshCommand 'ssh -o CheckHostIP=no -i /ssh/id_rsa'
  local DASHBOARD_URL="git@github.com:kubermatic/dashboard.git"

  local TMPDIR
  TMPDIR=$(mktemp -d dashboard.XXXXX)

  # git ls-remote cannot list tags in a meaningful way, so we have to clone the repo
  echodate "Cloning dashboard repository to find tags in $FOR_BRANCH branch..." > /dev/stderr
  git clone -b "$FOR_BRANCH" --single-branch --depth $DEPTH "$DASHBOARD_URL" "$TMPDIR"

  local TAG
  TAG="$(git --git-dir $TMPDIR/.git describe --abbrev=0 --tags --first-parent)"

  echodate "The latest dashboard tag in $FOR_BRANCH is $TAG" > /dev/stderr
  echo "$TAG"
}

check_dashboard_tag() {
  local TAG="$1"

  ensure_github_host_pubkey
  git config --global core.sshCommand 'ssh -o CheckHostIP=no -i /ssh/id_rsa'
  local DASHBOARD_URL="git@github.com:kubermatic/dashboard.git"

  retry 5 git ls-remote "$DASHBOARD_URL" "refs/tags/$TAG"
}

format_dashboard() {
  local filename="$1"
  local tmpfile="$filename.tmp"

  cat "$filename" |
    jq '(.templating.list[] | select(.type=="query") | .options) = []' |
    jq '(.templating.list[] | select(.type=="query") | .refresh) = 2' |
    jq '(.templating.list[] | select(.type=="query") | .current) = {}' |
    jq '(.templating.list[] | select(.type=="datasource") | .current) = {}' |
    jq '(.templating.list[] | select(.type=="interval") | .current) = {}' |
    jq '(.panels[] | select(.scopedVars!=null) | .scopedVars) = {}' |
    jq '(.templating.list[] | select(.type=="datasource") | .hide) = 2' |
    jq '(.annotations.list) = []' |
    jq '(.links) = []' |
    jq '(.refresh) = "30s"' |
    jq '(.time.from) = "now-6h"' |
    jq '(.editable) = true' |
    jq '(.panels[] | select(.type!="row") | .editable) = true' |
    jq '(.panels[] | select(.type!="row") | .transparent) = true' |
    jq '(.panels[] | select(.type!="row") | .timeRegions) = []' |
    jq '(.hideControls) = false' |
    jq '(.time.to) = "now"' |
    jq '(.timezone) = ""' |
    jq '(.graphTooltip) = 1' |
    jq 'del(.panels[] | select(.repeatPanelId!=null))' |
    jq 'del(.id)' |
    jq 'del(.iteration)' |
    jq --sort-keys '.' > "$tmpfile"

  mv "$tmpfile" "$filename"
}

# appendTrap appends to existing traps, if any. It is needed because Bash replaces existing handlers
# rather than appending: https://stackoverflow.com/questions/3338030/multiple-bash-traps-for-the-same-signal
# Needing this func is a strong indicator that Bash is not the right language anymore. Also, this
# basically needs unit tests.
appendTrap() {
  command="$1"
  signal="$2"

  # Have existing traps, must append
  if [[ "$(trap -p | grep $signal)" ]]; then
    existingHandlerName="$(trap -p | grep $signal | awk '{print $3}' | tr -d "'")"

    newHandlerName="${command}_$(head /dev/urandom | tr -dc A-Za-z0-9 | head -c 13)"
    # Need eval to get a random func name
    eval "$newHandlerName() { $command; $existingHandlerName; }"
    echodate "Appending $command as trap for $signal, existing command $existingHandlerName"
    trap $newHandlerName $signal
  # First trap
  else
    echodate "Using $command as trap for $signal"
    trap $command $signal
  fi
}

# returns the current time as a number of milliseconds
nowms() {
  echo $(($(date +%s%N) / 1000000))
}

# returns the number of milliseconds elapsed since the given time
elapsed() {
  echo $(($(nowms) - $1))
}

# pushes a Prometheus metric to a pushgateway
pushMetric() {
  local metric="$1"
  local value="$2"
  local labels="${3:-}"
  local kind="${4:-gauge}"
  local help="${5:-}"
  local pushgateway="${PUSHGATEWAY_URL:-}"
  local job="ci"
  local instance="$PROW_JOB_ID"
  local prowjob="$JOB_NAME"

  if [ -z "$pushgateway" ]; then
    return
  fi

  local payload="# TYPE $metric $kind"

  if [ -n "$help" ]; then
    payload="$payload\n# HELP $metric $help"
  fi

  if [ -n "$labels" ]; then
    labels=",$labels"
  fi

  payload="$payload\n$metric{prowjob=\"$prowjob\"$labels} $value\n"

  echo -e "$payload" | curl --data-binary @- -s "$pushgateway/metrics/job/$job/instance/$instance"
}

pushElapsed() {
  pushMetric "$1" $(elapsed $2) "${3:-}" "${4:-}" "${5:-}"
}

# err print an error log to stderr
err() {
  echo "$(date) E: $*" >> /dev/stderr
}

# fatal can be used to print logs to stderr
fatal() {
  echo "$(date) F: $*" >> /dev/stderr
  exit 1
}

check_all_deployments_ready() {
  local namespace="$1"

  # check that Deployments have been created
  local deployments
  deployments=$(kubectl -n $namespace get deployments -o json)

  if [ $(echo "$deployments" | jq '.items | length') -eq 0 ]; then
    echodate "No Deployments created yet."
    return 1
  fi

  # check that all Deployments are ready
  local unready
  unready=$(echo "$deployments" | jq -r '[.items[] | select(.spec.replicas > 0) | select (.status.availableReplicas < .spec.replicas) | .metadata.name] | @tsv')
  if [ -n "$unready" ]; then
    echodate "Not all Deployments have finished rolling out, namely: $unready"
    return 1
  fi

  return 0
}

cleanup_kubermatic_clusters_in_kind() {
  # Tolerate errors and just continue
  set +e
  # Clean up clusters
  echodate "Cleaning up clusters..."
  kubectl delete cluster --all --ignore-not-found=true
  echodate "Done cleaning up clusters"

  # Kill all descendant processes
  pkill -P $$
  set -e
}

start_docker_daemon() {
  # Start Docker daemon
  echodate "Starting Docker"
  dockerd > /tmp/docker.log 2>&1 &
  echodate "Started Docker successfully"

  function docker_logs {
    if [[ $? -ne 0 ]]; then
      echodate "Printing Docker logs"
      cat /tmp/docker.log
      echodate "Done printing Docker logs"
    fi
  }
  appendTrap docker_logs EXIT

  # Wait for Docker to start
  echodate "Waiting for Docker"
  retry 5 docker stats --no-stream
  echodate "Docker became ready"
}
