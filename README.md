# Kubermatic API

## Development environment

Due to the big dependency tree derived from Kubernetes it is strongly recommended to set up a separate `GOPATH` environment for Kubermatic:

```
$ mkdir $HOME/src/kubermatic
$ cd $HOME/src/kubermatic
$ export GOPATH=$PWD
$ mkdir -p bin pkg src
$ cd src/kubermatic
$ git clone git@github.com:kubermatic/api
$ git clone git@github.com:kubermatic/kubermatic
$ cd api
$ ln -s ../kubermatic/api/datacenter-secrets.yaml secrets.yaml
$ cd template
$ ln -s ../../kubermatic/nodes/digitalocean/template/coreos/ coreos
```

## Dependencies

How to update the Glide/vendor dependencies with a new Kubernetes version:

Update the vendored sources in Kubermatic:
```
$ ./glide-update.sh
```

Finally commit the vendored changes.

## Building locally

In order to use incremental compilation one can compile a binary as follows:
```
$ make GOBUILD="go install" kubermatic-api
```
Replace `kubermatic-api` with `kubermatic-cluster-controller` respectively depending on what you want to build.

Example for `kubermatic-api`

```
make build CMD=kubermatic-api && ./kubermatic-api --v=7 \
 --jwt-key="RE93Ef1Yt5-mrp2asikmfalfmcRaaa27gpH8hTAlby48LQQbUbn9d4F7yh01g_cc" \
--datacenters=datacenters.yaml --kubeconfig .kubeconfig --logtostderr
```

and `kubermatic-cluster-controller`

```
make build CMD=kubermatic-cluster-controller &&  \
 ./kubermatic-cluster-controller \
 -master-resources ../kubermatic/master \
  --kubeconfig=.kubeconfig --v=7 --dev --loglevel=4
```


# Misc

## Upload to S3

```
s3cmd put -P --multipart-chunk-size-mb=1 etcd2-proxy-proxy s3://kubermatic/coreos/etcd2-proxy-proxy
```
