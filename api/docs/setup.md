# Setup your development enviroment

````bash
mkdir -p $GOPATH/src/github.com/kubermatic
cd $GOPATH/src/github.com/kubermatic
git clone git@github.com:kubermatic/api
git clone git@github.com:kubermatic/config
git clone git@github.com:kubermatic/secrets
cd api

# Link nodeclasses
sudo mkdir -p /opt/template/nodes/ 
sudo ln -s $GOPATH/src/github.com/kubermatic/kubermatic/config/kubermatic/static/nodes/* /opt/template/nodes/

# Install the dependencies
make bootstrap
````
