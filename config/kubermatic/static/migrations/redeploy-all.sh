#!/usr/bin/env bash
echo "==================================="
echo "Redeploying all cluster resources"
echo "==================================="
echo
echo "Loading clusters..."
CLUSTERS=$(kubectl get cluster -o=custom-columns=:.metadata.name)
for CLUSTER in $CLUSTERS; do
    echo "Deleting deployments from $CLUSTER..."
    echo
    kubectl -n cluster-$CLUSTER delete deployment apiserver
    kubectl -n cluster-$CLUSTER delete deployment addon-manager
    kubectl -n cluster-$CLUSTER delete deployment controller-manager
    kubectl -n cluster-$CLUSTER delete deployment node-controller
    kubectl -n cluster-$CLUSTER delete deployment scheduler
    sleep 30

    echo "Set phase to pending for cluster $CLUSTER..."
    kubectl patch cluster $CLUSTER --type json --patch='[{"op": "replace", "path": "/status/phase", "value":"Pending"}]'

    phase=$(kubectl get cluster $CLUSTER -o jsonpath='{.status.phase}')
    while [ "$phase" != "Running" ]
    do
        echo "Waiting until cluster is in phase running again..."
        sleep 10
         phase=$(kubectl get cluster $CLUSTER -o jsonpath='{.status.phase}')
    done
    echo "Redeploying cluster $CLUSTER finished"
    echo
done
