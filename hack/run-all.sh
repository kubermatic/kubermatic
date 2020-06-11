#!/bin/bash

set -e -u

go get github.com/DarthSim/hivemind

cd $(go env GOPATH)/src/github.com/kubermatic/kubermatic

hivemind
