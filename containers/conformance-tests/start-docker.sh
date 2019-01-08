#!/usr/bin/env bash

set -euo pipefail

# Stolen from https://github.com/kubernetes/test-infra/blob/1be7a22cd0187dfa061feef4a71e958fd46306d6/images/bootstrap/runner.sh#L67
echo "Docker in Docker enabled, initializing..."
printf '=%.0s' {1..80}; echo
# If we have opted in to docker in docker, start the docker daemon,
service docker start
# the service can be started but the docker socket not ready, wait for ready
WAIT_N=0
MAX_WAIT=5
while true; do
    # docker ps -q should only work if the daemon is ready
    docker ps -q > /dev/null 2>&1 && break
    if [[ ${WAIT_N} -lt ${MAX_WAIT} ]]; then
        WAIT_N=$((WAIT_N+1))
        echo "Waiting for docker to be ready, sleeping for ${WAIT_N} seconds."
        sleep ${WAIT_N}
    else
        echo "Reached maximum attempts, not waiting any longer..."
        exit 1
    fi
done
printf '=%.0s' {1..80}; echo
echo "Done setting up docker in docker."
