#!/usr/bin/env sh

set -ex

cd $(dirname $0)/../..

IMAGE=$(cat config/kubermatic/values.yaml |grep 'addons:' -A3|grep repository|awk '{ print $2 }'|tr -d '"')
TAG=$(cat config/kubermatic/values.yaml |grep 'addons:' -A3|grep tag|awk '{ print $2 }'|tr -d '"')

docker run --rm \
  -v $PWD/addons:/tmp/addons-from-repo \
    $IMAGE:$TAG \
      /bin/sh -c "cd /tmp/addons-from-repo && cp .dockerignore Dockerfile README.md /addons/ && diff --brief -r /addons /tmp/addons-from-repo"
