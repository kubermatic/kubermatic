#!/usr/bin/env sh

cd $(dirname $0)

# Install yq if not installed
if ! [ -x "$(command -v yq)" ]; then
	echo "yq not installed / vailable in PATH!"
	echo "Executing go get on github.com/mikefarah/yq ..."
	go get github.com/mikefarah/yq
	echo "Done!"
fi

comment="# This file has been generated, do not edit."

for file in */*.yaml; do
  newfile=$(dirname $file)-$(basename $file)
  echo "$file => $newfile"
  yq r $file -j | jq 'del(.groups[].rules[].runbook)' | (echo "$comment"; yq r -) > ../$newfile
done
