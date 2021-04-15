# Adding/Updating Kubernetes Versions

This document describes the process of adding support for a new Kubernetes minor/patch version
to KKP.

The single source of truth for the set of Kubernetes versions we ship is defined in
`pkg/controller/operator/common/defaults.go`. From there, the `kubermatic` Helm chart and the
example documentation are generated.

## Adding/Removig Patch Releases

Update the `defaults.go` variable `DefaultKubernetesVersioning` accordingly.

Next, re-generate the Helm chart and documentation:

```bash
./hack/update-kubermatic-chart.sh
./hack/update-docs.sh
```

Bump the `kubermatic` chart version in `charts/kubermatic/Chart.yaml`.

As a last step, update the `.prow.yaml` and change the e2e jobs to use the most recent
patch versions for all support minor versions.

## Adding/Removing Minor Releases

Support for minor releases is a bit more involved to add. There are a coulple of places that
need to be updated.

Before a new minor can be added to KKP, the test binaries need to be included in the e2e Docker
image. Bump the `e2e-kind` and `e2e-kind-with-conformance-tests` Docker images and push new
images to quay before continuing.

Once new Docker images are ready, KKP can be updated as well.

- Add a new Prow Job to the `.prow.yaml` to test the latest patch release for the new
  minor (just copy an existing job and adjust accordingly). Make sure to change the Docker
  image tag for the e2e images to use the new tag you just created with the new test binaries.
- Update the CSI addon manifests (`addon/csi/*.yaml`) to include the new minor version.
- Ensure a kubelet ConfigMap for the new minor exists in `addons/kubelet-configmap/kubelet-configmap.yaml`.
- Update `addons/rbac/allow-kubeadm-join-configmap.yaml` to include the new ConfigMap.
- Update the OpenStack CCM manifest (`pkg/resources/cloudcontroller/openstack.go`) to
  include the new minor version.
  - The latest OpenStack CCM version can be found in the
  [`kubernetes/cloud-provider-openstack` repository](https://github.com/kubernetes/cloud-provider-openstack).
- The conformance-tests runner (`cmd/conformance-tests/runner.go`) has a list of
  exclusion filters to skip tests that cannot work in the CI environment. Make sure to
  update said list, or else you will be greeted by lots of NodePort Service related
  errors.
- Update the `pkg/controller/operator/common/defaults.go`. Set the default version to
  the most recent version minus 1 (i.e. if 1.19.2 is the most recent version we support,
  set the default to the latest 1.18 version) and make sure to define upgrade paths
  for previous Kubernetes versions as well.
- Update `pkg/resources/test/load_files_test.go` `TestLoadFiles()` to make it generate
  manifests for the new minor version.

Lastly, re-generate the Helm chart and documentation:

```bash
./hack/update-kubermatic-chart.sh
./hack/update-docs.sh
./hack/update-fixtures.sh
```

Bump the `kubermatic` chart version in `charts/kubermatic/Chart.yaml`.
