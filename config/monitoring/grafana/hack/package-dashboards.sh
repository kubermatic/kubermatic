#!/usr/bin/env bash

set -euo pipefail

cd $(dirname $0)/../dashboards
tar vcjf ../dashboards.tar.bz2 */*.json
