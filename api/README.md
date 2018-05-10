# Kubermatic API
[Drone](https://drone.loodse.com/kubermatic/kubermatic)

---

## Development environment
To setup your enviroment click [here](docs/setup.md).

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
