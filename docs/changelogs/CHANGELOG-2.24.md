# Kubermatic 2.24

- [v2.24.0](#v2240)

## [v2.24.0](https://github.com/kubermatic/kubermatic/releases/tag/v2.24.0)

Before upgrading, make sure to read the [general upgrade guidelines](https://docs.kubermatic.com/kubermatic/v2.24/installation/upgrading/). Consider tweaking `seedControllerManager.maximumParallelReconciles` to ensure user cluster reconciliations will not cause resource exhaustion on seed clusters. A [full upgrade guide is available from the official documentation](https://docs.kubermatic.com/kubermatic/v2.24/installation/upgrading/upgrade-from-2.23-to-2.24/).

### Read Before Upgrading

- **ACTION REQUIRED:** legacy backup controller has been removed. Before upgrading, please change to the backup and restore feature that uses [backup destinations](https://docs.kubermatic.com/kubermatic/v2.24/tutorials-howtos/etcd-backups/) if the legacy controller is still in use ([#12473](https://github.com/kubermatic/kubermatic/pull/12473))
- `s3-storeuploader` has been removed ([#12473](https://github.com/kubermatic/kubermatic/pull/12473))
- OpenVPN for control plane to node connectivity has been deprecated. It will be removed in future releases of KKP. Upgrading all user cluster to Konnectivity is strongly recommended ([#12691](https://github.com/kubermatic/kubermatic/pull/12691))

### API Changes

- The field `vmNetName` in `Cluster` and `Preset` resources for vSphere clusters is deprecated and `networks` should be used instead ([#12444](https://github.com/kubermatic/kubermatic/pull/12444))
- The field `konnectivityEnabled` in `Cluster` resources is deprecated. Clusters should set this to `true` to migrate off OpenVPN as Konnectivity being enabled will be assumed in future KKP releases ([#12691](https://github.com/kubermatic/kubermatic/pull/12691))

### Supported Kubernetes Versions

- Add Support for Kubernetes 1.28 ([#12593](https://github.com/kubermatic/kubermatic/pull/12593))
- Add support for Kubernetes 1.26.9, 1.27.6 and 1.28.2 ([#12638](https://github.com/kubermatic/kubermatic/pull/12638))
- Set default Kubernetes version to 1.27.6 ([#12638](https://github.com/kubermatic/kubermatic/pull/12638))
- Remove support for Kubernetes 1.24 ([#12570](https://github.com/kubermatic/kubermatic/pull/12570))

#### Supported Versions

- v1.26.1
- v1.26.4
- v1.26.6
- v1.26.9
- v1.27.3
- v1.27.6 (default)
- v1.28.2

### Cloud Providers

#### Azure

- Remove Azure NSG rules that only duplicated rules always present in NSGs ([#12565](https://github.com/kubermatic/kubermatic/pull/12565))
- The `icmp_allow_all` rule of the Azure NSG created for each cluster now only allows ICMP and takes precedence over the TCP and UDP catch-all rules that were guarding it ([#12559](https://github.com/kubermatic/kubermatic/pull/12559))

#### vSphere

- Support for configuring multiple networks for vSphere ([#12444](https://github.com/kubermatic/kubermatic/pull/12444))
- Support for propagating vSphere cluster tags to folders created by KKP ([#12581](https://github.com/kubermatic/kubermatic/pull/12581))
- Update vSphere CCM to 1.27.2 for Kubernetes 1.27 user clusters ([#12599](https://github.com/kubermatic/kubermatic/pull/12599))
- If a vSphere cluster uses a custom datastore, the Seed's default datastore should not be validated ([#12655](https://github.com/kubermatic/kubermatic/pull/12655))
- Add `basePath` optional configuration for vSphere clusters that will be used to construct a cluster-specific folder path (`<root path>/<base path>/<cluster ID>` or `<base path>/<cluster ID>`) ([#12668](https://github.com/kubermatic/kubermatic/pull/12668))
- Fix a bug where datastore cluster value was not being propagated to the CSI driver ([#12474](https://github.com/kubermatic/kubermatic/pull/12474))

#### DigitalOcean

- Digitalocean CCM versions now depend on the user cluster version, following the loose [upstream compatibility guarantees](https://github.com/digitalocean/digitalocean-cloud-controller-manager#releases) ([#12600](https://github.com/kubermatic/kubermatic/pull/12600))

#### Hetzner

- Hetzner CSI: recreate CSIDriver to allow upgrade from 1.6.0 to 2.2.0 ([#12432](https://github.com/kubermatic/kubermatic/pull/12432))
- EE: Correctly validate Hetzner API response for server type while calculating resource requirements and for networks while validating cloud spec ([#12716](https://github.com/kubermatic/kubermatic/pull/12716))

### CNIs

#### Cilium

- Add Cilium 1.13.7 & 1.14.2 as supported CNI versions, deprecate older Cilium versions 1.13.x as versions are impacted by CVE-2023-39347, CVE-2023-41333 (Moderate Severity), CVE-2023-41332 (Low Severity) ([#12670](https://github.com/kubermatic/kubermatic/pull/12670))
- Update Cilium v1.11 and v1.12 patch releases to v1.11.20 and v1.12.13 ([#12561](https://github.com/kubermatic/kubermatic/pull/12561))
- Remove and replace deprecated `clusterPoolIPv4PodCIDR` and `clusterPoolIPv6PodCIDR` Helm value with `clusterPoolIPv4PodCIDRList` and `clusterPoolIPv6PodCIDRList` for Cilium 1.13+ ([#12561](https://github.com/kubermatic/kubermatic/pull/12561))

#### Canal

- Add support for Canal v3.26.1 ([#12561](https://github.com/kubermatic/kubermatic/pull/12561))
- Deprecate Canal v3.23 ([#12561](https://github.com/kubermatic/kubermatic/pull/12561))
- Mark all Canal CRDs with preserveUnknownFields: false ([#12538](https://github.com/kubermatic/kubermatic/pull/12538))

### KubeLB (Enterprise Edition only)

- Add KubeLB integration with KKP; introduce KubeLB as a first-class citizen in KKP ([#12667](https://github.com/kubermatic/kubermatic/pull/12667))
- Extend cluster health status with KubeLB health check ([#12685](https://github.com/kubermatic/kubermatic/pull/12685))
- Support for enforcing KubeLB at the datacenter level ([#12685](https://github.com/kubermatic/kubermatic/pull/12685))
- Support to configure node address type for KubeLB at the datacenter level ([#12715](https://github.com/kubermatic/kubermatic/pull/12715))

### Metering (Enterprise Edition only)

- Update metering to v1.1.0. Following fields are removed: ([#12545](https://github.com/kubermatic/kubermatic/pull/12545))
  - Cluster reports
    - Removal of `total-used-cpu-seconds`, use `average-used-cpu-millicores` instead
    - Removal of `average-available-cpu-cores`, use `average-available-cpu-millicores` instead
  - Namespace reports
    - Removal of `total-used-cpu-seconds`, use `average-used-cpu-millicores` instead 
- Add `monthly` parameter for metering monthly report generation ([#12544](https://github.com/kubermatic/kubermatic/pull/12544))

### MLA

- Mark MLA Grafana dashboards as non-editable as they are managed by KKP ([#12626](https://github.com/kubermatic/kubermatic/pull/12626))
- Fix configuration live reload for monitoring-agent and logging-agent ([#12507](https://github.com/kubermatic/kubermatic/pull/12507))
- Grafana Kubernetes dashboard will not repeatedly ask to be saved ([#12614](https://github.com/kubermatic/kubermatic/pull/12614))
- Replace `irate` with `rate` for node cpu usage graphs ([#12427](https://github.com/kubermatic/kubermatic/pull/12427))
- The `kube_service_labels` metric was not scraped with all expected labels, due to a change in labels on the kube-state-metrics service. The related scraping config was adapted accordingly ([#12551](https://github.com/kubermatic/kubermatic/pull/12551))
- Fix default url configuration of Blacbox exporter ([#12412](https://github.com/kubermatic/kubermatic/pull/12412))
- Fix several Prometheus record and alert rules ([#12533](https://github.com/kubermatic/kubermatic/pull/12533))
- Made Prometheus Helm chart extensible so that external metric storage solutions like Thanos can be easily integrated for seed long-term monitoring ([#12425](https://github.com/kubermatic/kubermatic/pull/12425))
- Fixes for the Kubernetes overview dashboard in Grafana ([#12520](https://github.com/kubermatic/kubermatic/pull/12520))

### New Features

- EE: Default ApplicationCatalog can be deployed via `--deploy-default-app-catalog` flag ([#12623](https://github.com/kubermatic/kubermatic/pull/12623))
- Add `disableCsiDriver` as optional field on `Cluster` and `Seed` resources to disable CSI driver deployment. This can be configured at a user cluster and datacenter level. If the admin disables CSI drivers at a datacenter level then the user is prohibited from enabling them at the user cluster level ([#12515](https://github.com/kubermatic/kubermatic/pull/12515))
- Introduce `DisableAdminKubeconfig` flag in `KubermaticSettings` to disable the admin kubeconfig feature from dashboard ([#12679](https://github.com/kubermatic/kubermatic/pull/12679))
- Disabled CSI addon on user clusters where it was enabled & then disabled using `DisableCSIDriver` option. The CSI addon is removed only if the CSI drivers created by it are not in use ([#12621](https://github.com/kubermatic/kubermatic/pull/12621))
- Extend `kubermatic-installer mirror-images` command with an option to export a tarball instead of syncing to a remote repository. This can be helpful in airgapped scenarios ([#12613](https://github.com/kubermatic/kubermatic/pull/12613))
- Extend MinIO configuration options to allow enabling MinIO console access and exposing MinIO API and console via Ingress ([#12683](https://github.com/kubermatic/kubermatic/pull/12683))
- New configuration option for Dex (`oauth` chart): Allow modification of web frontend issuer ([#12608](https://github.com/kubermatic/kubermatic/pull/12608))
- Support for configuring IPFamilies and IPFamilyPolicy for nodeport-proxy ([#12472](https://github.com/kubermatic/kubermatic/pull/12472))
- Support for configuring OIDC username and group prefix for user clusters ([#12648](https://github.com/kubermatic/kubermatic/pull/12648))
- Support for configuring the Dex theme via values file ([#12560](https://github.com/kubermatic/kubermatic/pull/12560))
- Switch backup containers to use `etcd-launcher snapshot` for creating etcd database snapshots ([#12462](https://github.com/kubermatic/kubermatic/pull/12462))
- Use OCI VM images as preconfigured default for local KubeVirt setup ([#12534](https://github.com/kubermatic/kubermatic/pull/12534))
- Allow to modify allocation range in IPAM Pools ([#12423](https://github.com/kubermatic/kubermatic/pull/12423))

### Bugfixes

- Add missing cluster-autoscaler release for user clusters using Kubernetes 1.27 ([#12597](https://github.com/kubermatic/kubermatic/pull/12597))
- Add missing images from envoy-agent `DaemonSet` in Tunneling expose strategy when running `kubermatic-installer mirror-images` ([#12537](https://github.com/kubermatic/kubermatic/pull/12537))
- Fix always defaulting allowed node port IP ranges for user clusters to 0.0.0.0/0 and ::/0, even when a more specific IP range was given ([#12589](https://github.com/kubermatic/kubermatic/pull/12589))
- Fix an issue in Applications, which resulted in "empty git-upload-pack given" errors for git sources ([#12487](https://github.com/kubermatic/kubermatic/pull/12487))
- Fix an issue in the `kubermatic-installer mirror-images` command, which led to failure on the mla-consul chart ([#12513](https://github.com/kubermatic/kubermatic/pull/12513))
- Fix an issue where IPv6 IPs were being ignored when determining the address of a user cluster ([#12505](https://github.com/kubermatic/kubermatic/pull/12505))
- Fix node-labeller controller not applying the `x-kubernetes.io/distribution` label to RHEL nodes ([#12751](https://github.com/kubermatic/kubermatic/pull/12751))
- Fix reconcile loop for `seed-proxy-token` Secret on Kubernetes 1.27 ([#12557](https://github.com/kubermatic/kubermatic/pull/12557))
- Increase memory limit of kube-state-metrics addon to 600Mi ([#12692](https://github.com/kubermatic/kubermatic/pull/12692))
- KKP's CA bundle was not used when performing Application-related operations like installing a Helm chart from a private OCI registry ([#12514](https://github.com/kubermatic/kubermatic/pull/12514))
- `kubermatic-installer` will now validate the existing MinIO filesystem before attempting a `kubermatic-seed` stack installation ([#12477](https://github.com/kubermatic/kubermatic/pull/12477))

### Updates

- Update Vertical Pod Autoscaler to 0.14.0 ([#12604](https://github.com/kubermatic/kubermatic/pull/12604))
- Update `d3fk/s3cmd` to version (latest "arch-stable") with `fb4c4dcf` hash ([#12640](https://github.com/kubermatic/kubermatic/pull/12640))
- Update cert-manager to 1.12.2 ([#12443](https://github.com/kubermatic/kubermatic/pull/12443))
- Update curl in `kubermatic/util` image and `mla/grafana` chart to 8.4.0 (CVE-2023-38545 and CVE-2023-38546 do not affect KKP) ([#12694](https://github.com/kubermatic/kubermatic/pull/12694))
- Update `quay.io/kubermatic/util` (helper image) to 2.3.1 (includes curl version patched against CVE-2023-38545 and CVE-2023-38546) ([#12726](https://github.com/kubermatic/kubermatic/pull/12726))
- Update etcd for user clusters to 3.5.9 ([#12453](https://github.com/kubermatic/kubermatic/pull/12453))
- Update KubeVirt chart for the installer local command to 1.0.0 ([#12470](https://github.com/kubermatic/kubermatic/pull/12470))
- Update metering Prometheus to next LTS version 2.45.0 ([#12532](https://github.com/kubermatic/kubermatic/pull/12532))
- Update metrics-server for all deployments to 0.6.4 ([#12516](https://github.com/kubermatic/kubermatic/pull/12516))
- Update nginx-ingress-controller to 1.9.3 (fixes CVE-2023-44487, HTTP/2 rapid reset attack) ([#12712](https://github.com/kubermatic/kubermatic/pull/12712))
- Update supported Kubernetes releases for EKS/AKS ([#12579](https://github.com/kubermatic/kubermatic/pull/12579))
- Update telemetry-agent to 0.4.1 ([#12572](https://github.com/kubermatic/kubermatic/pull/12572))
- Update controller-runtime to 0.16.1 and Kubernetes libraries to 1.28 ([#12609](https://github.com/kubermatic/kubermatic/pull/12609))
- Update Go to 1.21.3 ([#12697](https://github.com/kubermatic/kubermatic/pull/12697))
- Update KubeVirt CDI for local installer to 1.57.0 ([#12605](https://github.com/kubermatic/kubermatic/pull/12605))

### Miscellaneous

- Use `etcd-launcher` to check if etcd is running before starting kube-apiserver and to defragment etcd clusters ([#12450](https://github.com/kubermatic/kubermatic/pull/12450))
- Create a `NetworkPolicy` for user cluster kube-apiserver to access the Seed Kubernetes API ([#12569](https://github.com/kubermatic/kubermatic/pull/12569))
- Improve `http-prober` performance in user clusters with a lot of CRDs ([#12634](https://github.com/kubermatic/kubermatic/pull/12634))
