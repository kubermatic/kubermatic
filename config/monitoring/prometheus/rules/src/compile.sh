#!/usr/bin/env sh

cd $(dirname $0)

comment="# This file has been generated, do not edit."

for file in */*.yaml; do
  newfile=$(dirname $file)-$(basename $file)
  echo "$file => $newfile"
  yq r $file -j | jq 'del(.groups[].rules[].runbook)' | (echo "$comment"; yq r -) > ../$newfile
done
