# Kubermatic API

## Dependencies

How to update the Godep/vendor dependencies with a new Kubernetes version:

- checkout the right Kubernetes version into `src/k8s.io/kubernetes`, e.g. v1.1.7
- in `src/k8s.io/kubernetes` call `godeps restore`
- in `src/github.com/kubermatic/api` call `GOOS=linux go get .`
- in `src/github.com/kubermatic/api` call `GOOS=linux GO15VENDOREXPERIMENT=1 godep save -v ./... github.com/docker/libcontainer/cgroups/fs github.com/docker/libcontainer/configs`
- commit`
