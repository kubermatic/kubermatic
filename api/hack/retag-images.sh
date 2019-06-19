#!/usr/bin/env bash
TARGET_REGISTRY=${TARGET_REGISTRY:-127.0.0.1:5000}

function retag {
  local IMAGE=$1

  TARGET_IMAGE="${TARGET_REGISTRY}/$(echo ${IMAGE} | cut -d / -f 2-)"
  echo "Retagging ${IMAGE} to ${TARGET_IMAGE}"
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
