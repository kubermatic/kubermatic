#!/usr/bin/env bash

set -euo pipefail

success_images=()
failed_images=()

if [ "$#" -lt 1 ] || [ "${1}" == "--help" ]; then
  cat << EOF
Usage:
  $(basename $0) -i image1,image2
  $(basename $0) -f fs1,fs2
Example:
  $(basename $0) -i quay.io/kubermatic/kubermatic:v2.21.1,quay.io/kubermatic/kubermatic:v2.21.0
  $(basename $0) -f https://github.com/kubermatic/kubeone
EOF
  exit 0
fi

function image_scan() {
  for i in "${image_array[@]}"; do
    EXIT_CODE=0
    echo -e "Starting Vurnability scanning for ${i}"
    trivy image --exit-code 1 --no-progress ${i} || EXIT_CODE=1
    if [[ EXIT_CODE -eq 1 ]]; then
      failed_images[${#failed_images[@]}]=${i}
    else
      success_images[${#failed_images[@]}]=${i}
    fi
  done
  if [ "${#failed_images[@]}" -gt 0 ]; then
    echo -e "Vurnabilities found on the images ${failed_images[@]}"
  else
    echo -e "No vurnabilitiies found on the images ${success_images[@]}"
  fi
}

while getopts ":i:f:" flag
do
  case "$flag" in
    i) set -f
       IFS=,
       image_array=($OPTARG)
       image_scan;;
    f) set -f
       IFS=,
       fs_array=($OPTARG) ;;
    :) echo "argument missing" ;;
    \?)echo "You have to use: [-i] or [-f]" ;;
  esac
done
