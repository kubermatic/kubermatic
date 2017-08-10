# Kubermatic API
[Jenkins](https://jenkins.loodse.com/blue/pipelines)

## Development environment
Copy the bootstrap script and execute it [bootstrap.sh](hack/bootstrap.sh)

### Dependencies
#### Update dependencies

```bash
glide update --strip-vendor
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

Valid worker-name label value must be 63 characters or less and must be empty or begin and end with an alphanumeric character ([a-z0-9A-Z]) with dashes (-), underscores (_), dots (.), and alphanumerics between.
The dev label should be also unique between a pair of api<->controller.

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

#Documentation

- [Apiserver public port](docs/apiserver-port-range.md)
