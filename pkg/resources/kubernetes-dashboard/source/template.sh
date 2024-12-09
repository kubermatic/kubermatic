#!/usr/bin/env bash

helm repo add kubernetes-dashboard https://kubernetes.github.io/dashboard/
helm repo update

helm template \
  --values values.yaml \
  --version 7.10.0 \
  --namespace cluster-xyz \
  kubernetes-dashboard kubernetes-dashboard/kubernetes-dashboard > rendered.yaml
