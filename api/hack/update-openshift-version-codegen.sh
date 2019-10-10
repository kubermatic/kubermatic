#!/bin/bash

# This file can be used to update the generated image names for Openshift.
# The desired versions msut be configured first in
# codegen/openshift_versions/main.go and a const for each version must be
# added to pkg/controller/openshift/resources/const.go
#
# Also, executing this script requires access to the ocp quay repo.

set -o errexit
set -o nounset
set -o pipefail

go generate pkg/controller/openshift/resources/const.go
