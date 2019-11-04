# Kubermatic API

## Development environment

```bash
mkdir -p $(go env GOPATH)/src/github.com/kubermatic
cd $(go env GOPATH)/src/github.com/kubermatic
git clone git@github.com:kubermatic/api
git clone git@github.com:kubermatic/secrets
cd api
```

There are a couple of scripts in the `api/hacks` directory to aid in running the components locally
for testing purposes.

You can create a cluster via the UI at `https://dev.kubermatic.io`, then use `kubectl` to add a
`worker-name=<<hostname-of-your-laptop>>` label to the cluster. This will make your locally
running controlers manage the cluster.

### Running locally
#### kubermatic-api

```bash
./hack/run-api.sh
```

#### kubermatic-controller-manager
```bash
./hack/run-controller.sh
```

#### master-controller-manager
```bash
./hack/run-master-controller-manager.sh
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

#### Update dependencies
Sure that you want to update? And not just install dependencies?
```bash
dep ensure -update
```
#### Update code generation

```bash
./hack/update-codegen.sh
```

# Documentation
- [AWS Account Creation](docs/aws-account-creation.md)
