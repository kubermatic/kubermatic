# Kubermatic | Container Engine

Scale apps one click in cloud or your own datacenter.
Deploy, manage and run multiple Kubernetes clusters with our production-proven platform.
On your preferred infrastructure.

### drone

If you want to change the `.drone.yml` please follow these steps:

```bash
go get -v -u github.com/google/go-jsonnet/jsonnet
go get -v -u github.com/metalmatze/drone-jsonnet
go get -v -u github.com/brancz/gojsontoyaml
```

Compiling .drone.jsonnet to .drone.yml:

`jsonnet -J $(go env GOPATH)/src/github.com/metalmatze/drone-jsonnet .drone.jsonnet | gojsontoyaml > .drone.yml`

For now we use jsonnet as the YAML parser doesn't suppor YAML anchors. https://github.com/go-yaml/yaml/issues/184
