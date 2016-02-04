# Kubermatic API

## Dependencies

How to update the Godep/vendor dependencies with a new Kubernetes version:

- delete `$GOPATH` with the exception of kubermatic
- delete `src/github.com/kubermatic/api/vendor`
- checkout the right Kubernetes version into `src/k8s.io/kubernetes`, e.g. v1.1.7
- in `src/k8s.io/kubernetes` call `godeps restore`
- in `src/github.com/kubermatic/api` call `go get .`
- in `src/github.com/kubermatic/api` call `GO15VENDOREXPERIMENT=1 godep save -v ./... github.com/docker/libcontainer/cgroups/fs github.com/docker/libcontainer/configs`
- commit`
