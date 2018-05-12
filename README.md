# Kubermatic | Container Engine

Scale apps one click in cloud or your own datacenter.  
Deploy, manage and run multiple Kubernetes clusters with our production-proven platform.  
On your preferred infrastructure.

### drone

If you want to change the `.drone.yml` please follow these steps:

`go get -v -u github.com/metalmatze/drone-jsonnet`  
`go get -v -u github.com/brancz/gojsontoyaml`

Compiling .drone.jsonnet to .drone.yml:

`jsonnet -J $GOPATH/github.com/metalmatze/drone-jsonnet .drone.jsonnet | gojsontoyaml > .drone.yml`
