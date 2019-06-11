#!/usr/bin/env bash

set -euo pipefail

TAG=v2.2.0-dev2
export TAG
make docker
