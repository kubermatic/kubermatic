#!/usr/bin/env sh

set -e

cleanup() {
  set +e
  docker rm -f $KUBERNETES_CONTAINER_NAME >/dev/null
  docker rm -f $OPENSHIFT_CONTAINER_NAME >/dev/null
  rm -rf ./addons-from-container*
}
trap cleanup EXIT

cd $(dirname $0)/../..

KUBERNETES_IMAGE=quay.io/kubermatic/addons
KUBERNETES_TAG="$(cat config/kubermatic/values.yaml |grep $KUBERNETES_IMAGE -A2|grep tag|awk '{ print $2 }'|tr -d '"')"
OPENSHIFT_IMAGE=quay.io/kubermatic/openshift-addons
OPENSHIFT_TAG="$(cat config/kubermatic/values.yaml |grep $OPENSHIFT_IMAGE -A2|grep tag|awk '{ print $2 }'|tr -d '"')"

KUBERNETES_CONTAINER_NAME=$(docker create $KUBERNETES_IMAGE:$KUBERNETES_TAG)
OPENSHIFT_CONTAINER_NAME=$(docker create $OPENSHIFT_IMAGE:$OPENSHIFT_TAG)

docker cp $KUBERNETES_CONTAINER_NAME:/addons ./addons-from-container-kubernetes
docker cp $OPENSHIFT_CONTAINER_NAME:/addons ./addons-from-container-openshift

for ignored_file in $(cat addons/.dockerignore); do
  cp addons/$ignored_file ./addons-from-container-kubernetes/
done
for ignored_file in $(cat addons/.dockerignore); do
  cp openshift_addons/$ignored_file ./addons-from-container-openshift/
done

diff --brief -r ./addons ./addons-from-container-kubernetes
diff --brief -r ./openshift_addons ./addons-from-container-openshift
