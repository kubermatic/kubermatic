#!/usr/bin/env bash

if [ "$#" -lt 1 ] || [ "${1}" == "--help" ]; then
  cat <<EOF
Usage: $(basename $0) <cluster-id>

  <cluster-id>                 id of the cluster (for example: hfvt4dkgb)
EOF
  exit 0
fi

echo "Resetting admin token..."
kubectl patch cluster $1 --type json --patch='[{"op": "remove", "path": "/address/adminToken"}]'
echo "Deleting apiserver deployment"
kubectl -n cluster-$1 delete deployment apiserver
echo "Deleting token-users secret"
kubectl -n cluster-$1 delete secret token-users
echo "Waiting 30 sec until the apiserver deployment and the secret are gone"
sleep 30
echo "Triggering new deployment"
kubectl patch cluster $1 --type json --patch='[{"op": "replace", "path": "/status/phase", "value":"Pending"}]'
sleep 30
