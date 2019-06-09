#!/usr/bin/env bash

set -euo pipefail

TAG=v2.1.1
export TAG
make docker
