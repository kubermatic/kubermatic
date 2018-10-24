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

This section covers the process to create a new Kubermatic release.

### Major|Minor release
1. The migration has been documented in https://github.com/kubermatic/docs
1. All new major features which offer configuration via flags & helm charts, have been documented in https://github.com/kubermatic/docs 
1. A release branch(example: `release/v2.7`) has been created in:
  - https://github.com/kubermatic/kubermatic
  - https://github.com/kubermatic/kubermatic-installer
1. Default branch for https://github.com/kubermatic/kubermatic-installer has been set to the new branch
1. The chart sync has been validated to be working (https://github.com/kubermatic/kubermatic-installer contains latest versions of helm charts from https://github.com/kubermatic/kubermatic)
1. Now the new release can be created via https://github.com/kubermatic/kubermatic/releases. 
  - Make sure to use the release branch to create the tag
  - Drone will publish the release(Docker images & charts) 
1. After the release has been created, the changelog must be updated. (Responsible: @kdomanski) 

### Patch release
1. After all relevant patches have been implemented in the master & release branch, a release can be created in Github https://github.com/kubermatic/kubermatic/releases. 
  - Make sure to use the release branch to create the tag
  - Drone will publish the release(Docker images & charts) 
1. After the release has been created, the changelog must be updated. (Responsible: @kdomanski) 
