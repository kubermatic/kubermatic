# Setup your development enviroment

````bash
mkdir -p $GOPATH/src/github.com/kubermatic
cd $GOPATH/src/github.com/kubermatic
git clone git@github.com:kubermatic/api
git clone git@github.com:kubermatic/config
git clone git@github.com:kubermatic/secrets
cd api

# Link cloud-init
mkdir -p template/coreos &&
pushd template/coreos &&
ln -s $GOPATH/src/github.com/kubermatic/kubermatic/config/kubermatic/static/nodes/coreos/cloud-init.yaml cloud-init.yaml &&
popd

# Install the dependencies
make bootstrap
````
