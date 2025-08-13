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

# Get the operating system
# Possible values are:
#		* linux for linux
#		* darwin for macOS
#
# usage:
# if [ "${OS}" == "darwin" ]; then
#   # do macos stuff
# fi
OS="$(echo $(uname) | tr '[:upper:]' '[:lower:]')"

# Set the container runtime to docker or nerdctl.
if [ -z "${CONTAINER_RUNTIME:-}" ]; then
  if command -v docker &> /dev/null; then
    CONTAINER_RUNTIME=docker
  elif command -v nerdctl &> /dev/null; then
    CONTAINER_RUNTIME=nerdctl
  else
    CONTAINER_RUNTIME=docker
  fi
fi

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
  TEST_CLASS="${TEST_CLASS:-Kubermatic}"
  cat << EOF > ${ARTIFACTS}/junit.$(echo $TEST_NAME | sed 's/ /_/g' | tr '[:upper:]' '[:lower:]').xml
<?xml version="1.0" ?>
<testsuites>
  <testsuite errors="$errors" failures="$errors" name="$TEST_CLASS" tests="1">
    <testcase classname="$TEST_CLASS" name="$TEST_NAME" time="$duration">
      $failure
    </testcase>
  </testsuite>
</testsuites>
EOF
}

is_containerized() {
  # we're inside a Kubernetes pod/container or inside a container launched by containerize()
  [ -n "${KUBERNETES_SERVICE_HOST:-}" ] || [ -n "${CONTAINERIZED:-}" ]
}

containerize() {
  local cmd="$1"
  local image="${CONTAINERIZE_IMAGE:-quay.io/kubermatic/util:2.5.0}"
  local gocache="${CONTAINERIZE_GOCACHE:-/tmp/.gocache}"
  local gomodcache="${CONTAINERIZE_GOMODCACHE:-/tmp/.gomodcache}"
  local skip="${NO_CONTAINERIZE:-}"

  # short-circuit containerize when in some cases it needs to be avoided
  [ -n "$skip" ] && return

  if ! is_containerized; then
    echodate "Running $cmd in a Docker container using $image..."
    mkdir -p "$gocache"
    mkdir -p "$gomodcache"

    exec $CONTAINER_RUNTIME run \
      -v "$PWD":/go/src/k8c.io/kubermatic \
      -v "$gocache":"$gocache" \
      -v "$gomodcache":"$gomodcache" \
      -w /go/src/k8c.io/kubermatic \
      -e "GOCACHE=$gocache" \
      -e "GOMODCACHE=$gomodcache" \
      -e "CONTAINERIZED=true" \
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
    github_rsa_key="github.com ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCj7ndNxQowgcQnjshcLrqPEiiphnt+VTTvDP6mHBL9j1aNUkY4Ue1gvwnGLVlOhGeYrnZaMgRK6+PKCUXaDbC7qtbW8gIkhL7aGCsOr/C56SJMy/BCZfxd1nWzAOxSDPgVsmerOBYfNqltV9/hWCqBywINIR+5dIg6JTJ72pcEpEjcYgXkE2YEFXV1JHnsKgbLWNlhScqb2UmyRkQyytRLtL+38TGxkxCflmO+5Z8CSSNY7GidjMIZ7Q4zMjA2n1nGrlTDkzwDCsw+wqFPGQA179cnfGWOWRVruj16z6XyvxvjJwbz0wQZ75XK5tKSb7FNyeIEs4TT4jk+S4dhPeAUC5y+bDYirYgM4GC7uEnztnZyaVWQ7B381AK4Qdrwt51ZqExKbQpTUNn+EjqoTwvqNj4kqx5QUCI0ThS/YkOxJCXmPUWZbhjpCg56i+2aB6CmK2JGhn57K5mj0MNdBXA4/WnwH6XoPWJzK5Nyu2zB3nAZp+S5hpQs+p1vN1/wsjk="

    echo " [*] Adding Github's SSH host key to known hosts" > /dev/stderr
    mkdir -p "$HOME/.ssh"
    chmod 700 "$HOME/.ssh"
    echo "$github_rsa_key" >> "$HOME/.ssh/known_hosts"
    chmod 600 "$HOME/.ssh/known_hosts"
  fi
}

