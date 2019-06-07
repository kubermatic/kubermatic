#!/usr/bin/env bash

set -euo pipefail

TAG=v2.1.0
export TAG
make docker
