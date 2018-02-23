# Setup your development enviroment

````bash
mkdir -p $(go env GOPATH)/src/github.com/kubermatic
cd $(go env GOPATH)/src/github.com/kubermatic
git clone git@github.com:kubermatic/api
git clone git@github.com:kubermatic/secrets
cd api

# Link nodeclasses
sudo mkdir -p /opt/template/nodes/ 
sudo ln -s $(go env GOPATH)/src/github.com/kubermatic/kubermatic/config/kubermatic/static/nodes/* /opt/template/nodes/

````
