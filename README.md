# Kubermatic API

## Development environment

Due to the big dependency tree derived from Kubernetes it is strongly recommended to set up a separate `GOPATH` environment for Kubermatic:

```bash
mkdir $HOME/src/kubermatic
cd $HOME/src/kubermatic
export GOPATH=$PWD
mkdir -p bin pkg src
cd src/kubermatic
git clone git@github.com:kubermatic/api
git clone git@github.com:kubermatic/config
git clone git@github.com:kubermatic/secrets
cd api

mkdir -p template/coreos &&
pushd template/coreos &&
ln -s $GOPATH/src/github.com/kubermatic/config/kubermatic/static/nodes/coreos/cloud-init.yaml cloud-init.yaml &&
popd
```

Or you can use regular `GOPATH`

```bash
mkdir -p $GOPATH/src/github.com/kubermatic
cd $GOPATH/src/github.com/kubermatic
git clone git@github.com:kubermatic/api
git clone git@github.com:kubermatic/config
git clone git@github.com:kubermatic/secrets
cd api
mkdir -p template/coreos &&
pushd template/coreos &&
ln -s $GOPATH/src/github.com/kubermatic/config/kubermatic/static/nodes/coreos/cloud-init.yaml cloud-init.yaml &&
popd
```

### Dependencies

#### Install dependencies

```bash
glide install --strip-vendor
```

#### Update dependencies

```bash
glide update --strip-vendor
```

### Building locally

In order to use incremental compilation one can compile a binary as follows:
```bash
make GOBUILD="go install" build
```

### Running locally
#### kubermatic-api

```bash
./kubermatic-api \                                                                          
  --worker-name="unique-label-abcdef123" \
  --kubeconfig=$GOPATH/src/github.com/kubermatic/secrets/seed-clusters/dev.kubermatic.io/kubeconfig \
  --datacenters=$GOPATH/src/github.com/kubermatic/secrets/seed-clusters/dev.kubermatic.io/datacenters.yaml \
  --logtostderr \
  --v=8 \
  --token-issuer=https://kubermatic.eu.auth0.com/ \
  --client-id=xHLUljMUUEFP95wmlODWexe1rvOXuyTT \
  --address=127.0.0.1:8080 \                 
  --master-kubeconfig=$GOPATH/src/github.com/kubermatic/secrets/seed-clusters/dev.kubermatic.io/kubeconfig
```

#### kubermatic-cluster-controller
```bash
./kubermatic-cluster-controller \
  --datacenters=$GOPATH/src/github.com/kubermatic/secrets/seed-clusters/dev.kubermatic.io/datacenters.yaml \
  --kubeconfig=$GOPATH/src/github.com/kubermatic/secrets/seed-clusters/dev.kubermatic.io/kubeconfig \
  --worker-name="unique-label-abcdef123" \
  --logtostderr=1 \
  --master-resources=$GOPATH/src/github.com/kubermatic/config/kubermatic/static/master \
  --v=4 \
  --external-url=dev.kubermatic.io
```

Valid worker-name label value must be 63 characters or less and must be empty or begin and end with an alphanumeric character ([a-z0-9A-Z]) with dashes (-), underscores (_), dots (.), and alphanumerics between.
The dev label should be also unique between a pair of api<->controller.

## Linting / Testing
### Install linters
```bash
go get -u github.com/golang/lint/golint
go get -u github.com/client9/misspell/cmd/misspell
go get -u github.com/kisielk/errcheck
go get -u github.com/Masterminds/glide
go get -u github.com/opennota/check/cmd/varcheck
go get -u github.com/opennota/check/cmd/structcheck
go get -u honnef.co/go/tools/cmd/unused
go get -u honnef.co/go/tools/cmd/gosimple
```
### Run linters
Before every push, make sure you run:
```bash
make check
```

gofmt errors can be automatically fixed by running
```bash
make fix
```

### Run tests
```bash
make test
```

## CI/CD
Currently: [Wercker](https://app.wercker.com/Kubermatic/api) - Which uses the `wercker.yaml` & does a build on every push. 

Future: [Jenkins](https://jenkins.loodse.com) which uses the `Jenkinsfile` & also does a build on every push.


#Documentation

- [Apiserver public port](docs/apiserver-port-range.md)
