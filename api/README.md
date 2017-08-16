# Kubermatic API
[Jenkins](https://jenkins.loodse.com/blue/pipelines)

---

## Development environment
To setup your enviroment clock [here](docs/setup.md).

### Dependencies
#### Bootstrap project

```
make bootstrap
```

#### Update dependencies

```bash
dep ensure -update
```

### Running locally
#### kubermatic-api

```bash
./hack/run-api.sh
```

#### kubermatic-cluster-controller
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
When you want to test it on the Jenkins build server prefix your commit with an `!e2e`

# Documentation
- [AWS Account Creation](docs/aws-account-creation.md)
- [Load Script Usage](docs/load-script-usage.md)
