# Kubermatic | Container Engine

Scale apps one click in cloud or your own datacenter.
Deploy, manage and run multiple Kubernetes clusters with our production-proven platform.
On your preferred infrastructure.

# Documentation

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

# Repository layout

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
foo
