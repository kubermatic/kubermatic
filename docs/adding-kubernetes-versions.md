# Adding/Updating Kubernetes Versions

This document describes the process of adding support for a new Kubernetes minor/patch version
to KKP.

The single source of truth for the set of Kubernetes versions we ship is
`DefaultKubernetesVersioning` in `pkg/defaulting/configuration.go` (around line 225). The example
documentation in `docs/` is generated from it.

## Version Skews

When removing support for a Kubernetes release, care must be taken because of existing userclusters.
If for example Kubernetes 1.20 is removed, that KKP version must still be able to reconcile 1.20
clusters while the upgrades are running (KKP potentially doesn't reconcile all userclusters at the
same time).

So removing support for a Kubernetes release is a 2-step process:

1. Remove it from the list of supported versions (in `pkg/defaulting/configuration.go`)
   and release this as a new KKP minor version.
2. In the next KKP minor version, all the reconciling code for the removed Kubernetes version
   can be deleted.

The removal PR itself (the pattern of #15832, `bf99c183a`) touches more than the version list:

- Version-keyed maps (`autoscalerImageTags`, the konnectivity, dashboard and CSI tables) keep
  exactly one entry below the supported floor. A drop removes the entry two minors below the new
  floor and keeps the just-dropped minor: existing clusters at that version still reconcile and
  render application values until they are upgraded away.
- `pkg/applications/test/kubernetes.go` builds the fake cluster used by the helm integration
  tests at the autoscaler map's lowest entry. Every drop breaks `TestHelmProvider` in
  `pre-kubermatic-test-integration` with "failed to parse autoscaler version" until the helper
  is bumped (#16518). Pin it at the oldest creatable version instead of the map's lowest entry
  and it stops re-breaking.
- `TestNetworkProxyVersion` in `pkg/resources/konnectivity/sidecar_test.go` carries rows for the
  removed konnectivity cases; delete them together with the switch case or `pre-kubermatic-test`
  fails.
- Regenerate `docs/zz_generated.kubermaticConfiguration.{ce,ee}.yaml`; the generated files carry
  the removed versions too.

## Adding/Removing Patch Releases

Update `DefaultKubernetesVersioning` in `pkg/defaulting/configuration.go` accordingly.

Next, re-generate the Helm chart and documentation:

```bash
./hack/update-docs.sh
```

As a last step, update the e2e jobs in the `.prow/` files (`provider-*.yaml`, `tests.yaml`) to use
the most recent patch versions for all supported minor versions.

## Adding/Removing Minor Releases

Support for a new minor release is a cross-repo effort. Land the pieces in this order:

1. `operating-system-manager` and `machine-controller` bump their `k8s.io/v0.X` Go libraries first.
2. The `kubermatic` PR (this repo) lands.
3. The dashboard bumps its KKP Go dependency (`make update-kkp` in `modules/api`, both
   `k8c.io/kubermatic/v2` and `k8c.io/sdk/v2`).

On the operating-system-manager side the version bump is not the only change. The default OSPs in
`deploy/osps/default/` select the crictl version per minor via a `semverCompare "~1.X.0"` branch in
`CRI_TOOLS_RELEASE`; every in-scope OSP needs a branch and a fallback bump for the new minor (this
was forgotten for 1.36 and needed a follow-up PR). Also audit the kubelet systemd units against
upstream flag removals each minor: kubelet hard-fails on removed flags (1.35 removed
`--pod-infra-container-image`, 1.37 made the deprecated cAdvisor flags fatal). Containerd needs no
per-minor pin: the ubuntu OSP apt-installs `containerd.io`, flatcar ships containerd with the OS via
torcx, and OSM only writes `/etc/containerd/config.toml`; check that the new minor does not raise
the minimum containerd version.

Review expectations distilled from the last six minor-support PRs in each repo (raw review dumps:
`docs/analysis/k8s-minor-support-reviews/`):

Machine-controller (from #1840, #1955, #1995, #2042, #1985, #2016):

- The e2e matrix update is what makes support real; a support PR without the `helper.go` versions
  list and selector updates has been rejected outright.
- Keep the matrix on the latest patch release of every supported minor, not .0 or stale pins.
- Negative version selectors (`Not(VersionSelector(...))`) from the in-tree CCM era are removed,
  not extended.
- New `.golangci.yml` exclusions are tolerated only with a linked follow-up issue that the
  contributor owns; otherwise fix the code.
- Example manifests pin kubelet at newest-minus-one.
- Release notes enumerate the support add, the dropped minor, and the controller-runtime/Go bumps.
- Run the provider e2e suites before merging; the suite is matrix-driven, so one run covers every
  version in the list.

Operating-system-manager (from #410, #465, #512, #555, #556, #613, #615):

- Ship the cri-tools branch for the new minor in the support PR itself; the 1.36 cycle forgot it
  and needed a follow-up PR one day later.
- Kubelet flag compatibility draws the most scrutiny: check the new minor's CHANGELOG for removed
  flags and cite it when adjusting the OSP units.
- The Go version pin must be the current patch release, not a stale one.
- `docs/compatibility-matrix.md` moves in the same PR.
- Support-PR backports conflict on the fixture commits and need manual resolution.

kubermatic (from #13593, #13984, #14419, #14940, #15347, #15986):

- Regenerate addon YAML with `make` in the addon directory (or
  `cd addons && find . -type d -exec make -C {} \;`); never hand-edit rendered manifests. Windows
  DaemonSets that chart regeneration re-imports are stripped via `customizations.patch`.
- CCM pins: an explicit case per minor when the image exists; fallthrough only to the newest
  available tag, with a comment. Settle tag questions with a registry query (`gcrane ls` or a
  manifest GET), not the GitHub release page; images appear in the registry later than the tag.
- CSI version maps: reusing the previous minor's sidecar tag is fine only while no newer upstream
  release exists; check each driver's releases page.
- The cluster-autoscaler tag must match the upstream release; upstream ships these mid-review, so
  re-check right before merge.
- kubectl binaries must cover the oldest cluster version KKP can still reconcile, not just the
  newest.
- Copied prow jobs are the historical minefield: DISTRIBUTIONS and RELEASES_TO_TEST must match the
  job name and provider.
- Fixture drift blocks merge; regenerate before pushing.
- E2e: run the new minor across providers plus regression runs of older minors; `/hold` the PR on
  cross-repo dependencies (for example the machine-controller and OSM tag bumps) and `/override`
  only with a stated reason.
- Etcd version gating gets correctness review including nil-safety on the apiserver semver.
- Release notes name the dropped minor and the notable dependency bumps.

### Go modules

- Bump the `k8s.io` stack in `go.mod`. Since 1.35 the same bump is needed in `sdk/go.mod` as well.
- `controller-runtime` and `controller-tools` usually need to follow.
- The library bump typically forces a Go directive bump in both modules.

### Version list

- Update `pkg/defaulting/configuration.go`. Set the default version to the most recent version
  minus 1 (i.e. if 1.19.2 is the most recent version we support, set the default to the latest 1.18
  version) and make sure to define upgrade paths for previous Kubernetes versions as well.
- Update `pkg/resources/test/load_files_test.go` `TestLoadFiles()` to make it generate
  manifests for the new minor version.

### Per-provider CCM images

`pkg/resources/cloudcontroller/` has one file per provider (`aws.go`, `azure.go`, `gcp.go`,
`openstack.go`, `vsphere.go`, `digitalocean.go`, `hetzner.go`, `anexia.go`) with a version switch
for the CCM image tag, plus shared version constants in `util.go`. Add a case for the new minor in
each. When upstream has not published an image for the new minor yet, fall through to the previous
minor's tag and leave a comment (AWS did this for 1.36 in #15986).

### Version-gated images to review every minor

- etcd: `ImageTag` in `pkg/resources/etcd/statefulset.go` (clusters >= 1.36 switched to etcd 3.6.12,
  older versions stay on 3.5). The corruption-check version constraint in the same file gates flags
  passed to etcd-launcher: when it matches, etcd is started with
  `--experimental-initial-corrupt-check` / `--experimental-corrupt-check-time`
  (cmd/etcd-launcher/pkg/etcd/cmd.go). Etcd 3.7 removed those flag names (only `--corrupt-check-time`
  exists), so before widening the constraint past 3.7, etcd-launcher must graduate its flags first;
  otherwise every etcd pod on the new version crashloops on startup.
- konnectivity: `NetworkProxyVersion` in `pkg/resources/konnectivity/sidecar.go`.
- kubernetes-dashboard: `DashboardVersion` in `pkg/resources/kubernetes-dashboard/deployment.go`.
- cluster-autoscaler: the `autoscalerImageTags` map in
  `pkg/applications/providers/template/util.go`. There is no cluster-autoscaler addon anymore
  (deleted in #15311), do not look for one.

### CNI manifests

- Canal: add a per-version yaml under `addons/canal/` when the new minor needs a newer Canal
  release (`canal_v3.30.yaml` was added for 1.34).
- Cilium: check the Cilium version handling in the `pkg/applications` util area.

### CSI addons

- Bump the chart version in the `Makefile` of every driver that needs it under `addons/csi/`
  (aws-ebs, azure-disk, azure-file, openstack, vsphere, digitalocean, hetzner, nutanix,
  azure-snapshot-controller, gcp, kubevirt, vmware-cloud-director) and run `make` in the driver
  directory to regenerate `driver.yaml`.
- The azure-disk, azure-file and openstack `_header.txt` files gate the sidecar image version on
  `.Cluster.MajorMinorVersion`; add a block for the new minor. If upstream has no image for it yet,
  reuse the previous minor's tag (azure-disk 1.35 and 1.36 both fall through to v1.34.5).
- Upstream sometimes ships the driver image release without publishing the matching helm chart:
  azure-disk 1.34.6 and azure-file 1.35.8 existed as images while the chart indexes stopped at
  1.34.5 and 1.35.7 (#16469). The `Makefile` pin cannot follow the image then; `_header.txt` is
  an input to the regeneration, so the image bump survives `make`. The check is that `make`
  reproduces the committed `driver.yaml` byte for byte, not that the pins match.
- `addons/csi/digitalocean/csi-driver.yaml` also gates on `.Cluster.MajorMinorVersion` and defaults
  to `UNSUPPORTED`, with the whole manifest skipped for unknown minors: without a block for the new
  minor, DigitalOcean clusters get no CSI driver at all. The DigitalOcean CCM
  (`pkg/resources/cloudcontroller/digitalocean.go`) uses a chained fallthrough instead, so the two
  behaviors differ.
- Update the version conditionals in `addons/azure-cloud-node-manager/cloud-node-manager.yaml`.
- The installer mirrors cri-tools per minor via the `criToolsReleases` map in
  `cmd/kubermatic-installer/cmd_mirror_binaries.go`. It lives in `cmd/`, outside `pkg/resources`,
  and is easy to miss; without an entry, airgapped clusters get a stale crictl.

### CRDs and generated docs

Regenerate the CRDs in `pkg/crd/k8c.io/` and `docs/zz_generated.kubermaticConfiguration.{ce,ee}.yaml`
with the appropriate `hack/` scripts.

### kubectl binaries

- Update `BinaryForClusterVersion` in `pkg/util/kubectl/kubectl.go`. Version skew is exploited
  here: not every minor ships its own kubectl binary, so only add a new case when needed.
- Add new kubectl binaries to the root `Dockerfile`.
- Update the `util` image (`hack/images/util/Dockerfile`) to use a newer kubectl version if needed.

### CI

Prow config lives in 15 files under `.prow/` (`provider-*.yaml`, `tests.yaml`, `features.yaml`,
`dualstack.yaml`, `applications-catalog.yaml`). E2E job definitions are in the
`provider-*.yaml` files. Since 1.35, jobs are added per provider additively: copy an existing job
as `pre-kubermatic-e2e-<provider>-ubuntu-1.X` with `RELEASES_TO_TEST` set to the new minor, instead
of renaming an existing job for the oldest minor.

The conformance-tester (`cmd/conformance-tester/pkg/tests/conformance.go`) has a list of exclusion
filters for tests that cannot run in the CI environment. Check it; usually no change is needed.

### Regenerate

```bash
./hack/update-docs.sh
./hack/update-fixtures.sh
```

`update-docs.sh` runs inside the build container. On Apple Silicon the container dies on a
Rosetta error, and the native fallback has a trap: the script rewrites tag markers in the
vendored SDK before generating, and a global `GOFLAGS=-mod=mod` makes `go run` ignore the
vendor tree, so the generators silently drop every optional field from the seed and
configuration example yamls (about 2000 lines) and `pre-kubermatic-verify` keeps failing. Run
the native fallback with `GOFLAGS=-mod=vendor` and check that the diff only adds the expected
entries. `verify-docs.sh` itself does not run on macOS (`cp -ar`); validate by generating twice
and expecting no second diff.

Fixtures make up 85 to 95 percent of the PR diff (roughly 520 to 570 files) and carry no review
signal.

### Verifying the integration tests locally

`go test ./pkg/applications/...` returns a cached result that never ran the helm integration
tests; they sit behind `//go:build integration`. The CI-faithful invocation needs
`-tags "integration,ce"`, `CGO_ENABLED=1`, the envtest control-plane binaries
(`KUBEBUILDER_ASSETS`, via the repo's `download_envtest` recipe) and a registry hostname that
resolves to loopback (`-registry-hostname <host>.local`; containerd rejects a literal localhost
match).

### Follow-up work to budget for

- cluster-autoscaler RBAC alignment with the new minor's APIs (e.g. `resource.k8s.io` DRA access
  for 1.34+, #16251).
- konnectivity version alignment.
- Per-provider e2e stabilization after the version lands.
- Separate PRs dropping the oldest supported minor (see Version Skews above).
