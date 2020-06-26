#!/usr/bin/env sh

# Copyright 2020 The Kubermatic Kubernetes Platform contributors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

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
