# Kubermatic Kubernetes Platform

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

_The code and sample YAML files in the master branch of the kubermatic repository are under active development and are not guaranteed to be stable. Use them at your own risk!_

## More information

[The documentation][21] provides a getting started guide, plus information about building from source, architecture, extending kubermatic, and more.

Please use the version selector at the top of the site to ensure you are using the appropriate documentation for your version of kubermatic.

## Troubleshooting

If you encounter issues [file an issue][1] or talk to us on the [#kubermatic channel][12] on the [Kubermatic Slack][15].

## Contributing

Thanks for taking the time to join our community and start contributing!

Feedback and discussion are available on [the mailing list][11].

### Before you start

* Please familiarize yourself with the [Code of Conduct][4] before contributing.
* See [CONTRIBUTING.md][2] for instructions on the developer certificate of origin that we require.
* Read how [we're using ZenHub][13] for project and roadmap planning

### Repository layout

```
├── addons            # Default Kubernetes addons
├── CHANGELOG.md      # The changelog
├── cmd               # Various Kubermatic binaries for the API, controller-managers etc.
├── codegen           # Helper programs to generate Go code and Helm charts
├── charts            # The Helm charts we use to deploy, gets exported to https://github.com/kubermatic/kubermatic-installer
├── containers        # Various utility container images
├── docs              # Some basic docs
├── hack              # scripts for development and CI
├── openshift_addons  # Default Openshift addons
├── OWNERS
├── OWNERS_ALIASES
├── pkg               # most of the actual codebase
├── Procfile
└── README.md
```

### Development environment

```bash
mkdir -p $(go env GOPATH)/src/k8c.io
cd $(go env GOPATH)/src/k8c.io
git clone git@github.com:kubermatic/kubermatic
cd kubermatic
```

Or alternatively:

```bash
go get k8c.io/kubermatic
```

There are a couple of scripts in the `hacks` directory to aid in running the components locally
for testing purposes.

You can create a cluster via the UI at `https://dev.kubermatic.io`, then use `kubectl` to add a
`worker-name=<<hostname-of-your-laptop>>` label to the cluster. This will make your locally
running controllers manage the cluster.

#### Running locally

##### kubermatic-api

```bash
./hack/run-api.sh
```

##### seed-controller-manager

```bash
./hack/run-controller.sh
```

##### master-controller-manager

```bash
./hack/run-master-controller-manager.sh
```

#### Run linters

Before every push, make sure you run:

```bash
make check
```

gofmt errors can be automatically fixed by running

```bash
make fix
```

#### Run tests

```bash
make test
```

#### Update code generation

Kuberneres code-generator tool does not work out of GOPATH
[see](https://github.com/kubernetes/kubernetes/issues/86753), if you followed
the recommendation and cloned your repository at `$(go env GOPATH)/src/k8c.io`
you can run the following command:

```bash
hack/update-codegen.sh
```

Otherwise run the following (requires Docker):

```bash
make update-codegen-in-docker
```

### Pull requests

* We welcome pull requests. Feel free to dig through the [issues][1] and jump in.

## Changelog

See [the list of releases][3] to find out about feature changes.

[1]: https://github.com/kubermatic/kubermatic/issues
[2]: https://github.com/kubermatic/kubermatic/blob/master/CONTRIBUTING.md
[3]: https://github.com/kubermatic/kubermatic/releases
[4]: https://github.com/kubermatic/kubermatic/blob/master/CODE_OF_CONDUCT.md

[11]: https://groups.google.com/forum/#!forum/kubermatic-dev
[12]: https://kubermatic.slack.com/messages/kubermatic
[13]: https://github.com/kubermatic/kubermatic/blob/master/Zenhub.md
[15]: http://slack.kubermatic.io/

[21]: https://docs.kubermatic.com/kubermatic/
