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

This section covers the process to create a new Kubermatic release. Reflects the procedure as of 2020-09-16.

### Patch release

For a patch release on an existing `release/vX.Y` branch:

1. Bump the OSM and machine-controller images in a **single PR**, not two.
    - Update both image `Tag` values: `pkg/resources/operatingsystemmanager/deployment.go`
      and `pkg/resources/machinecontroller/deployment.go`, plus the matching `go.mod`
      lines and the re-vendored tree.
    - One PR runs the conformance suite once for both components. Two separate PRs run
      it twice and, on a release branch, force a Tide base-move retest of the second PR
      after the first merges, costing an extra full conformance run.
    - The `pre-kubermatic-verify-component-bumps` presubmit enforces this: a PR that
      bumps only one of the two image Tags fails the check.
    - If you genuinely need to bump only one component (OSM and machine-controller
      release independently), add the `release/single-component-bump` label and comment
      `/retest` to re-run the check, which then passes.
1. Add the changelog in a separate cheap PR (`docs/changelogs/CHANGELOG-X.Y.md`).
1. Tag the release (dashboard first, then kubermatic) as in the tagging step below.

### Major|Minor release

1. Kubernetes lifecycle
    - Ensure ConfigMap for new kubelet version is added
    - Ensure RBAC for that ConfigMap is added
    - Ensure end-of-life Kubernetes has been disabled
1. Ensure all providers' conformance tests run
1. Branching out
    - A release branch(example: `release/v2.15`) has been created in:
      - https://github.com/kubermatic/kubermatic
      - https://github.com/kubermatic/dashboard
1. Duplicate the `main` documentation for KKP in https://github.com/kubermatic/docs
   (i.e. copy the content, data, etc. files and adjust accordingly)
1. Adjust postsubmit jobs in the infra repo to start running for
   the new release branch
1. Tagging
    - Tag the release in `dashboard` repo
    - Ensure it's built and pushed successfully
    - Tag the matching release in `kubermatic` repo
    - Ensure it's built and pushed successfully
1. Documentation:
    - Update changelog (using `hack/changelog-gen.sh`)
      - Remember to include changes from the `dashboard` repo as well, if any
    - Copy it over to matching chapters and branches in docs
      - Strip the Github links from the GCHL version
