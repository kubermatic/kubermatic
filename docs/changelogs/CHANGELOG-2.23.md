# Kubermatic 2.23

- [v2.23.0](#v2230)
- [v2.23.1](#v2231)
- [v2.23.2](#v2232)
- [v2.23.3](#v2233)
- [v2.23.4](#v2234)
- [v2.23.5](#v2235)
- [v2.23.6](#v2236)
- [v2.23.7](#v2237)
- [v2.23.8](#v2238)
- [v2.23.9](#v2239)
- [v2.23.10](#v22310)
- [v2.23.11](#v22311)
- [v2.23.12](#v22312)
- [v2.23.13](#v22313)
- [v2.23.14](#v22314)
- [v2.23.15](#v22315)
- [v2.23.16](#v22316)
- [v2.23.17](#v22317)
- [v2.23.18](#v22318)

## v2.23.18

**GitHub release: [v2.23.18](https://github.com/kubermatic/kubermatic/releases/tag/v2.23.18)**

### Bugfixes

- Deduplicate alerts in alertmanager ([#13606](https://github.com/kubermatic/kubermatic/pull/13606))
- Fix KubermaticConfiguration getting deleted when a Seed on a shared master/seed cluster is deleted ([#13585](https://github.com/kubermatic/kubermatic/pull/13585))
- Fix usercluster-ctrl-mgr spamming oldest node version in its logs ([#13440](https://github.com/kubermatic/kubermatic/pull/13440))
- Restore missing bgpconfigurations CRD in Canal 3.27 ([#13505](https://github.com/kubermatic/kubermatic/pull/13505))

### Miscellaneous

- Add the label `name: nodeAgent` to the Velero daemon set pods ([#13538](https://github.com/kubermatic/kubermatic/pull/13538))
- The secret `velero-restic-credentials` is renamed to `velero-repo-credentials` ([#13538](https://github.com/kubermatic/kubermatic/pull/13538))

### Updates

- Update machine-controller to v1.57.9 ([#13561](https://github.com/kubermatic/kubermatic/pull/13561))

## v2.23.17

**GitHub release: [v2.23.17](https://github.com/kubermatic/kubermatic/releases/tag/v2.23.17)**

### Bugfixes

- Fix the pagination in project members table ([#6744](https://github.com/kubermatic/dashboard/pull/6744))


## v2.23.16

**GitHub release: [v2.23.16](https://github.com/kubermatic/kubermatic/releases/tag/v2.23.16)**

### Bugfixes

- Add `displayName` and `scope` columns for printing the cluster templates; `kubectl get clustertemplates` will now show the actual display name and scope for the cluster templates ([#13419](https://github.com/kubermatic/kubermatic/pull/13419))
- Fix a bug where unrequired `cloud-config` secret was being propagated to the user clusters ([#13373](https://github.com/kubermatic/kubermatic/pull/13373))
- Fix: use correct networkpolicy port for metrics-server ([#13447](https://github.com/kubermatic/kubermatic/pull/13447))

### Updates

- Update machine-controller to v1.57.8, fixing support for Rockylinux 8 on AWS ([#13431](https://github.com/kubermatic/kubermatic/pull/13431))
- Update OSM to v1.3.6; fixing cloud-init bootstrapping issues on Ubuntu 22.04 on Azure ([#13379](https://github.com/kubermatic/kubermatic/pull/13379))



## [v2.23.15](https://github.com/kubermatic/kubermatic/releases/tag/v2.23.15)

### Bugfixes

* [ACTION REQUIRED] The latest Ubuntu 22.04 images ship with cloud-init 24.x package. This package has breaking changes and thus rendered our OSPs as incompatible. It's recommended to refresh your machines with latest provided OSPs to ensure that a system-wide package update, that updates cloud-init to 24.x, doesn't break the machines ([#13359](https://github.com/kubermatic/kubermatic/pull/13359))

### Updates

* Update operating-system-manager to v1.3.5.


## [v2.23.14](https://github.com/kubermatic/kubermatic/releases/tag/v2.23.14)

### New Feature

- Seed MLA: introduce `signout_redirect_url` field to configure the URL to redirect the user to after signing out from Grafana ([#13313](https://github.com/kubermatic/kubermatic/pull/13313))

### Bugfixes

- Enable `local` command for Enterprise Edition ([#13333](https://github.com/kubermatic/kubermatic/pull/13333))
- Fix template value for MachineDeployments in edit mode ([#6669](https://github.com/kubermatic/dashboard/pull/6669))
- Hotfix to mitigate a bug in new releases of Chromium that causes browser crashes on `mat-select` component. For more details: https://issuetracker.google.com/issues/335553723 ([#6667](https://github.com/kubermatic/dashboard/pull/6667))
- Improve Helm repository prefix handling for system applications; only prepend `oci://` prefix if it doesn't already exist in the specified URL ([#13343](https://github.com/kubermatic/kubermatic/pull/13343))
- Installer does not validate IAP `client_secrets` for Grafana and Alertmanager the same way it does for `encryption_key` ([#13315](https://github.com/kubermatic/kubermatic/pull/13315))

### Chore

- Update machine-controller to v1.57.7 ([#13347](https://github.com/kubermatic/kubermatic/pull/13347))


## [v2.23.13](https://github.com/kubermatic/kubermatic/releases/tag/v2.23.13)

### API Changes

- Add `spec.componentsOverride.operatingSystemManager` to allow overriding OSM settings and resources ([#13288](https://github.com/kubermatic/kubermatic/pull/13288))

### Bugfixes

- Fix high CPU usage in master-controller-manager ([#13217](https://github.com/kubermatic/kubermatic/pull/13217))

### Updates

- Add Canal CNI version v3.27.3 ([#13308](https://github.com/kubermatic/kubermatic/pull/13308))
- Add support for Kubernetes 1.27.13 (fixes CVE-2024-3177) ([#13300](https://github.com/kubermatic/kubermatic/pull/13300))


## [v2.23.12](https://github.com/kubermatic/kubermatic/releases/tag/v2.23.12)

### Bugfixes

- Exclude `test` folders which contain symlinks that break once the archive is untarred ([#13151](https://github.com/kubermatic/kubermatic/pull/13151))
- Fix a bug where OSPs were not being listed for VMware Cloud Director ([#6592](https://github.com/kubermatic/dashboard/pull/6592))
- Fix invalid project ID in API requests for Nutanix provider ([#6572](https://github.com/kubermatic/dashboard/pull/6572))
- Fix a bug where dedicated credentials were incorrectly being required as mandatory input when editing vSphere provider settings for a cluster ([#6567](https://github.com/kubermatic/dashboard/pull/6567))


### Chore
- Update to Go 1.20.13 ([#13165](https://github.com/kubermatic/kubermatic/pull/13165)) and ([#6594](https://github.com/kubermatic/dashboard/pull/6594))

## [v2.23.11](https://github.com/kubermatic/kubermatic/releases/tag/v2.23.11)

### Bugfixes

- **ACTION REQUIRED:** For velero helm chart upgrade related change. If you use `velero.restic.deploy: true`, you will see new daemonset `node-agent` running in `velero` namespace. You might need to remove existing daemonset named `restic` manually ([#12998](https://github.com/kubermatic/kubermatic/pull/12998))
- Fix panic, if no KubeVirt DNS config was set in the datacenter ([#13029](https://github.com/kubermatic/kubermatic/pull/13029))

### Updates

- Update metering to v1.0.6, fixing an error when a custom CA bundle is used ([#13012](https://github.com/kubermatic/kubermatic/pull/13012))
- Update operating-system-manager (OSM) to [v1.3.4](https://github.com/kubermatic/operating-system-manager/releases/tag/v1.3.4) ([#13083](https://github.com/kubermatic/kubermatic/pull/13083))
  - This includes a fix for Flatcar stable channel (version 3815.2.0) failing to provision as new nodes.

## [v2.23.10](https://github.com/kubermatic/kubermatic/releases/tag/v2.23.10)

### Bugfixes

- Stop constantly re-deploying operating-system-manager when registry mirrors are configured ([#12972](https://github.com/kubermatic/kubermatic/pull/12972))

### Updates

- Update EKS/AKS version matrix to only include Kubernetes versions supported by those managed offerings. For  AKS 1.26-1.28 are supported, for EKS 1.24 to 1.28. The default for newly created external clusters is now 1.28 ([#12964](https://github.com/kubermatic/kubermatic/pull/12964))
- Add support for Kubernetes v1.24.17, v1.25.16, v1.26.13, v1.27.10 and set default version to v1.26.13 ([#12983](https://github.com/kubermatic/kubermatic/pull/12983))

## [v2.23.9](https://github.com/kubermatic/kubermatic/releases/tag/v2.23.9)

### Bugfixes

- Applied a fix to VPA caused by [upstream release issue](https://github.com/kubernetes/autoscaler/issues/5982) which caused insufficient RBAC permission for VPA recommender pod ([#12872](https://github.com/kubermatic/kubermatic/pull/12872))
- Fix cert-manager values block. cert-manager deployment will get updated as part of upgrade ([#12854](https://github.com/kubermatic/kubermatic/pull/12854))
- Fix cases where, when using dedicated infra- and ccm-credentials, infra-credentials were always overwritten by ccm-credentials ([#12421](https://github.com/kubermatic/kubermatic/pull/12421))
- No longer fail constructing vSphere endpoint when a `/` suffix is present in the datacenter configuration ([#12861](https://github.com/kubermatic/kubermatic/pull/12861))

### Updates

- Update machine-controller to [v1.57.4](https://github.com/kubermatic/machine-controller/releases/tag/v1.57.4) ([#12903](https://github.com/kubermatic/kubermatic/pull/12903))
- Update Anexia CCM (cloud-controller-manager) to version 1.5.5 ([#12910](https://github.com/kubermatic/kubermatic/pull/12910))
    - Fixes leaking LoadBalancer reconciliation metric
    - Updates various dependencies

### Miscellaneous

- KKP is now built with Go 1.20.12 ([#12900](https://github.com/kubermatic/kubermatic/pull/12900))
- Increase the default resources for VPA components to prevent OOMs ([#12887](https://github.com/kubermatic/kubermatic/pull/12887))

## [v2.23.8](https://github.com/kubermatic/kubermatic/releases/tag/v2.23.8)

### Dashboard

- Fix a bug where the API call to list projects was failing due to slowness ([#6385](https://github.com/kubermatic/dashboard/pull/6385))

## [v2.23.7](https://github.com/kubermatic/kubermatic/releases/tag/v2.23.7)

### Action Required

- **ACTION REQUIRED (EE ONLY):** Update metering component to v1.0.5, fixing highly inaccurate data in cluster reports. Reports generated in KKP v2.23.2+ or v2.22.5+ do not represent actual consumption. Ad-hoc reports for time frames that need correct consumption data can be generated [by following our documentation](https://docs.kubermatic.com/kubermatic/v2.23/tutorials-howtos/metering/#custom-reports) ([#12823](https://github.com/kubermatic/kubermatic/pull/12823))

### Bugfixes

- Extend project-synchronizer controller in `kubermatic-master-controller-manager` to propagate labels from Projects in the master cluster to Projects in the seed cluster. This fixes an issue where the metering report doesn't contain project-labels in separate master/seed setups ([#12792](https://github.com/kubermatic/kubermatic/pull/12792))
- Fix CPU Utilization graph showing no data for User Cluster MLA dashboard "Nodes Overview" ([#12814](https://github.com/kubermatic/kubermatic/pull/12814))
- Fix empty panels in Grafana dashboard "Resource Usage per Namespace" for Master/Seed MLA ([#12816](https://github.com/kubermatic/kubermatic/pull/12816))
- Fix Helm 3.13 failing to install the MLA Minio chart due to "resource name may not be empty" error ([#12806](https://github.com/kubermatic/kubermatic/pull/12806))

## [v2.23.6](https://github.com/kubermatic/kubermatic/releases/tag/v2.23.6)

### Bugfixes

- Fix Digitalocean CSI addon failing to render ([#12739](https://github.com/kubermatic/kubermatic/pull/12739))
- Fix node-labeller controller not applying the `x-kubernetes.io/distribution` label to RHEL nodes ([#12751](https://github.com/kubermatic/kubermatic/pull/12751))
- Increase default CPU limits for KKP API/seed/master-controller-managers to prevent general slowness ([#12764](https://github.com/kubermatic/kubermatic/pull/12764))

### Updates

- Add support for Cilium 1.13.8, mitigating an high CVE-2023-44487 ([#12762](https://github.com/kubermatic/kubermatic/pull/12762))

## [v2.23.5](https://github.com/kubermatic/kubermatic/releases/tag/v2.23.5)

### Bugfixes

- Correctly validate Hetzner API response for server type while calculating resource requirements and for networks while validating cloud spec ([#12716](https://github.com/kubermatic/kubermatic/pull/12716))

### Updates

- Update nginx-ingress-controller to v1.9.3 (fixes CVE-2023-44487, HTTP/2 rapid reset attack) ([#12714](https://github.com/kubermatic/kubermatic/pull/12714))
- Update to Go 1.20.10 ([#12698](https://github.com/kubermatic/kubermatic/pull/12698))
- Update to OSM [v1.3.3](https://github.com/kubermatic/operating-system-manager/releases/tag/v1.3.3) ([#12710](https://github.com/kubermatic/kubermatic/pull/12710))
- Add Cilium 1.13.7 as supported CNI version, deprecate cilium version 1.13.6 as it's impacted by CVE-2023-39347, CVE-2023-41333 (Moderate Severity), CVE-2023-41332 (Low Severity) ([#12695](https://github.com/kubermatic/kubermatic/pull/12695))
- Update to `quay.io/kubermatic/util:2.3.1` as helper image (includes curl version patched against CVE-2023-38545 and CVE-2023-38546) ([#12733](https://github.com/kubermatic/kubermatic/pull/12733))

### New Feature

- Introduce `DisableAdminKubeconfig` flag in `KubermaticSettings` to disable the admin kubeconfig feature from dashboard ([#12679](https://github.com/kubermatic/kubermatic/pull/12679))

## [v2.23.4](https://github.com/kubermatic/kubermatic/releases/tag/v2.23.4)

### Bugfixes

- Fix vSphere cluster validation: If a Cluster uses a custom datastore, the Seed's default datastore should not be validated ([#12655](https://github.com/kubermatic/kubermatic/pull/12655))
- Remove Cilium 1.14.1 from list of supported CNI versions visible in the dashboard as it is not supported in KKP 2.23 ([#12659](https://github.com/kubermatic/kubermatic/pull/12659))

## [v2.23.3](https://github.com/kubermatic/kubermatic/releases/tag/v2.23.3)

### Supported Kubernetes Versions

- Add support for Kubernetes 1.25.14, 1.26.9 and 1.27.6 ([#12639](https://github.com/kubermatic/kubermatic/pull/12639))
- Set default Kubernetes version to 1.26.9 ([#12639](https://github.com/kubermatic/kubermatic/pull/12639))

### Bugfixes

- Add missing cluster-autoscaler release for user clusters using Kubernetes 1.27 ([#12597](https://github.com/kubermatic/kubermatic/pull/12597))
- Fix always defaulting allowed node port IP ranges for user clusters to 0.0.0.0/0 and ::/0, even when a more specific IP range was given ([#12589](https://github.com/kubermatic/kubermatic/pull/12589))
- Mark MLA Grafana dashboards as non-editable as they are managed by KKP ([#12627](https://github.com/kubermatic/kubermatic/pull/12627))
- MLA Grafana Kubernetes dashboards won't repeatedly ask to be saved ([#12614](https://github.com/kubermatic/kubermatic/pull/12614))

### Updates

- Update `d3fk/s3cmd` to version (latest "arch-stable") with `fb4c4dcf` hash ([#12644](https://github.com/kubermatic/kubermatic/pull/12644))
- Update to Go 1.20.8 ([#12642](https://github.com/kubermatic/kubermatic/pull/12642))
- Add Cilium 1.13.6 as supported CNI version and deprecate older versions 1.13.3 and 1.13.4 for security reasons (GHSA-pvgm-7jpg-pw5g, GHSA-69vr-g55c-v2v4, GHSA-mc6h-6j9x-v3gq, GHSA-7mhv-gr67-hq55) ([#12635](https://github.com/kubermatic/kubermatic/pull/12635))
- Update Vertical Pod Autoscaler to 0.14 (compatible with Kubernetes 1.25+) ([#12611](https://github.com/kubermatic/kubermatic/pull/12611))

## [v2.23.2](https://github.com/kubermatic/kubermatic/releases/tag/v2.23.2)

### Bugfixes

- Add missing images from envoy-agent DaemonSet in Tunneling expose strategy when running `kubermatic-installer mirror-images` ([#12537](https://github.com/kubermatic/kubermatic/pull/12537))
- Fix an issue in the kubermatic-installer mirror-images command, which led to failure on the mla-consul chart ([#12513](https://github.com/kubermatic/kubermatic/pull/12513))
- Fix an issue in the kubermatic-installer mirror-images command, which led to failure on the mla-consul chart ([#12518](https://github.com/kubermatic/kubermatic/pull/12518))
- Fix an issue where IPv6 IPs were being ignored when determining the address of a user cluster ([#12511](https://github.com/kubermatic/kubermatic/pull/12511))
- Fix reconcile loop for `seed-proxy-token` Secret on Kubernetes 1.27 ([#12566](https://github.com/kubermatic/kubermatic/pull/12566))
- Mark all canal CRDs with `preserveUnknownFields: false` ([#12549](https://github.com/kubermatic/kubermatic/pull/12549))
- MLA: fixes configuration live reload for monitoring-agent and logging-agent ([#12507](https://github.com/kubermatic/kubermatic/pull/12507))
- MLA: fixes for the kubernetes overview dashboard in grafana ([#12520](https://github.com/kubermatic/kubermatic/pull/12520))
- The kube_service_labels metric was not scraped with all expected labels, due to a change in labels on the kube-state-metrics service. The related scraping config was adapted accordingly ([#12551](https://github.com/kubermatic/kubermatic/pull/12551))
- VSphere: Fix a bug where datastore cluster value was not being propagated to the CSI driver ([#12474](https://github.com/kubermatic/kubermatic/pull/12474))

### Updates

- Update machine-controller to v1.57.3 and OSM to v1.3.2 ([#12577](https://github.com/kubermatic/kubermatic/pull/12577))
- Update metering to v1.0.4 with increased namespace report generation performance and prometheus to v2.37.9 ([#12546](https://github.com/kubermatic/kubermatic/pull/12546))
- Update operating-system-manager (OSM) to [v1.3.1](https://github.com/kubermatic/operating-system-manager/releases/tag/v1.3.1) ([#12564](https://github.com/kubermatic/kubermatic/pull/12564))
- Update telemetry-agent to v0.4.1 ([#12572](https://github.com/kubermatic/kubermatic/pull/12572))

### New Feature

- Support for configuring the dex theme via values file ([#12560](https://github.com/kubermatic/kubermatic/pull/12560))

## [v2.23.1](https://github.com/kubermatic/kubermatic/releases/tag/v2.23.1)

### Features

- Made Prometheus helm chart extensible so that external metric storage solutions like Thanos can be easily integrated for seed long-term monitoring ([#12469](https://github.com/kubermatic/kubermatic/pull/12469))

### Bugfixes

- Fix default url configuration of blackbox exporter ([#12412](https://github.com/kubermatic/kubermatic/pull/12412))
- Hetzner CSI: recreate CSIDriver to allow upgrade from 1.6.0 to 2.2.0 ([#12432](https://github.com/kubermatic/kubermatic/pull/12432))
- Replace `irate` with `rate` for node cpu usage graphs ([#12427](https://github.com/kubermatic/kubermatic/pull/12427))
- The Kubermatic Installer will now validate the existing Minio filesystem before attempting a `kubermatic-seed` stack installation ([#12493](https://github.com/kubermatic/kubermatic/pull/12493))

### Updates

- Update to Go 1.20.6 ([#12502](https://github.com/kubermatic/kubermatic/pull/12502))
- Update Cilium CNI to 1.13.4, marking 1.13.0 as deprecated but kept 1.13.3 because 1.13.4 breaks IPSec support ([#12478](https://github.com/kubermatic/kubermatic/pull/12478))
- Update machine-controller to [v1.57.1](https://github.com/kubermatic/machine-controller/releases/tag/v1.57.1) ([#12492](https://github.com/kubermatic/kubermatic/pull/12492))

### Misc

- Support for configuring multiple networks for vSphere ([#12458](https://github.com/kubermatic/kubermatic/pull/12458))
- Support for configuring IPFamilies and IPFamilyPolicy for nodeport-proxy ([#12472](https://github.com/kubermatic/kubermatic/pull/12472))

## [v2.23.0](https://github.com/kubermatic/kubermatic/releases/tag/v2.23.0)

Before upgrading, make sure to read the [general upgrade guidelines](https://docs.kubermatic.com/kubermatic/v2.23/installation/upgrading/). Consider tweaking `seedControllerManager.maximumParallelReconciles` to ensure user cluster reconciliations will not cause resource exhaustion on seed clusters. A [full upgrade guide is available from the official documentation](https://docs.kubermatic.com/kubermatic/v2.23/installation/upgrading/upgrade-from-2.22-to-2.23/).

### Breaking Changes

- Move to Egress based cluster isolation network policies for KubeVirt ([#12329](https://github.com/kubermatic/kubermatic/pull/12329))
  - **ACTION REQUIRED:** Custom Network policies for KubeVirt datacenters might need adjustment
- The `kubermatic-installer` now recognizes CSIDrivers automatically and will use them when creating the `kubermatic-fast` StorageClass. Admins can still choose to simply copy the default StorageClass if it's heavily customized by continuing to specify `--storageclass copy-default` ([#12012](https://github.com/kubermatic/kubermatic/pull/12012))
  - **ACTION REQUIRED:** The flag value `gce` was renamed to `gcp` for `--storageclass`
- Introduce `EnableShareCluster` flag in `KubermaticSettings` to toggle the share cluster feature for the dashboard ([#11950](https://github.com/kubermatic/kubermatic/pull/11950))
  - **ACTION REQUIRED:** `share_kubeconfig` field in the UI configuration for KubermaticConfiguration has been replaced with `EnableShareCluster` flag in KubermaticSettings. `share_kubeconfig` is no-op and will be ignored by the dashboard

### Known Issues

The following issues have been identified and will be fixed in the upcoming patch releases.

- CSI addon for Hetzner fails to apply after upgrade ([12429](https://github.com/kubermatic/kubermatic/issues/12429))
  - **REMEDIATION:** A workaround is to manually delete the `CSIDriver` and let the `addon-controller` reconcile it - `kubectl delete csidriver csi.hetzner.cloud`.
- Crashing MinIO after upgrade ([12430](https://github.com/kubermatic/kubermatic/issues/12430))
  - **REMEDIATION:** A workaround is to downgrade Minio to the last release supporting the `fs` storage driver. You can pin the `minio` image `tag` to `RELEASE.2022-10-24T18-35-07Z` in the [`values.yaml`](https://github.com/kubermatic/kubermatic/blob/v2.23.0/charts/minio/values.yaml#L24) and re-run the installer.

### Security

- Fix potential path traversal in mirror-images command ([#12293](https://github.com/kubermatic/kubermatic/pull/12293))

### API Changes

- Add short name for  Application  CRDs ([#12017](https://github.com/kubermatic/kubermatic/pull/12017))
    - `applicationdefinition` -> `appdef`, e.g `kubectl get appdef`
    - `applicationinstallation` -> `appinstall`, e.g `kubectl get appinstall`
- Support added to specify the suffix `dockerTagSuffix` in `KubermaticConfiguration` for dashboard images. With `dockerTagSuffix` the tag becomes <CURRENT_KKP_VERSION:SUFFIX> i.e. "v2.15.0-SUFFIX" ([#12056](https://github.com/kubermatic/kubermatic/pull/12056))
- Add support for disabling Changelog popup in `KubermaticSettings` ([#12175](https://github.com/kubermatic/kubermatic/pull/12175))
- Add support for enforcing/enabling auto-updates and updates on first boot for Machine Deployments in `KubermaticSettings` ([#12152](https://github.com/kubermatic/kubermatic/pull/12152))
- Add `componentOverride.userClusterController` to `Cluster` and `ClusterTemplate` resources to configure the `usercluster-controller` Deployment for each user cluster ([#12211](https://github.com/kubermatic/kubermatic/pull/12211))
- Revert CRD split between master and seed by installing all CRDs on the master again ([#12282](https://github.com/kubermatic/kubermatic/pull/12282))
- Add component override settings for etcd that allow configuring the type of anti-affinity ([#12313](https://github.com/kubermatic/kubermatic/pull/12313))

### Supported Kubernetes Versions

- Add support for Kubernetes 1.24.13, 1.25.9 and 1.26.4 ([#12165](https://github.com/kubermatic/kubermatic/pull/12165))
- Add support for Kubernetes 1.27 ([#12230](https://github.com/kubermatic/kubermatic/pull/12230))
- Remove auto-upgrade rule for user clusters from 1.23 to 1.24. All user clusters must be migrated to Kubernetes 1.24 before updating to KKP 2.23 ([#12280](https://github.com/kubermatic/kubermatic/pull/12280))
- Add support for Kubernetes 1.24.15, 1.25.11, 1.26.6 and 1.27.3 (fixing CVE-2023-2431, CVE-2023-2727 and CVE-2023-2728) ([#12374](https://github.com/kubermatic/kubermatic/pull/12374))
- Set default Kubernetes version to 1.26.6 ([#12374](https://github.com/kubermatic/kubermatic/pull/12374))
- Do not allow Kubernetes >= 1.27 with in-tree CCM on AWS ([#12417](https://github.com/kubermatic/kubermatic/pull/12417))

#### Supported Versions

- 1.24.8
- 1.24.9
- 1.24.10
- 1.24.13
- 1.24.15
- 1.25.4
- 1.25.5
- 1.25.6
- 1.25.9
- 1.25.11
- 1.26.1
- 1.26.4
- 1.26.6 (default)
- 1.27.3

### Cloud Providers

#### AWS

- Update AWS CCM for Kubernetes 1.25 to 1.25.3 ([#11967](https://github.com/kubermatic/kubermatic/pull/11967))
- Update AWS Node Termination Handler to 1.19.0 ([#11967](https://github.com/kubermatic/kubermatic/pull/11967))
- Update AWS EBS CSI to 2.18.0 ([#12227](https://github.com/kubermatic/kubermatic/pull/12227))
- Update AWS CCM to 1.26.1 / 1.27.1 ([#12227](https://github.com/kubermatic/kubermatic/pull/12227))

#### Azure

- Update Azure Cloud Node Manager to 1.24.18 / 1.25.12 / 1.26.8 ([#12222](https://github.com/kubermatic/kubermatic/pull/12222))
- Update Azure Disk CSI to 1.27.1 ([#12222](https://github.com/kubermatic/kubermatic/pull/12222))
- Update Azure File CSI to 1.27.0 ([#12222](https://github.com/kubermatic/kubermatic/pull/12222))
- Update Azure CCM to 1.24.18 / 1.25.12 / 1.26.8 / 1.27.1 ([#12222](https://github.com/kubermatic/kubermatic/pull/12222))

#### vSphere

- Fix a bug where KKP managed vSphere folders are enforced but shouldn't ([#11962](https://github.com/kubermatic/kubermatic/pull/11962))
- Update vSphere CCM/CSI to 1.23.4 / 1.24.5 / 1.25.2 / 1.26.1 ([#12229](https://github.com/kubermatic/kubermatic/pull/12229))

#### VMware Cloud Director

- Update VMware Cloud Director CSI driver to 1.3.2 ([#12096](https://github.com/kubermatic/kubermatic/pull/12096))
- VMware Cloud Director now supports authentication using API Token ([#12124](https://github.com/kubermatic/kubermatic/pull/12124))

#### OpenStack

- Update external-snapshotter validation webhook server to v6.0.1 ([#12120](https://github.com/kubermatic/kubermatic/pull/12120))
- Addons: openstack: service account for CSI snapshot webhook server ([#12201](https://github.com/kubermatic/kubermatic/pull/12201))
- Bugfix: don't override floating IP settings from user input for OpenStack initial MD ([#12261](https://github.com/kubermatic/kubermatic/pull/12261))
- Update OpenStack CCM/CSI to 1.25.5 / 1.26.2. Container images are now using `registry.k8s.io` instead of `docker.io` ([#12228](https://github.com/kubermatic/kubermatic/pull/12228))
- Fix storage calculation for Openstack resource quota when custom disk size is provided ([#12370](https://github.com/kubermatic/kubermatic/pull/12370))

#### KubeVirt

- Add option to disable deployment of default network policies in KubeVirt cluster ([#12082](https://github.com/kubermatic/kubermatic/pull/12082))

#### DigitalOcean

- Update Digitalocean CCM to 0.1.42 ([#11982](https://github.com/kubermatic/kubermatic/pull/11982))

#### Anexia

- Update Anexia CCM (cloud-controller-manager) to version 1.5.4 ([#12212](https://github.com/kubermatic/kubermatic/pull/12212))

#### Hetzner

- Update Hetzner CCM to 1.15.0 ([#12191](https://github.com/kubermatic/kubermatic/pull/12191))
- Update Hetzner CSI to 2.3.2 ([#12191](https://github.com/kubermatic/kubermatic/pull/12191))

### CNIs

#### Calico

- Add support for Canal 3.25 ([#12297](https://github.com/kubermatic/kubermatic/pull/12297))
- Deprecate Canal 3.22 and enforce update for Canal below 3.22 on Kubernetes 1.25 and above ([#12347](https://github.com/kubermatic/kubermatic/pull/12347), [#12403](https://github.com/kubermatic/kubermatic/pull/12403))

#### Cilium

- Set proper NodePort range in Cilium config if non-default range is used ([#11963](https://github.com/kubermatic/kubermatic/pull/11963))
- Update Cilium versions to 1.12.9 and 1.11.16 ([#12264](https://github.com/kubermatic/kubermatic/pull/12264))
- Add support for Cilium 1.13.3 as user cluster CNI ([#12199](https://github.com/kubermatic/kubermatic/pull/12199), [#12320](https://github.com/kubermatic/kubermatic/pull/12320))

### Installer

- Add `--skip-charts` flag to `kubermatic-installer deploy` command to make helm chart deployment skippable ([#12059](https://github.com/kubermatic/kubermatic/pull/12059))
- Include etcd-launcher and Gatekeeper images in `kubermatic-installer mirror-images` ([#12130](https://github.com/kubermatic/kubermatic/pull/12130))
- `--mla-skip-minio` and `--mla-skip-minio-lifecycle-mgr` for `kubermatic-installer deploy usercluster-mla` work properly now ([#12140](https://github.com/kubermatic/kubermatic/pull/12140))
- Include metering images in `kubermatic-installer mirror-images` (EE) ([#12144](https://github.com/kubermatic/kubermatic/pull/12144))
- Add experimental `kubermatic-installer local` command to spin up a local KKP environment ([#12216](https://github.com/kubermatic/kubermatic/pull/12216))
- Add support for `oidc` authentication in kubeconfigs passed to `kubermatic-installer` ([#12252](https://github.com/kubermatic/kubermatic/pull/12252))

### MLA

- Fix mla-monitoring-agent configuration being invalid when custom scraping configuration is provided ([#11988](https://github.com/kubermatic/kubermatic/pull/11988))
- Enable Loki Compactor rotation and set retention to 1 month by default ([#12029](https://github.com/kubermatic/kubermatic/pull/12029))
- Fix calculation of node CPU utilisation in Grafana dashboards for multi-core nodes ([#12034](https://github.com/kubermatic/kubermatic/pull/12034))
- Disable PodSecurityPolicy in MLA Grafana deployment ([#12101](https://github.com/kubermatic/kubermatic/pull/12101))
- Fix MLA stack constantly updating Grafana datasources ([#12182](https://github.com/kubermatic/kubermatic/pull/12182))
- The MLA stack is now able to recover from a lost Grafana volume, properly recreating organizations for KKP projects ([#12195](https://github.com/kubermatic/kubermatic/pull/12195))
- User Cluster MLA Alertmanager now allows blackbox exporter to perform healthcheck API call without AuthFailure ([#12217](https://github.com/kubermatic/kubermatic/pull/12217))
- Add a new controller-runtime metrics dashboard in grafana to the monitoring chart ([#12257](https://github.com/kubermatic/kubermatic/pull/12257))
- Add monitoring and dashboard for envoy-agent and nodeport-proxy ([#12302](https://github.com/kubermatic/kubermatic/pull/12302))
- Limit EtcdDatabaseHighFragmentationRatio rule to avoid triggering excessively for small etcd instances ([#12305](https://github.com/kubermatic/kubermatic/pull/12305))
- Add new alert `NodeTimeDrift` ([#12275](https://github.com/kubermatic/kubermatic/pull/12275))
- Add `KubermaticSeedNotHealthy` alert if a Seed is not healthy ([#12194](https://github.com/kubermatic/kubermatic/pull/12194))

### Metering (EE)

- Add support for ca-bundle to metering cronjobs ([#11979](https://github.com/kubermatic/kubermatic/pull/11979))
- Update Metering to v1.0.3 ([#12035](https://github.com/kubermatic/kubermatic/pull/12035))
    - Add non machine-controller managed machines to `average-cluster-machines`. Note that this is based on a new metric that will be collected together in the same release, therefore information prior this update is not available
    - Fixes a bug that leads to low CPU usage values* Remove redundant label quotation
- Fix metering CronJobs after KKP upgrades ([#12139](https://github.com/kubermatic/kubermatic/pull/12139))
- Fix a bug that lead to metering reports overwriting each other when used with multiple seeds. Report names now include the Seed name as a Prefix ([#12221](https://github.com/kubermatic/kubermatic/pull/12221))

### Bugfixes

- Fix worker-name handing in resource-quota updates (EE) ([#11943](https://github.com/kubermatic/kubermatic/pull/11943))
- An internal NetworkPolicy for apiserver communication is now being created and the previous NetworkPolicy `cluster-external-addr-allow` is cleaned up ([#12348](https://github.com/kubermatic/kubermatic/pull/12348))
- Fix OOM on usercluster-controller by limiting the history of Helm releases for Applications ([#12089](https://github.com/kubermatic/kubermatic/pull/12089))
- Do not try to watch `Cluster` resources on the master in `usersshkey-synchronizer` and use Seeds as correct source instead ([#12271](https://github.com/kubermatic/kubermatic/pull/12271))
- Fix a bug that causes dedicated Seeds to be stuck in deletion ([#12131](https://github.com/kubermatic/kubermatic/pull/12131))
- Fix wrong labels in cluster/project metrics when uppercase labels were used ([#11947](https://github.com/kubermatic/kubermatic/pull/11947))
- Metrics server write timeout increased ([#12314](https://github.com/kubermatic/kubermatic/pull/12314))
- Pull `kas-network-proxy/proxy-server:v0.0.35` and `kas-network-proxy/proxy-agent:v0.0.35` image from `registry.k8s.io` instead of legacy GCR registry (`eu.gcr.io/k8s-artifacts-prod`) ([#12067](https://github.com/kubermatic/kubermatic/pull/12067))
- Support for configuring additional volumes for the UI ([#12103](https://github.com/kubermatic/kubermatic/pull/12103))
- The kubeconfig used by konnectivity's server component gets renewed automatically now, no longer causing konnectivity to stop working when the embedded certificate expires ([#12344](https://github.com/kubermatic/kubermatic/pull/12344))
- Use seed proxy configuration for seed deployed webhook ([#12070](https://github.com/kubermatic/kubermatic/pull/12070))
- Use serializable etcd liveness probes and add a startup probe, as per upstream recommendations ([#12190](https://github.com/kubermatic/kubermatic/pull/12190))
- The validating webhook for `Cluster` resources now properly checks for provider incompatibilities ([#11996](https://github.com/kubermatic/kubermatic/pull/11996))
- nginx-ingress-controller: set default memory limit to 1Gi ([#12411](https://github.com/kubermatic/kubermatic/pull/12411))

### Updates

- Update machine-controller to [1.57.0](https://github.com/kubermatic/machine-controller/releases/tag/v1.57.0) ([#12390](https://github.com/kubermatic/kubermatic/pull/12390))
- Update KubeOne to [1.6.2](https://github.com/kubermatic/kubeone/releases/tag/v1.6.2) ([#12390](https://github.com/kubermatic/kubermatic/pull/12390))
- Update operating-system-manager (OSM) to [1.3.0](https://github.com/kubermatic/operating-system-manager/releases/tag/v1.3.0) ([#12410](https://github.com/kubermatic/kubermatic/pull/12410))
- Update Alertmanager to 0.25.0 ([#12237](https://github.com/kubermatic/kubermatic/pull/12237))
- Update blackbox-exporter to 0.23.0 ([#12235](https://github.com/kubermatic/kubermatic/pull/12235))
- Update cert-manager to 1.11.1 ([#12243](https://github.com/kubermatic/kubermatic/pull/12243))
- Update cluster-autoscaler to 1.24.1 / 1.25.1 / 1.26.2 ([#12223](https://github.com/kubermatic/kubermatic/pull/12223))
- Update configmap-reload to 0.8.0 ([#12238](https://github.com/kubermatic/kubermatic/pull/12238))
- Update Dex to 2.36.0 ([#12233](https://github.com/kubermatic/kubermatic/pull/12233))
- Update Envoy to 1.26.1 ([#12246](https://github.com/kubermatic/kubermatic/pull/12246))
- Update etcd-backup Minio to RELEASE.2023-05-04T21-44-30Z, change image to `quay.io/minio/minio` ([#12241](https://github.com/kubermatic/kubermatic/pull/12241))
- Update Gatekeeper to 3.12.0 ([#12260](https://github.com/kubermatic/kubermatic/pull/12260))
- Update Grafana to 9.5.1 ([#12240](https://github.com/kubermatic/kubermatic/pull/12240))
- Update helm-exporter to 1.2.5 ([#12239](https://github.com/kubermatic/kubermatic/pull/12239))
- Update IAP (oauth2-proxy) to 7.4.0 ([#12242](https://github.com/kubermatic/kubermatic/pull/12242))
- Update k8s-dns-node-cache to 1.22.20 ([#12245](https://github.com/kubermatic/kubermatic/pull/12245))
- Update Karma to 0.114 ([#12236](https://github.com/kubermatic/kubermatic/pull/12236))
- Update konnectivity proxy-agent/server to 0.0.37 for user clusters using Kubernetes up until 1.26 ([#12259](https://github.com/kubermatic/kubermatic/pull/12259))
- Update konnectivity proxy-agent/server to 0.1.2 for user clusters using Kubernetes 1.27+ ([#12259](https://github.com/kubermatic/kubermatic/pull/12259))
- Update kube-state-metrics to 2.8.2 ([#12225](https://github.com/kubermatic/kubermatic/pull/12225))
- Update metrics-server to 0.6.3 ([#12244](https://github.com/kubermatic/kubermatic/pull/12244))
- Update nginx-ingress-controller to 1.7.1; this removes support for Kubernetes 1.23 for KKP master clusters ([#12234](https://github.com/kubermatic/kubermatic/pull/12234))
- Update node-exporter Helm chart (seed clusters) and addon (user clusters) to 1.5.0 ([#11984](https://github.com/kubermatic/kubermatic/pull/11984))
- Update Prometheus to 2.43.1 ([#12232](https://github.com/kubermatic/kubermatic/pull/12232))
- Update to Go 1.20.5 ([#12361](https://github.com/kubermatic/kubermatic/pull/12361))
- Update Velero to 1.10.1 ([#11966](https://github.com/kubermatic/kubermatic/pull/11966))
- Use Alpine Linux 3.17 for container images ([#12007](https://github.com/kubermatic/kubermatic/pull/12007))

### Miscellaneous

- Anti-affinity rules for control plane components have been simplified to optimise scheduler performance while yielding the same results ([#12215](https://github.com/kubermatic/kubermatic/pull/12215))
- Remove long deprecated heapster addon ([#12055](https://github.com/kubermatic/kubermatic/pull/12055))
- The context name for admin Kubeconfig has been changed to the cluster ID from `default` ([#12006](https://github.com/kubermatic/kubermatic/pull/12006))
- Use buildx instead of Buildah to create multi-architecture KKP container images ([#12393](https://github.com/kubermatic/kubermatic/pull/12393))
- Change `etcd-defragger` CronJob `SuccessfulJobsHistoryLimit` from 0 to 1 to save logs of the most recent successful job ([#12303](https://github.com/kubermatic/kubermatic/pull/12303))- Add `kubermatic_seed_info` metric containing Seed metadata like version, location or phase ([#12194](https://github.com/kubermatic/kubermatic/pull/12194))
- Add `kubermatic_seed_clusters` metric containing the number of user clusters per Seed ([#12194](https://github.com/kubermatic/kubermatic/pull/12194))
- Add `kubermatic_seed_condition` metric describing the conditions for each Seed ([#12194](https://github.com/kubermatic/kubermatic/pull/12194))
- Add `kubermatic_seed_labels` metric containing the Kubernetes labels on Seed resources ([#12194](https://github.com/kubermatic/kubermatic/pull/12194))
- Add option to restrict project deletion to admin ([#12198](https://github.com/kubermatic/kubermatic/pull/12198))
- All Helm charts shipped by KKP now support specifying image pull secrets ([#12098](https://github.com/kubermatic/kubermatic/pull/12098))

### Dashboard & API

#### New Features

- Add new option to restrict project deletion in the admin settings ([#5925](https://github.com/kubermatic/dashboard/pull/5925))
- Introduce Enable Share Cluster settings to toggle the share cluster feature from Admin panel ([#5764](https://github.com/kubermatic/dashboard/pull/5764))
- Add an option in admin settings to enable/enforce auto upgrades for machine deployments ([#5893](https://github.com/kubermatic/dashboard/pull/5893))
- Add support to disable changelog popup ([#5905](https://github.com/kubermatic/dashboard/pull/5905))
- Add support to import digitalocean KubeOne cluster ([#5827](https://github.com/kubermatic/dashboard/pull/5827))
- Add support to import hetzner KubeOne cluster ([#5830](https://github.com/kubermatic/dashboard/pull/5830))
- Add support to import openstack kubeone cluster ([#5951](https://github.com/kubermatic/dashboard/pull/5951))
- Add support to import VSphere kubeone cluster ([#5989](https://github.com/kubermatic/dashboard/pull/5989))
- Configure Ingress Hostname cluster settings of OpenStack provider ([#5861](https://github.com/kubermatic/dashboard/pull/5861))
- Configure report types in schedule configuration ([#5894](https://github.com/kubermatic/dashboard/pull/5894))
- Do not set Assign Public IP by default for AWS and Azure providers ([#5938](https://github.com/kubermatic/dashboard/pull/5938))
- Set Azure data disk size default value to 0 ([#5987](https://github.com/kubermatic/dashboard/pull/5987))
- Support to enable accelerated networking for machines on Azure ([#5906](https://github.com/kubermatic/dashboard/pull/5906))
- The context name for OIDC Kubeconfig has been changed to the cluster ID from `default` ([#5810](https://github.com/kubermatic/dashboard/pull/5810))
- VMware Cloud Director now supports authentication using API Token ([#5885](https://github.com/kubermatic/dashboard/pull/5885))

#### Bugfixes

- UI/UX improvements for vSphere credentials in provider settings step ([#5959](https://github.com/kubermatic/dashboard/pull/5959))
    - By default, username/password will be configured and dedicated credentials will be used to configure infra management user for vSphere
- Add cache busting mechanism for theme styles ([#5943](https://github.com/kubermatic/dashboard/pull/5943))
- Allow removing cluster label when PodNodeSelector admission plugin and clusterDefaultNodeSelector namespace are set ([#5981](https://github.com/kubermatic/dashboard/pull/5981))
- Allow updating of the `clusterNetwork.proxyMode` via the KKP API (PATCH endpoint) ([#5803](https://github.com/kubermatic/dashboard/pull/5803))
- AWS subnets are fetched correctly if credentials are provided directly instead of using a preset ([#5883](https://github.com/kubermatic/dashboard/pull/5883))
- Fix cluster wizard not selecting a default version if custom versions are configured in `KubermaticConfiguration` ([#5879](https://github.com/kubermatic/dashboard/pull/5879))
- Fix Datacenter MachineFlavorFilter not used ([#5787](https://github.com/kubermatic/dashboard/pull/5787))
- Machine Deployments are initialized without waiting for all cluster details to finish loading ([#5922](https://github.com/kubermatic/dashboard/pull/5922))
- Show correct health information for Machine Deployments with no replicas ([#5837](https://github.com/kubermatic/dashboard/pull/5837))

#### Design

- Add an option to clear VSphere tags category so it doesn't get stuck when there are no tags ([#5940](https://github.com/kubermatic/dashboard/pull/5940))
- Add color to required indicator of untouched and empty required form fields ([#5937](https://github.com/kubermatic/dashboard/pull/5937))
- Add indicator of what was changed on editing dialogs ([#5843](https://github.com/kubermatic/dashboard/pull/5843))
- Add warning message in the cluster list page in case some seeds are not reachable ([#5982](https://github.com/kubermatic/dashboard/pull/5982))
- Allow selection of items per page under every table along with user settings page ([#5954](https://github.com/kubermatic/dashboard/pull/5954))
- Improve page responsiveness for smaller screen sizes ([#5801](https://github.com/kubermatic/dashboard/pull/5801))
- Update Dialogs to follow latest material design specifications ([#5927](https://github.com/kubermatic/dashboard/pull/5927))
- Update the notification design and improve user experience ([#5970](https://github.com/kubermatic/dashboard/pull/5970))

#### Updates

- Update to Go 1.20.5 ([#6025](https://github.com/kubermatic/dashboard/pull/6025))
- Use Alpine Linux 3.17 for container images ([#5814](https://github.com/kubermatic/dashboard/pull/5814))

