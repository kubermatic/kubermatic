#!/usr/bin/env sh

set -e

function cleanup {
  OLD_EXIT_CODE=$?
  docker rm -f $CONTAINER_NAME >/dev/null
  rm -rf ./addons-from-container
  exit $OLD_EXIT_CODE
}
trap cleanup EXIT

cd $(dirname $0)/../..

IMAGE=$(cat config/kubermatic/values.yaml |grep 'addons:' -A3|grep repository|awk '{ print $2 }'|tr -d '"')
TAG=$(cat config/kubermatic/values.yaml |grep 'addons:' -A3|grep tag|awk '{ print $2 }'|tr -d '"')

CONTAINER_NAME=$(docker create $IMAGE:$TAG)

docker cp $CONTAINER_NAME:/addons ./addons-from-container

for ignored_file in $(cat addons/.dockerignore); do
  cp addons/$ignored_file ./addons-from-container/
done

diff --brief -r ./addons ./addons-from-container
