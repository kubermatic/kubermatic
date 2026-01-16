# Adding/Updating Kubernetes Versions

This document describes the process of adding support for a new Kubernetes minor/patch version
to KKP.

The single source of truth for the set of Kubernetes versions we ship is defined in
`pkg/controller/operator/defaults/defaults.go`. From there the example documentation is generated.

## Version Skews

When removing support for a Kubernetes release, care must be taken because of existing userclusters.
If for example Kubernetes 1.20 is removed, that KKP version must still be able to reconcile 1.20
clusters while the upgrades are running (KKP potentially doesn't reconcile all userclusters at
the same time).

So removing support for a Kubernetes release is a 2-step process:

1. Remove it from the list of supported versions (in `pkg/controller/operator/defaults/defaults.go`)
   and release this as a new KKP minor version.
2. In the next KKP minor version, all the reconciling code for the removed Kubernetes version
   can be deleted.

## Adding/Removig Patch Releases

Update the `defaults.go` variable `DefaultKubernetesVersioning` accordingly.

Next, re-generate the Helm chart and documentation:

```bash
./hack/update-docs.sh
```

As a last step, update the `.prow.yaml` and change the e2e jobs to use the most recent
patch versions for all support minor versions.

## Adding/Removing Minor Releases

Support for minor releases is a bit more involved to add. There are a couple of places that
need to be updated.

Before a new minor can be added to KKP, the `build` Docker image must be updated to:

- include test binaries for the new Kubernetes version
- include the appropriate kubectl versions

Bump the `build` Docker image and push new images to quay before continuing.

Once new Docker images are ready, KKP can be updated as well.

- Add a new Prow Job to the `.prow.yaml` to test the latest patch release for the new
  minor (just copy an existing job and adjust accordingly). Make sure to change the Docker
  image tag for the e2e images to use the new tag you just created with the new test binaries.
- Update the CSI addon manifests (`addon/csi/*.yaml`) to include the new minor version.
- Update the CCM manifests located in `pkg/resources/cloudcontroller`) to include the new minor version.
- The conformance-tests runner (`cmd/conformance-tester/pkg/tests/conformance.go`) has a list of
  exclusion filters to skip tests that cannot work in the CI environment. Make sure to
  update said list, or else you will be greeted by lots of NodePort Service related
  errors.
- Update the `pkg/defaulting/configuration.go`. Set the default version to the most recent version
  minus 1 (i.e. if 1.19.2 is the most recent version we support, set the default to the latest 1.18
  version) and make sure to define upgrade paths for previous Kubernetes versions as well.
- Update `pkg/resources/test/load_files_test.go` `TestLoadFiles()` to make it generate
  manifests for the new minor version.
- Update `pkg/util/kubectl/kubectl.go` to use the appropriate kubectl version for the
  new minor version
  - Update the Dockerfile to include used kubectl versions.
  - Update the `util` image (`hack/images/util/Dockerfile`) to use a newer kubectl version if needed.

Lastly, re-generate the Helm chart and documentation:

```bash
./hack/update-docs.sh
./hack/update-fixtures.sh
```
