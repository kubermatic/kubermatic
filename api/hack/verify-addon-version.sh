#!/usr/bin/env sh

set -e

function finish {
  OLD_EXIT_CODE=$?
  docker rm -f $CONTAINER_NAME >/dev/null
  rm -rf ./addons-from-container
  exit $OLD_EXIT_CODE
}
trap finish EXIT

cd $(dirname $0)/../..

IMAGE=$(cat config/kubermatic/values.yaml |grep 'addons:' -A3|grep repository|awk '{ print $2 }'|tr -d '"')
TAG=$(cat config/kubermatic/values.yaml |grep 'addons:' -A3|grep tag|awk '{ print $2 }'|tr -d '"')

CONTAINER_NAME=$(docker create $IMAGE:$TAG)

docker cp $CONTAINER_NAME:/addons ./addons-from-container

cp addons/.dockerignore ./addons-from-container
cp addons/Dockerfile ./addons-from-container
cp addons/README.md ./addons-from-container

diff --brief -r ./addons ./addons-from-container
