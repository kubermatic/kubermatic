<p align="center">
  <img src="docs/kkp-logo.png#gh-light-mode-only" width="700px" />
  <img src="docs/kkp-logo-dark.png#gh-dark-mode-only" width="700px" />
</p>

<p align="center">
  <img src="https://img.shields.io/github/v/release/kubermatic/kubermatic" alt="last stable release">

  <a href="https://goreportcard.com/report/k8c.io/kubermatic/v2">
    <img src="https://goreportcard.com/badge/k8c.io/kubermatic/v2" alt="go report card">
  </a>

  <a href="https://pkg.go.dev/k8c.io/kubermatic/v2">
    <img src="https://pkg.go.dev/badge/k8c.io/kubermatic/v2" alt="godoc">
  </a>
</p>

## Overview / User Guides

Kubermatic Kubernetes Platform is in an open source project to centrally manage the global automation of thousands of Kubernetes clusters across multicloud, on-prem and edge with unparalleled density and resilience.

All user documentation is available at the [Kubermatic Kubernetes Platform docs website][21].

## Editions

There are two editions of Kubermatic Kubernetes Platform:

Kubermatic Kubernetes Platform Community Edition (CE) is available freely under the Apache License, Version 2.0.
Kubermatic Kubernetes Platform Enterprise Edition (EE) includes premium features that are most useful for organizations with large-scale Kubernetes installations with more than 50 clusters. To access the Enterprise Edition and get official support please become a subscriber.

## Licensing

See the [LICENSE](LICENSE) file for licensing information as it pertains to files in this repository.

## Installation

We strongly recommend that you use an official release of Kubermatic Kubernetes Platform. Follow the instructions under the **Installation** section of [our documentation][21] to get started.

_The code and sample YAML files in the main branch of the kubermatic repository are under active development and are not guaranteed to be stable. Use them at your own risk!_

## More information

[The documentation][21] provides a getting started guide, plus information about building from source, architecture, extending kubermatic, and more.

Please use the version selector at the top of the site to ensure you are using the appropriate documentation for your version of kubermatic.

## Troubleshooting

If you encounter issues [file an issue][1] or talk to us on the [#kubermatic channel][12] on the [Kubermatic Community Slack][15] ([click here to join][16]).

## Contributing

Thanks for taking the time to join our community and start contributing!

### Before you start

* Please familiarize yourself with the [Code of Conduct][4] before contributing.
* See [CONTRIBUTING.md][2] for instructions on the developer certificate of origin that we require.

### Repository layout

```
├── addons    # Default Kubernetes addons
├── charts    # The Helm charts we use to deploy
├── cmd       # Various Kubermatic binaries for the controller-managers, operator etc.
├── codegen   # Helper programs to generate Go code and Helm charts
├── docs      # Some basic developer-oriented documentation
├── hack      # scripts for development and CI
└── pkg       # most of the actual codebase
```

### Development environment

```bash
git clone git@github.com:kubermatic/kubermatic.git
cd kubermatic
```

There are a couple of scripts in the `hacks` directory to aid in running the components locally
for testing purposes.

#### Running components locally

##### user-cluster-controller-manager

In order to instrument the seed-controller to allow for a local user-cluster-controller-manager, you need to add a `worker-name` label with your local machine's name as its value. Additionally, you need to scale down the already running deployment.

```sh
# Using a kubeconfig, which points to the seed-cluster
export cluster_id="<id-of-your-user-cluster>"
kubectl label cluster ${cluster_id} worker-name=$(uname -n)
kubectl scale deployment -n cluster-${cluster_id} usercluster-controller --replicas=0
```

Afterwards, you can start your local user-cluster-controller-manager.

```sh
# Using a kubeconfig, which points to the seed-cluster
./hack/run-user-cluster-controller-manager.sh
```

##### seed-controller-manager

```bash
./hack/run-seed-controller-manager.sh
```

##### master-controller-manager

```bash
./hack/run-master-controller-manager.sh
```

#### Run linters

Before every push, make sure you run:

```bash
make lint
```
#### Run tests

```bash
make test
```

#### Update code generation

The Kubernetes code-generator tool does not work outside of `GOPATH`
([upstream issue](https://github.com/kubernetes/kubernetes/issues/86753)), so the script
below will automatically run the code generation in a Docker container.

```bash
hack/update-codegen.sh
```

### Pull requests

* We welcome pull requests. Feel free to dig through the [issues][1] and jump in.

## Changelog

See [the list of releases][3] to find out about feature changes.

[1]: https://github.com/kubermatic/kubermatic/issues
[2]: https://github.com/kubermatic/kubermatic/blob/main/CONTRIBUTING.md
[3]: https://github.com/kubermatic/kubermatic/releases
[4]: https://github.com/kubermatic/kubermatic/blob/main/CODE_OF_CONDUCT.md

[12]: https://kubermatic-community.slack.com/messages/kubermatic
[15]: http://kubermatic-community.slack.com
[16]: https://join.slack.com/t/kubermatic-community/shared_invite/zt-vqjjqnza-dDw8BuUm3HvD4VGrVQ_ptw

[21]: https://docs.kubermatic.com/kubermatic/
test
