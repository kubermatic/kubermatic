#!/usr/bin/env bash

set -euo pipefail

dqreplacement='"{{ Image \1 }}"'
sqreplacement=$'\'{{ Image "\1" }}\''
replacement='{{ Image "\1" }}'

for registry in docker.io quay.io public.ecr.aws; do
  sed -i "s/\(\"$registry.*\"\)/$dqreplacement/g" $@
  sed -i "s/\('$registry.*'\)/$sqreplacement/g" $@

  # the space makes it so that this does not match the
  # output of the previous two sed invocations
  sed -i "s/ \($registry.*\)/ $replacement/g" $@
done