vault_ci_login() {
  # already logged in
  if [ -n "${VAULT_TOKEN:-}" ]; then
    return 0
  fi

  # check environment variables
  if [ -z "${VAULT_ROLE_ID:-}" ] || [ -z "${VAULT_SECRET_ID:-}" ]; then
    echo "VAULT_ROLE_ID and VAULT_SECRET_ID must be set to programmatically authenticate against Vault."
    return 1
  fi

  local token
  token=$(vault write --format=json auth/approle/login "role_id=$VAULT_ROLE_ID" "secret_id=$VAULT_SECRET_ID" | jq -r '.auth.client_token')

  export VAULT_TOKEN="$token"
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
  local instance="${PROW_JOB_ID:-}"
  local prowjob="${JOB_NAME:-}"

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

check_seed_ready() {
  status="$(kubectl --namespace "$1" get seed "$2" --output json | jq -r '.status.conditions.ResourcesReconciled.status')"
  if [ "$status" != "True" ]; then
    echodate "Seed does not yet have ResourcesReconciled=True condition."
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

docker_logs() {
  if [[ $? -ne 0 ]]; then
    echodate "Printing Docker logs"
    cat /tmp/docker.log
    echodate "Done printing Docker logs"
  fi
}

start_docker_daemon_ci() {
  # DOCKER_REGISTRY_MIRROR_ADDR is injected via Prow preset;
  # start-docker.sh is part of the build image.
  DOCKER_REGISTRY_MIRROR="${DOCKER_REGISTRY_MIRROR_ADDR:-}" DOCKER_MTU=1400 start-docker.sh
}

start_docker_daemon() {
  if docker stats --no-stream > /dev/null 2>&1; then
    echodate "Not starting Docker again, it's already running."
    return
  fi

  # Start Docker daemon
  echodate "Starting Docker"
  dockerd > /tmp/docker.log 2>&1 &

  echodate "Started Docker successfully"
  appendTrap docker_logs EXIT

  # Wait for Docker to start
  echodate "Waiting for Docker"
  retry 5 docker stats --no-stream
  echodate "Docker became ready"
}

repeat() {
  local end=$1
  local str="${2:-=}"

  for i in $(seq 1 $end); do
    echo -n "${str}"
  done
}

heading() {
  local title="$@"
  echo "$title"
  repeat ${#title} "="
  echo
}

# This is used during releases to set the correct version on all Helm charts.
set_helm_charts_version() {
  local version="$1"
  local dockerTag="${2:-$version}"

  # trim leading v; Helm allows "v1.2.3" as chart versions, but some other tools
  # consuming charts are more strict and require pure, prefixless semvers.
  local semver="${version#v}"

  echodate "Setting Helm chart version to $semver..."

  while IFS= read -r -d '' chartFile; do
    chart="$(basename $(dirname "$chartFile"))"
    if [ "$chart" = "mla-secrets" ]; then
      continue
    fi

    # download all charts dependencies
    chartRepoURL=$(yq eval '.dependencies[0].repository' $chartFile)
    if [ "$chartRepoURL" != "null" ]; then
      chartDepName=$(yq eval '.dependencies[0].name' $chartFile)
      # Skip OCI repositories as they don't need to be added to helm repos
      if [[ "$chartRepoURL" == oci://* ]]; then
        echodate "Skipping OCI repository: $chartRepoURL"
      else
        helm repo add $chartDepName $chartRepoURL
      fi

      chartDirParent=$(dirname "$chartFile")
      helm dependency build $chartDirParent --skip-refresh
    fi

    yq --inplace ".version = \"$semver\"" "$chartFile"
    if [ "$chart" = "kubermatic-operator" ]; then
      yq --inplace ".appVersion = \"$version\"" "$chartFile"
      yq --inplace ".kubermaticOperator.image.tag = \"$dockerTag\"" "$(dirname "$chartFile")/values.yaml"
    fi
  done < <(find charts -name 'Chart.yaml' -print0 | sort --zero-terminated)
}

# copy_crds_to_chart is used during GitHub releases and for e2e tests,
# it ensures that the auto-generated CRDs in pkg/ are copied into the
# operator chart.
copy_crds_to_chart() {
  chartCRDs=charts/kubermatic-operator/crd/k8c.io
  sourceCRDs=pkg/crd/k8c.io

  mkdir -p $chartCRDs
  cp $sourceCRDs/* $chartCRDs
}

# set_crds_version_annotation will inject a annotation label into all YAML files
# of the given directory. This is important for GitOps installations that
# do not use the KKP installer, so that the CRDs are still properly annotated,
# as the KKP operator needs to determine the CRD versions.
set_crds_version_annotation() {
  local version="${1:-}"
  local directory="${2:-charts/kubermatic-operator/crd/k8c.io}"

  if [ -z "$version" ]; then
    version="$(git describe)"
  fi

  while IFS= read -r -d '' filename; do
    yq --inplace ".metadata.annotations.\"app.kubernetes.io/version\" = \"$version\"" "$filename"
  done < <(find "$directory" -name '*.yaml' -print0 | sort --zero-terminated)
}

# go_test wraps running `go test` commands. The first argument needs to be file name
# for a junit result file that will be generated if go-junit-report is present and
# $ARTIFACTS is set. The remaining arguments are passed to `go test`.
go_test() {
  local junit_name="${1:-}"
  shift

  # only run go-junit-report if binary is present and we're in CI / the ARTIFACTS environment is set
  if [ -x "$(command -v go-junit-report)" ] && [ ! -z "${ARTIFACTS:-}" ]; then
    go test "$@" 2>&1 | go-junit-report -set-exit-code -iocopy -out ${ARTIFACTS}/junit.${junit_name}.xml
  else
    go test "$@"
  fi
}

# safebase64 ensures the given value is base64-encoded.
# If the given value is already encoded, it will be echoed
# unchanged.
safebase64() {
  local value="$1"

  set +e
  decoded="$(echo "$value" | base64 -d 2> /dev/null)"
  if [ $? -eq 0 ]; then
    echo "$value"
    return 0
  fi

  echo "$value" | base64 -w0
  echo
}

pr_has_label() {
  if [ -z "${REPO_OWNER:-}" ] || [ -z "${REPO_NAME:-}" ] || [ -z "${PULL_NUMBER:-}" ]; then
    echo "PR check only works on CI."
    return 1
  fi

  matched=$(curl \
    --header "Accept: application/vnd.github+json" \
    --silent \
    --fail \
    https://api.github.com/repos/$REPO_OWNER/$REPO_NAME/pulls/$PULL_NUMBER |
    jq --arg labelName "$1" '.labels[] | select(.name == $labelName)')

  [ -n "$matched" ]
}

provider_disabled() {
  # e.g. "VSPHERE_E2E_DISABLED"
  local disableEnv="${1^^}_E2E_DISABLED"
  local labelName="test/require-$1"
  local branchName="${2:-}"

  # Tests can be globally disabled by having a special environment
  # variable injected via the Prow preset; if they are not disabled,
  # we are done here.

  if [ -z "${!disableEnv:-}" ]; then
    return 1
  fi

  # To prevent breaking existing releases, this check does apply to
  # release branches. On those every failed test needs to be manually
  # overridden.

  if [ -z "$branchName" ]; then
    branchName="${PULL_BASE_REF:-main}"
  fi

  if [[ "$branchName" =~ release/.* ]]; then
    echodate "\$$disableEnv is set, but in a release branch ($branchName) this has no effect."
    return 1
  fi

  # Even if tests are disabled, they can be forcefully re-enabled
  # (e.g. if provider X is disabled for all tests until a certain
  # pull requests fixes some underlying issue and for that certain
  # PR we want to run the tests regardless).
  # Importantly, one cannot use labels to _disable_ any tests, only
  # _re-enable_ them.

  if pr_has_label "$labelName"; then
    echodate "\$$disableEnv is set, but PR has $labelName label, so tests will not be disabled."
    return 1
  fi

  echodate "\$$disableEnv is set, tests will be disabled. Apply the label $labelName to this PR to forcefully enable them."
  return 0
}

# This is an alias for curl, but will in the CI system rewrite the
# request to allow it to pass through an in-cluster caching proxy.
# The proxy is not a regular proxy (due to TLS limitations), but a
# simply plain-HTTP reverse proxy, so a request to GitHub would
# go plain HTTP between this script and the proxy, and HTTPS between
# the proxy and GitHub.
download_archive() {
  local url="$1"
  shift

  domain="$(echo "$url" | sed -E 's#^https?://([^/]+)/.*#\1#')"
  proxiedDomains="github.com,codeberg.org,dl.k8s.io"

  if [[ -z "${PROW_JOB_ID:-}" ]] || [[ -z "${DOWNLOAD_CACHE_HOST:-}" ]] || ! echo "$proxiedDomains" | grep -w -q "$domain"; then
    # do nothing special when running outside of the CI environment
    # or when using a domain we do not proxy internally
    curl "$url" "$@"
  else
    # determine target domain
    echodate "Note: Proxying request to $domain through download proxy." >&2
    url="$(echo "$url" | sed -E "s#https?://$domain/#http://$DOWNLOAD_CACHE_HOST/#")"
    curl --header "Host: $domain" "$url" "$@"
  fi
}
