# Kubermatic API
[Drone](https://drone.loodse.com/kubermatic/kubermatic)

---

## Development environment

```bash
mkdir -p $(go env GOPATH)/src/github.com/kubermatic
cd $(go env GOPATH)/src/github.com/kubermatic
git clone git@github.com:kubermatic/api
git clone git@github.com:kubermatic/secrets
cd api
```

There are a couple of scripts in the `api/hacks` directory to aid in running the components locally for testing
purposes.

#### Update dependencies
Sure that you want to update? And not just install dependencies?
```bash
dep ensure -update
```
#### Update code generation

```bash
./hack/update-codegen.sh
```

### Running locally
#### kubermatic-api

```bash
./hack/run-api.sh
```

#### kubermatic-controller-manager
```bash
./hack/run-controller.sh
```

#### kubermatic-rbac-generator
```bash
./hack/run-rbac-generator.sh
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

#### E2E-tests
Locally run:
```bash
make e2e         #run the e2e container (needs _artifacts/kubeconfig)
make client-down #deletes all clusters from the given user
```

# Documentation
- [AWS Account Creation](docs/aws-account-creation.md)
- [Load Script Usage](docs/load-script-usage.md)
