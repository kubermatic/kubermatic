#!/usr/bin/env bash
set -xeuo pipefail

go build github.com/kubermatic/kubermatic/api/cmd/cleanup/aws

./aws \
  -aws-access-key-id="$(vault kv get -field=accessKeyID dev/e2e-aws)" \
  -aws-secret-access-key="$(vault kv get -field=secretAccessKey dev/e2e-aws)"
