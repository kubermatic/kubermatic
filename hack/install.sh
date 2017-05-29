#!/usr/bin/env bash
dirs=$(find . -name glide.yaml | grep -v vendor | xargs -I{} dirname {})
for d in ${dirs}
do
	pushd "${d}"
	glide i -v
	popd
done
