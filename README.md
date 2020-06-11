# Kubermatic | Container Engine

Scale apps one click in cloud or your own datacenter.
Deploy, manage and run multiple Kubernetes clusters with our production-proven platform.
On your preferred infrastructure.

## Documentation

This covers the existing, developer focusing, documentation:

- General
  - [API docs](docs/api-docs.md)
  - [Releasing kubermatic](docs/release-process.md)
  - [datacenters.yaml](docs/datacenters.md)
- Development guidelines
  - [Code style](docs/code-style.md)
  - [Logging guideline](docs/logging.md)
  - [Events guideline](docs/events.md)
- [Proposals](docs/proposals)
- [Things that break](docs/things-that-break.md)
- [Resource handling](docs/resource-handling.md)

## Repository layout

```
├── addons            # Default Kubernetes addons
├── api 							# All the code. If you are a dev, you can initially ignore everything else
├── CHANGELOG.md      # The changelog
├── config            # The Helm charts we use to deploy, gets exported to https://github.com/kubermatic/kubermatic-installer
├── containers        # Various utility container images
├── docs              # Some basic docs
├── openshift_addons  # Default Openshift addons
├── OWNERS
├── OWNERS_ALIASES
├── Procfile
└── README.md
```

Customer facing documentation can be found at https://github.com/kubermatic/docs which gets published at https://docs.kubermatic.io

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

#### seed-controller-manager

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
