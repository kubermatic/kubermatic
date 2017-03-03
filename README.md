# Kubermatic API

## Development environment

Due to the big dependency tree derived from Kubernetes it is strongly recommended to set up a separate `GOPATH` environment for Kubermatic:

```bash
$ mkdir $HOME/src/kubermatic
$ cd $HOME/src/kubermatic
$ export GOPATH=$PWD
$ mkdir -p bin pkg src
$ cd src/kubermatic
$ git clone git@github.com:kubermatic/api
$ git clone git@github.com:kubermatic/config
$ cd api
$ echo 'dummy: dummy' > secrets.yaml

mkdir -p template/coreos &&
pushd template/coreos &&
ln -s ../../../config/api/static/nodes/digitalocean/template/coreos/arptables.service arptables.service &&
ln -s ../../../config/api/static/nodes/aws/template/coreos/cloud-config-node.yaml aws-cloud-config-node.yaml &&
ln -s ../../../config/api/static/nodes/digitalocean/template/coreos/cloud-config-node.yaml do-cloud-config-node.yaml &&
ln -s ../../../config/api/static/nodes/digitalocean/template/coreos/floating-ip.service floating-ip.service &&
popd
```

## Dependencies

### Install dependencies

```bash
glide install --strip-vendor
```

### Update dependencies

```bash
glide update --strip-vendor
```

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
  --kubeconfig=.kubeconfig --v=7 --dev
```


# Misc

## Upload to S3

```
s3cmd put -P --multipart-chunk-size-mb=1 etcd2-proxy-proxy s3://kubermatic/coreos/etcd2-proxy-proxy
```
