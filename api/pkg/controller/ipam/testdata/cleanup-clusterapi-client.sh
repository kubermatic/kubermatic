#!/usr/bin/env bash

set -euxo pipefail

cd $(dirname $0)/../../../..
git checkout HEAD vendor/sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset/fake/*
