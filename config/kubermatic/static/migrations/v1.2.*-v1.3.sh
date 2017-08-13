#!/usr/bin/env bash
echo "==================================="
echo "Kubermatic v1.2 -> v1.3"
echo "==================================="
echo "After this change, every old kubeconfig will not work anymore."
echo "Make sure everyone fetches a new one!"
echo "Also: Nodes need to be recreated!"
echo
echo "Loading clusters..."
CLUSTERS=$(kubectl get ns -o=custom-columns=:.metadata.name | egrep ^cluster-[a-z0-9]{9}$)
echo "Migration customer clusters..."
for CLUSTER in $CLUSTERS; do
    echo "Deleting deprecated resources from $CLUSTER..."
    echo
    kubectl -n $CLUSTER delete deployment apiserver
    kubectl -n $CLUSTER delete deployment controller-manager
    kubectl -n $CLUSTER delete deployment scheduler
    kubectl -n $CLUSTER delete service apiserver
    kubectl -n $CLUSTER delete ingress k8sniff
    echo

    echo "Update annotations & labels for $CLUSTER..."
    echo
    kubectl annotate namespace $CLUSTER kubermatic.io/url-
    kubectl annotate --overwrite namespace $CLUSTER kubermatic.io/phase-ts=$(date +'%Y-%m-%dT%H:%M:%S')
    kubectl label namespace $CLUSTER node-port-
    echo
    echo "Set phase to pending for $CLUSTER..."
    echo
    kubectl label --overwrite namespace $CLUSTER phase=pending
    phase=$(kubectl get ns $CLUSTER -o jsonpath='{.metadata.labels.phase}')
    while [ "$phase" != "running" ]
    do
        echo "Waiting until cluster is in phase running again..."
        sleep 10
         phase=$(kubectl get ns $CLUSTER -o jsonpath='{.metadata.labels.phase}')
    done
    echo "Migration for $CLUSTER finished"
    echo
done
