# Kubermatic API

## Development environment

Due to the big dependency tree derived from Kubernetes it is strongly recommended to set up a separate `GOPATH` environment for Kubermatic:

```
$ export GO15VENDOREXPERIMENT=1
$ mkdir $HOME/src/kubermatic
$ cd $HOME/src/kubermatic
$ mkdir -p bin pkg src
$ cd src/kubermatic
$ git clone git@github.com:kubermatic/api
$ cd api
$ godep restore
```

## Dependencies

How to update the Godep/vendor dependencies with a new Kubernetes version:

Check out the target Kubernetes version:
```
$ export GO15VENDOREXPERIMENT=1
$ cd $HOME/src/kubermatic/src/k8s.io/kubernetes
$ git fetch
$ git checkout <GIT_REF> # i.e. v1.1.7
```

Restore the Kubermatic `GOPATH` with the dependencies from Kubernetes:
```
$ godep restore
```

Update the vendored sources in Kubermatic:
```
$ cd $HOME/src/kubermatic/src/kubermatic/api
$ GOOS=linux go get .
$ godep save -v ./... github.com/docker/libcontainer/cgroups/fs github.com/docker/libcontainer/configs
```

Finally commit the vendored changes.

## Building locally

In order to use incremental compilation one can compile a binary as follows:
```
$ make GOBUILD="go install" kubermatic-api
```
Replace `kubermatic-api` with `kubermatic-cluster-controller` respectively depending on what you want to build.

# Misc

## Upload to S3

```
s3cmd put -P --multipart-chunk-size-mb=1 etcd2-proxy-proxy s3://kubermatic/coreos/etcd2-proxy-proxy
```
