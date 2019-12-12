#!/usr/bin/env bash
TARGET_REGISTRY=${TARGET_REGISTRY:-127.0.0.1:5000}

function retag {
  local IMAGE=$1

  ORG="$(echo ${IMAGE} | cut -d / -f2)"
  NAME="$(echo ${IMAGE} | cut -d / -f3 | cut -d : -f1)"
  TAG="$(echo ${IMAGE} | cut -d / -f3 | cut -d : -f2)"
  TARGET_IMAGE="${TARGET_REGISTRY}/${ORG}/${NAME}:${TAG}"
  echo "Retagging ${IMAGE} as ${TARGET_IMAGE}"

  if curl -Ss --fail "http://${TARGET_REGISTRY}/v2/${ORG}/${NAME}/tags/list" | jq -e ".tags | index(\"${TAG}\")" >/dev/null; then
    echo "Skipping image ${TARGET_IMAGE} because it already exists in the target registry"
    return
  fi

  docker pull ${IMAGE}
  docker tag ${IMAGE} ${TARGET_IMAGE}
  docker push ${TARGET_IMAGE}
}

IMAGES=$(cat /dev/stdin | grep "image: " | cut -d : -f 2,3)
for IMAGE in ${IMAGES}
do
  # Make sure we strip all quotes
  IMAGE="${IMAGE%\'}"
  IMAGE="${IMAGE#\'}"
  IMAGE="${IMAGE%\"}"
  IMAGE="${IMAGE#\"}"
  retag ${IMAGE}
done
