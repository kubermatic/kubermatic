#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

go generate pkg/controller/openshift/resources/const.go
