# Release

This document explains our process and requirements for creating a new Kubermatic release.

## Testing prior releasing

This section provides a non-exhaustive minimal list of functionality that must be validated prior
to creating a new release. All these tests must be executed with every supported minor Kubernetes version.

1. Conformance tests on all supported providers (AWS, Openstack, Digitalocean, VSphere, Azure, Hetzner)
1. Cloud provider functionality `service type LoadBalancer` on all providers that support it (AWS, Openstack, Azure)
1. Cloud provider functionality `PersistentVolume` on all providers that support it (AWS, Openstack, Vsphere, Azure)
1. HorizontalPodAutoscaler

## Releasing a new version

This section covers the process to create a new Kubermatic release. Reflects the procedure as of 2020-07-20.

### Major|Minor release
1. Kubernetes lifecycle
    - Ensure ConfigMap for new kubelet version is added
    - Ensure RBAC for that ConfigMap is added
    - Ensure end-of-life Kubernetes has been disabled
1. Branching out
    - A release branch(example: `release/v2.7`) has been created in:
      - https://github.com/kubermatic/kubermatic
      - https://github.com/kubermatic/dashboard
      - https://github.com/kubermatic/kubermatic-installer
    - Default branch for https://github.com/kubermatic/kubermatic-installer has been set to the new branch
1. Add a new upgrade pre-submit in infra repo
1. Ensure all providers' conformance tests run on the new branch
1. Ensure upgrade tests runs and can reconcile a cluster
1. Tagging
    - Tag the release in `dashboard` repo
    - Ensure it's built and pushed successfully
    - Tag the matching release in `kubermatic` repo
    - Ensure it's built and pushed successfully
    - Ensure chart sync worked
1. Documenation:
    - Update changelog (using https://github.com/kubermatic/gchl in `master` branch of Kubermatic repo)
      - Remember to include changes from the `dashboard` repo as well, if any
    - Copy it over to matching chapters and branches in docs
      - Strip the Github links from the GCHL version
    - Use the `ACTION REQUIRED` sections of changelog to draft a migration guide (e.g. https://docs.kubermatic.io/kubermatic/v2.12/upgrading/)
    - Have PS/dev test the upgrade and update the migration guide if necessary

## Kubernetes version policy

A Kubermatic minor version supports all Kubernetes versions which were supported upstream at the time of its release. As time passes, patch versions of Kubermatic will support new patch versions of Kubernetes that have been released since, as well as drop old patch versions when they are affected by critical bugs.

Kubermatic (including stable versions) will typically drop support of minor versions of Kubernetes which have gone EOL. This of course results in any stable version of Kubermatic being eventually limited to two and later one minor version of Kubernetes.

One notable exception is when upgrading from an older version of Kubernetes might require extensive migration of loads running within the updated clusters (e.g. API version deprecations) - in these cases Kubermatic will maintain LIMITED support of an EOL Kubernetes version(s) for an additional release cycle for the purpose of facilitating these migrations.
