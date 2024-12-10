# Kubermatic 2.24

- [v2.24.0](#v2240)
- [v2.24.1](#v2241)
- [v2.24.2](#v2242)
- [v2.24.3](#v2243)
- [v2.24.4](#v2244)
- [v2.24.5](#v2245)
- [v2.24.6](#v2246)
- [v2.24.7](#v2247)
- [v2.24.8](#v2248)
- [v2.24.9](#v2249)
- [v2.24.10](#v22410)
- [v2.24.11](#v22411)
- [v2.24.12](#v22412)
- [v2.24.13](#v22413)
- [v2.24.14](#v22414)

## v2.24.14

**GitHub release: [v2.24.14](https://github.com/kubermatic/kubermatic/releases/tag/v2.24.14)**

### Bugfixes

- Fix seed controller panic while creating `nodeport-proxy-envoy` deployment for user clusters ([#13835](https://github.com/kubermatic/kubermatic/pull/13835))
- Fix TOML/YAML configuration mixup in the IAP Helm chart ([#13786](https://github.com/kubermatic/kubermatic/pull/13786))
- Select correct template value when editing MD of VCD provider ([#6927](https://github.com/kubermatic/dashboard/pull/6927))

### Updates

- Security: Update Cilium to 1.13.14 / 1.14.16 because the previous versions are affected by multiple CVEs ([#13849](https://github.com/kubermatic/kubermatic/pull/13849))


## v2.24.13

**GitHub release: [v2.24.13](https://github.com/kubermatic/kubermatic/releases/tag/v2.24.13)**

### Bugfixes

- Fix vSphere CCM/CSI images (pre 1.28 clusters will now use a Kubermatic-managed mirror on quay.io for the images). ([#13720](https://github.com/kubermatic/kubermatic/pull/13720))
- Kubevirt provider waits for the etcd backups to get deleted before removing the namespace, when a cluster is deleted ([#13635](https://github.com/kubermatic/kubermatic/pull/13635))
- Fix runbook URL for Prometheus alerting rules ([#13691](https://github.com/kubermatic/kubermatic/pull/13691))
- `local` command in KKP installer does not check / wait for DNS anymore ([#13693](https://github.com/kubermatic/kubermatic/pull/13693))
- Fix missing registry overwrites for cluster-backup (Velero) images, kubevirt CSI images and KubeOne jobs ([#13695](https://github.com/kubermatic/kubermatic/pull/13695))

### Updates

- Update Canal 3.27 to 3.27.4 ([#13633](https://github.com/kubermatic/kubermatic/pull/13633))

## v2.24.12

**GitHub release: [v2.24.12](https://github.com/kubermatic/kubermatic/releases/tag/v2.24.12)**

### Bugfixes

- Deduplicate alerts in alertmanager ([#13605](https://github.com/kubermatic/kubermatic/pull/13605))
- Fix KubermaticConfiguration getting deleted when a Seed on a shared master/seed cluster is deleted ([#13585](https://github.com/kubermatic/kubermatic/pull/13585))
- Fix usercluster-ctrl-mgr spamming oldest node version in its logs ([#13440](https://github.com/kubermatic/kubermatic/pull/13440))
- Restore missing bgpconfigurations CRD in Canal 3.27 ([#13505](https://github.com/kubermatic/kubermatic/pull/13505))
- Add the label `name: nodeAgent` to the Velero DaemonSet pods ([#13516](https://github.com/kubermatic/kubermatic/pull/13516))
- The secret `velero-restic-credentials` is renamed to `velero-repo-credentials` ([#13516](https://github.com/kubermatic/kubermatic/pull/13516))

### Updates

- Update Go version to 1.21.12 ([#13557](https://github.com/kubermatic/kubermatic/pull/13557))
- Update machine-controller to v1.58.6 ([#13560](https://github.com/kubermatic/kubermatic/pull/13560))


## v2.24.11

**GitHub release: [v2.24.11](https://github.com/kubermatic/kubermatic/releases/tag/v2.24.11)**

### Bugfixes

- Fix an issue with Azure support that prevented successful provisioning of user clusters on some Azure locations ([#13405](https://github.com/kubermatic/kubermatic/pull/13405))
- Fix the pagination in project members table ([#6743](https://github.com/kubermatic/dashboard/pull/6743))

### Updates

- Update Go version to 1.21.12 ([#13487](https://github.com/kubermatic/kubermatic/pull/13487), [#6731](https://github.com/kubermatic/dashboard/pull/6731))


## v2.24.10

**GitHub release: [v2.24.10](https://github.com/kubermatic/kubermatic/releases/tag/v2.24.10)**

### Bugfixes

- Fix a critical regression in Applications with Helm sources, which resulted in "release: not found" errors ([#13462](https://github.com/kubermatic/kubermatic/pull/13462))


## v2.24.9

**GitHub release: [v2.24.9](https://github.com/kubermatic/kubermatic/releases/tag/v2.24.9)**

### New Features

- Add `spec.componentsOverride.coreDNS` to Cluster objects, deprecate `spec.clusterNetwork.coreDNSReplicas` in favor of the new `spec.componentsOverride.coreDNS.replicas` field ([#13418](https://github.com/kubermatic/kubermatic/pull/13418))

### Bugfixes

- Add `displayName` and `scope` columns for printing the cluster templates; `kubectl get clustertemplates` will now show the actual display name and scope for the cluster templates ([#13419](https://github.com/kubermatic/kubermatic/pull/13419))
- Add Kubernetes 1.28.x support for cluster-autoscaler addon ([#13386](https://github.com/kubermatic/kubermatic/pull/13386))
- Address inconsistencies in Helm that lead to an Application being stuck in "pending-install" ([#13301](https://github.com/kubermatic/kubermatic/pull/13301))
- Fix a bug where unrequired `cloud-config` secret was being propagated to the user clusters ([#13372](https://github.com/kubermatic/kubermatic/pull/13372))
- Fix null pointer exception that occurred while our controllers checked whether the CSI addon is in use or not ([#13369](https://github.com/kubermatic/kubermatic/pull/13369))
- Fix: use correct networkpolicy port for metrics-server ([#13446](https://github.com/kubermatic/kubermatic/pull/13446))

### Updates

- Update Go version to 1.21.11 ([#13429](https://github.com/kubermatic/kubermatic/pull/13429), [#6711](https://github.com/kubermatic/dashboard/pull/6711))
- Update OSM to v1.4.3; fixing cloud-init bootstrapping issues on Ubuntu 22.04 on Azure ([#13378](https://github.com/kubermatic/kubermatic/pull/13378))
- Update machine-controller to v1.58.5, fixing support for Rockylinux 8 on AWS ([#13432](https://github.com/kubermatic/kubermatic/pull/13432))


## [v2.24.8](https://github.com/kubermatic/kubermatic/releases/tag/v2.24.8)

### Bugfixes

* [ACTION REQUIRED] The latest Ubuntu 22.04 images ship with cloud-init 24.x package. This package has breaking changes and thus rendered our OSPs as incompatible. It's recommended to refresh your machines with latest provided OSPs to ensure that a system-wide package update, that updates cloud-init to 24.x, doesn't break the machines ([#13359](https://github.com/kubermatic/kubermatic/pull/13359))

### Updates

* Update operating-system-manager to v1.4.2.


## [v2.24.7](https://github.com/kubermatic/kubermatic/releases/tag/v2.24.7)

### New Feature

- Monitoring: introduce `signout_redirect_url` field to configure the URL to redirect the user to after signing out from Grafana ([#13313](https://github.com/kubermatic/kubermatic/pull/13313))

### Bugfixes

- Enable `local` command for Enterprise Edition ([#13333](https://github.com/kubermatic/kubermatic/pull/13333))
- Fix template value for MachineDeployments in edit mode ([#6669](https://github.com/kubermatic/dashboard/pull/6669))
- Hotfix to mitigate a bug in new releases of Chromium that causes browser crashes on `mat-select` component. For more details: https://issuetracker.google.com/issues/335553723 ([#6667](https://github.com/kubermatic/dashboard/pull/6667))
- Fix Azure CCM not being reconciled because of labelling changes ([#13334](https://github.com/kubermatic/kubermatic/pull/13334))
- Improve Helm repository prefix handling for system applications; only prepend `oci://` prefix if it doesn't already exist in the specified URL ([#13336](https://github.com/kubermatic/kubermatic/pull/13336))
- Installer does not validate IAP `client_secrets` for Grafana and Alertmanager the same way it does for `encryption_key` ([#13315](https://github.com/kubermatic/kubermatic/pull/13315))

### Chore

- Update machine-controller to v1.58.4 ([#13348](https://github.com/kubermatic/kubermatic/pull/13348))


## [v2.24.6](https://github.com/kubermatic/kubermatic/releases/tag/v2.24.6)

### API Changes

- Add `spec.componentsOverride.operatingSystemManager` to allow overriding OSM settings and resources ([#13287](https://github.com/kubermatic/kubermatic/pull/13287))

### Bugfixes

- Fix high CPU usage in master-controller-manager ([#13209](https://github.com/kubermatic/kubermatic/pull/13209))

### Updates

- Add Canal CNI version v3.27.3, having a fix to the ipset incompatibility bug ([#13246](https://github.com/kubermatic/kubermatic/pull/13246))
- Add support for Kubernetes 1.27.13 and 1.28.9 (fixes CVE-2024-3177) ([#13299](https://github.com/kubermatic/kubermatic/pull/13299))
- Update to Go 1.21.9 ([#13247](https://github.com/kubermatic/kubermatic/pull/13247))

### Cleanup

- Addons reconciliation is triggered more consistently for changes to Cluster objects, reducing the overall number of unnecessary addon reconciliations ([#13252](https://github.com/kubermatic/kubermatic/pull/13252))

## [v2.24.5](https://github.com/kubermatic/kubermatic/releases/tag/v2.24.5)

### Bugfixes

- Add images for Velero and KubeLB to mirrored images list ([#13198](https://github.com/kubermatic/kubermatic/pull/13198))
- Exclude `test` folders which contain symlinks that break once the archive is untarred ([#13151](https://github.com/kubermatic/kubermatic/pull/13151))
- Fix missing image registry override for hubble-ui components if Cilium is deployed as System Application ([#13139](https://github.com/kubermatic/kubermatic/pull/13139))
- Fix: usercluster-controller-manager failed to reconcile cluster with disable CSI drivers ([#13183](https://github.com/kubermatic/kubermatic/pull/13183))
- Fix Azure loadbalancer-related issues by updating Azure CCM from v1.28.0 to v1.28.5 for the user clusters created with Kubernetes v1.28 ([#13173](https://github.com/kubermatic/kubermatic/pull/13173))
- Fix a bug where OSPs were not being listed for VMware Cloud Director ([#6592](https://github.com/kubermatic/dashboard/pull/6592))
- Fix invalid project ID in API requests for Nutanix provider ([#6572](https://github.com/kubermatic/dashboard/pull/6572))
- Fix a bug where dedicated credentials were incorrectly being required as mandatory input when editing vSphere provider settings for a cluster ([#6567](https://github.com/kubermatic/dashboard/pull/6567))


### Chore

- Update to Go 1.21.8 ([#13164](https://github.com/kubermatic/kubermatic/pull/13164)) and ([#6593](https://github.com/kubermatic/dashboard/pull/6593))

### Design

- Improve compatibility with cluster-autoscaler 1.27.1+: Pods using temporary volumes are now marked as evictable ([#13197](https://github.com/kubermatic/kubermatic/pull/13197))

## [v2.24.4](https://github.com/kubermatic/kubermatic/releases/tag/v2.24.4)

### Bugfixes

- Fix the panic of the seed controller manager while checking CSI addon usage for user clusters, when a user cluster has PVs which were migrated from the in-tree provisioner to the CSI provisioner ([#13126](https://github.com/kubermatic/kubermatic/pull/13126))

### New Feature

- We maintain now a dedicated docker image for the conformance tester, mainly for internal use ([#13113](https://github.com/kubermatic/kubermatic/pull/13113))

## [v2.24.3](https://github.com/kubermatic/kubermatic/releases/tag/v2.24.3)

### Bugfixes

- **ACTION REQUIRED:** For velero helm chart upgrade related change. If you use `velero.restic.deploy: true`, you will see new daemonset `node-agent` running in `velero` namespace. You might need to remove existing daemonset named `restic` manually ([#12998](https://github.com/kubermatic/kubermatic/pull/12998))
- Fix a bug where resources deployed in the user cluster namespace on the seed, for the CSI drivers, were not being removed when the CSI driver was disabled ([#13048](https://github.com/kubermatic/kubermatic/pull/13048))
- Fix panic, if no KubeVirt DNS config was set in the datacenter ([#13028](https://github.com/kubermatic/kubermatic/pull/13028))
- Validation - Added check for PVs having CSI provisioner before disabling the CSI addon ([#13092](https://github.com/kubermatic/kubermatic/pull/13009))

### Updates

- Update metering to v1.1.2, fixing an error when a custom CA bundle is used ([#13013](https://github.com/kubermatic/kubermatic/pull/13013))
- Update operating-system-manager (OSM) to [v1.4.1](https://github.com/kubermatic/operating-system-manager/releases/tag/v1.4.1) ([#13082](https://github.com/kubermatic/kubermatic/pull/13082))
  - This includes a fix for Flatcar stable channel (version 3815.2.0) failing to provision as new nodes.
- Update go-git. This enables Applications to work with private Azure DevOps Git repositories ([#12995](https://github.com/kubermatic/kubermatic/pull/12995))


## [v2.24.2](https://github.com/kubermatic/kubermatic/releases/tag/v2.24.2)

### Action Required

- **ACTION REQUIRED:** User Cluster MLA `cortex` chart has been upgraded to resolve issues for cortex-compactor and improve stability of the User Cluster MLA feature. Few actions are required to be taken to use new upgraded charts ([#12935](https://github.com/kubermatic/kubermatic/pull/12935))
    - Refer to [Upstream helm chart values](https://github.com/cortexproject/cortex-helm-chart/blob/v2.1.0/values.yaml) to see the latest default values
    - Some of the values from earlier `values.yaml` are now incompatible with latest version. They are removed in the `values.yaml` in the current chart. But if you had copied the original values.yaml to customize it further, you may see that `kubermatic-installer` will detect such incompatible options and churn out errors and explain that action that needs to be taken.
    - The memcached-* charts are now subcharts of cortex chart so if you provided configuration for `memcached-*` blocks in your `values.yaml` for user-mla, you must move them under `cortex:` block

### Updates

- Add support for Kubernetes v1.26.13, v1.27.10, v1.28.6 and set default version to v1.27.10 ([#12982](https://github.com/kubermatic/kubermatic/pull/12982))

### Bugfixes

- If the seed cluster is using Cilium as CNI, create CiliumClusterwideNetworkPolicy for api-server connectivity ([#12966](https://github.com/kubermatic/kubermatic/pull/12966))
- Stop constantly re-deploying operating-system-manager when registry mirrors are configured ([#12972](https://github.com/kubermatic/kubermatic/pull/12972))
- The Kubermatic installer will now detect DNS settings based on the Ingress instead of the nginx-ingress LoadBalancer, allowing for other ingress solutions to be properly detected ([#12934](https://github.com/kubermatic/kubermatic/pull/12934))

### Removals

- Remove 1.25 from list of supported versions on AKS (EOL on January 14th) ([#12962](https://github.com/kubermatic/kubermatic/pull/12962))

## [v2.24.1](https://github.com/kubermatic/kubermatic/releases/tag/v2.24.1)

### Bugfixes

- Applied a fix to VPA caused by [upstream release issue](https://github.com/kubernetes/autoscaler/issues/5982) which caused insufficient RBAC permission for VPA recommender pod ([#12872](https://github.com/kubermatic/kubermatic/pull/12872))
- Fix cert-manager values block. cert-manager deployment will get updated as part of upgrade ([#12854](https://github.com/kubermatic/kubermatic/pull/12854))
- Fix `mirror-images` command in installer not being able to extract the addons ([#12868](https://github.com/kubermatic/kubermatic/pull/12868))
- Fix cases where, when using dedicated infra- and ccm-credentials, infra-credentials were always overwritten by ccm-credentials ([#12421](https://github.com/kubermatic/kubermatic/pull/12421))
- No longer fail constructing vSphere endpoint when a `/` suffix is present in the datacenter configuration ([#12861](https://github.com/kubermatic/kubermatic/pull/12861))

### New Features

- Openstack: allow configuring Cinder CSI topology support either on `Cluster` or `Seed` resource field `cinderTopologyEnabled` ([#12878](https://github.com/kubermatic/kubermatic/pull/12878))

### Updates

- Update machine-controller to [v1.58.1](https://github.com/kubermatic/machine-controller/releases/tag/v1.58.1) ([#12902](https://github.com/kubermatic/kubermatic/pull/12902))
- Update Anexia CCM (cloud-controller-manager) to version 1.5.5 ([#12911](https://github.com/kubermatic/kubermatic/pull/12911))
    - Fixes leaking LoadBalancer reconciliation metric
    - Updates various dependencies

### Miscellaneous

- KKP is now built with Go 1.21.5 ([#12898](https://github.com/kubermatic/kubermatic/pull/12898))
- Increase the default resources for VPA components to prevent OOMs ([#12887](https://github.com/kubermatic/kubermatic/pull/12887))

## [v2.24.0](https://github.com/kubermatic/kubermatic/releases/tag/v2.24.0)

Before upgrading, make sure to read the [general upgrade guidelines](https://docs.kubermatic.com/kubermatic/v2.24/installation/upgrading/). Consider tweaking `seedControllerManager.maximumParallelReconciles` to ensure user cluster reconciliations will not cause resource exhaustion on seed clusters. A [full upgrade guide is available from the official documentation](https://docs.kubermatic.com/kubermatic/v2.24/installation/upgrading/upgrade-from-2.23-to-2.24/).

### Read Before Upgrading

- **ACTION REQUIRED:** legacy backup controller has been removed. Before upgrading, please change to the backup and restore feature that uses [backup destinations](https://docs.kubermatic.com/kubermatic/v2.24/tutorials-howtos/etcd-backups/) if the legacy controller is still in use ([#12473](https://github.com/kubermatic/kubermatic/pull/12473))
- `s3-storeuploader` has been removed ([#12473](https://github.com/kubermatic/kubermatic/pull/12473))
- OpenVPN for control plane to node connectivity has been deprecated. It will be removed in future releases of KKP. Upgrading all user cluster to Konnectivity is strongly recommended ([#12691](https://github.com/kubermatic/kubermatic/pull/12691))
- User clusters require upgrading to Kubernetes 1.26 prior to upgrading to KKP 2.24 ([#12740](https://github.com/kubermatic/kubermatic/pull/12740))

### Action Required

- **ACTION REQUIRED (EE ONLY):** Update metering component to v1.1.1, fixing highly inaccurate data in cluster reports. Reports generated in KKP v2.23.2+ or v2.22.5+ do not represent actual consumption. Ad-hoc reports for time frames that need correct consumption data can be generated [by following our documentation](https://docs.kubermatic.com/kubermatic/v2.24/tutorials-howtos/metering/#custom-reports) ([#12822](https://github.com/kubermatic/kubermatic/pull/12822))

### API Changes

- The field `vmNetName` in `Cluster` and `Preset` resources for vSphere clusters is deprecated and `networks` should be used instead ([#12444](https://github.com/kubermatic/kubermatic/pull/12444))
- The field `konnectivityEnabled` in `Cluster` resources is deprecated. Clusters should set this to `true` to migrate off OpenVPN as Konnectivity being enabled will be assumed in future KKP releases ([#12691](https://github.com/kubermatic/kubermatic/pull/12691))
- Set Cilium as default CNI for user clusters ([#12752](https://github.com/kubermatic/kubermatic/pull/12752))

### Supported Kubernetes Versions

- Add support for Kubernetes 1.28 ([#12593](https://github.com/kubermatic/kubermatic/pull/12593))
- Add support for Kubernetes 1.26.9, 1.27.6 and 1.28.2 ([#12638](https://github.com/kubermatic/kubermatic/pull/12638))
- Set default Kubernetes version to 1.27.6 ([#12638](https://github.com/kubermatic/kubermatic/pull/12638))
- Remove support for Kubernetes 1.24 ([#12570](https://github.com/kubermatic/kubermatic/pull/12570))
- Remove support for Kubernetes 1.25 ([#12740](https://github.com/kubermatic/kubermatic/pull/12740))

#### Supported Versions

- v1.26.1
- v1.26.4
- v1.26.6
- v1.26.9
- v1.27.3
- v1.27.6 (default)
- v1.28.2

### KubeLB (Enterprise Edition only)

This release adds support for [KubeLB](https://docs.kubermatic.com/kubelb/), a cloud native multi-tenant load balancing solution by Kubermatic.

- Add KubeLB integration with KKP; introduce KubeLB as a first-class citizen in KKP ([#12667](https://github.com/kubermatic/kubermatic/pull/12667))
- Extend cluster health status with KubeLB health check ([#12685](https://github.com/kubermatic/kubermatic/pull/12685))
- Support for enforcing KubeLB at the datacenter level ([#12685](https://github.com/kubermatic/kubermatic/pull/12685))
- Support to configure node address type for KubeLB at the datacenter level ([#12715](https://github.com/kubermatic/kubermatic/pull/12715))
- Update KubeLB CCM image to v0.4.0 ([#12786](https://github.com/kubermatic/kubermatic/pull/12786))

### Metering (Enterprise Edition only)

- Following fields are removed from metering reports ([#12545](https://github.com/kubermatic/kubermatic/pull/12545))
  - Cluster reports
    - Removal of `total-used-cpu-seconds`, use `average-used-cpu-millicores` instead
    - Removal of `average-available-cpu-cores`, use `average-available-cpu-millicores` instead
  - Namespace reports
    - Removal of `total-used-cpu-seconds`, use `average-used-cpu-millicores` instead
- Add `monthly` parameter for metering monthly report generation ([#12544](https://github.com/kubermatic/kubermatic/pull/12544))
- Update metering component to v1.1.1, fixing highly inaccurate data in cluster reports (see [Action Required](#action-required) for more details) ([#12822](https://github.com/kubermatic/kubermatic/pull/12822))

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
- Migrate `CSIDriver` `csi.vsphere.vmware.com` to no longer advertise inline ephemeral volumes as supported ([#12813](https://github.com/kubermatic/kubermatic/pull/12813))

#### DigitalOcean

- Digitalocean CCM versions now depend on the user cluster version, following the loose [upstream compatibility guarantees](https://github.com/digitalocean/digitalocean-cloud-controller-manager#releases) ([#12600](https://github.com/kubermatic/kubermatic/pull/12600))

#### Hetzner

- Hetzner CSI: recreate CSIDriver to allow upgrade from 1.6.0 to 2.2.0 ([#12432](https://github.com/kubermatic/kubermatic/pull/12432))
- EE: Correctly validate Hetzner API response for server type while calculating resource requirements and for networks while validating cloud spec ([#12716](https://github.com/kubermatic/kubermatic/pull/12716))

### CNIs

#### Cilium

- Set Cilium as default CNI for user clusters ([#12752](https://github.com/kubermatic/kubermatic/pull/12752))
- Add support for Cilium 1.14.3 and 1.13.8 and deprecate previous patch releases, mitigating CVE-2023-44487, CVE-2023-39347, CVE-2023-41333, CVE-2023-41332 ([#12761](https://github.com/kubermatic/kubermatic/pull/12761))
- Update Cilium v1.11 and v1.12 patch releases to v1.11.20 and v1.12.13 ([#12561](https://github.com/kubermatic/kubermatic/pull/12561))
- Remove and replace deprecated `clusterPoolIPv4PodCIDR` and `clusterPoolIPv6PodCIDR` Helm value with `clusterPoolIPv4PodCIDRList` and `clusterPoolIPv6PodCIDRList` for Cilium 1.13+ ([#12561](https://github.com/kubermatic/kubermatic/pull/12561))

#### Canal

- Add support for Canal v3.26.1 ([#12561](https://github.com/kubermatic/kubermatic/pull/12561))
- Deprecate Canal v3.23 ([#12561](https://github.com/kubermatic/kubermatic/pull/12561))
- Mark all Canal CRDs with preserveUnknownFields: false ([#12538](https://github.com/kubermatic/kubermatic/pull/12538))

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
- Fix CPU Utilization graph showing no data for User Cluster MLA dashboard "Nodes Overview" ([#12814](https://github.com/kubermatic/kubermatic/pull/12814))
- Fix empty panels in Grafana dashboard "Resource Usage per Namespace" for Master/Seed MLA ([#12816](https://github.com/kubermatic/kubermatic/pull/12816))

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
- `kubermatic-installer` will now validate the existing MinIO filesystem before attempting a `kubermatic-seed` stack installation ([#12477](https://github.com/kubermatic/kubermatic/pull/12477))
- Increase default CPU limits for KKP API/seed/master-controller-managers to prevent general slowness ([#12764](https://github.com/kubermatic/kubermatic/pull/12764))
- Extend project-synchronizer controller in kubermatic-master-controller-manager to propagate labels from Projects in the master cluster to Projects in the seed cluster. This fixes an issue where the metering report doesn't contain project-labels in separate master/seed setups ([#12791](https://github.com/kubermatic/kubermatic/pull/12791))

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
- Add Kubernetes 1.28 to EKS versions, remove Kubernetes 1.23 ([#12789](https://github.com/kubermatic/kubermatic/pull/12789))
- Update machine-controller to [v1.58.0](https://github.com/kubermatic/machine-controller/releases/tag/v1.58.0) ([#12825](https://github.com/kubermatic/kubermatic/pull/12825))
- Update operating-system-manager to [v1.4.0](https://github.com/kubermatic/operating-system-manager/releases/tag/v1.4.0) ([#12826](https://github.com/kubermatic/kubermatic/pull/12826))

### Miscellaneous

- Use `etcd-launcher` to check if etcd is running before starting kube-apiserver and to defragment etcd clusters ([#12450](https://github.com/kubermatic/kubermatic/pull/12450))
- Create a `NetworkPolicy` for user cluster kube-apiserver to access the Seed Kubernetes API ([#12569](https://github.com/kubermatic/kubermatic/pull/12569))
- Improve `http-prober` performance in user clusters with a lot of CRDs ([#12634](https://github.com/kubermatic/kubermatic/pull/12634))
- Update Velero helm chart's apiVersion to v2; Helm 3 & above would be required to install it ([#12765](https://github.com/kubermatic/kubermatic/pull/12765))

### Dashboard & API

#### Cleanup

- Remove unused v1 endpoints for KKP API ([#6116](https://github.com/kubermatic/dashboard/pull/6116))

#### Bugfixes

- Add operating system profile to the machine deployment patch object ([#6264](https://github.com/kubermatic/dashboard/pull/6264))
- Add vertical scroll to the install Addon dialog ([#6123](https://github.com/kubermatic/dashboard/pull/6123))
- Allow expansion of sidenav on small screen sizes ([#6218](https://github.com/kubermatic/dashboard/pull/6218))
- Fix a bug where available version upgrades for CNI plugins were not being properly deduced ([#6317](https://github.com/kubermatic/dashboard/pull/6317))
- Fix a bug where network and IPv6 subnet pool options were not loading during Openstack cluster creation ([#6120](https://github.com/kubermatic/dashboard/pull/6120))
- Fix a bug where project scope endpoints for GCP were working only with the presets instead of one of presets or credentials ([#6078](https://github.com/kubermatic/dashboard/pull/6078))
- CE: Fix a bug where the values configured for vSphere, Hetzner, and Nutanix nodes were not being persisted ([#6171](https://github.com/kubermatic/dashboard/pull/6171))
- Fix an issue where a custom OSP value was not selected when editing/customizing cluster template ([#6325](https://github.com/kubermatic/dashboard/pull/6325))
- Fix docs link about OIDC groups on user settings page ([#6208](https://github.com/kubermatic/dashboard/pull/6208))
- Fix listing events for external clusters ([#6337](https://github.com/kubermatic/dashboard/pull/6337))
- Fix support for keycloak OIDC logout. New field `oidc_provider` was introduced to support OIDC provider specific configurations. Configuring `oidc_provider` as `keycloak` will properly configure the logout workflow ([#6144](https://github.com/kubermatic/dashboard/pull/6144))
- Fix the default value for CNI plugin version ([#6258](https://github.com/kubermatic/dashboard/pull/6258))
- Fix the empty `id_token_hint` value when logout from Keycloak ([#6248](https://github.com/kubermatic/dashboard/pull/6248))
- Fix: vSphere tags for initial machine deployments ([#6179](https://github.com/kubermatic/dashboard/pull/6179))
- OpenStack: Fix project and projectID header propagation for project scoped endpoints ([#6082](https://github.com/kubermatic/dashboard/pull/6082))
- Openstack: take `TenantID` into account while listing networks, security groups and subnet pools ([#6156](https://github.com/kubermatic/dashboard/pull/6156))
- VMware Cloud Director: fix an issue where the API Token from preset was not being sourced to the cluster ([#6196](https://github.com/kubermatic/dashboard/pull/6196))
- Fix `Enable Share Cluster` button in Admin Settings ([#6340](https://github.com/kubermatic/dashboard/pull/6340))
- Fix an issue where `clusterDefaultNodeSelector` label was being added back on opening of edit cluster dialog ([#6362](https://github.com/kubermatic/dashboard/pull/6362))
- Fix issue with managing clusters if some seeds are down ([#6374](https://github.com/kubermatic/dashboard/pull/6374))
- Fix a bug where API call to list projects was failing due to slowness ([#6385](https://github.com/kubermatic/dashboard/pull/6385))

#### New Features

- Support for enabling/disabling operating systems for machines of user clusters ([#6070](https://github.com/kubermatic/dashboard/pull/6070))
- Add functionality to configure `basePath` in preset and cluster for vSphere ([#6281](https://github.com/kubermatic/dashboard/pull/6281))
- Add support for encrypted root volumes in AWS ([#6125](https://github.com/kubermatic/dashboard/pull/6125))
- Add VM anti-affinity setting for vSphere machine deployments ([#6068](https://github.com/kubermatic/dashboard/pull/6068))
- EE: Support for configuring KubeLB for user clusters ([#6256](https://github.com/kubermatic/dashboard/pull/6256))
- Support for configuring multiple networks for vSphere ([#6069](https://github.com/kubermatic/dashboard/pull/6069))
- Support for disabling admin kubeconfig endpoint ([#6246](https://github.com/kubermatic/dashboard/pull/6246))
- Support multiple NodePort allowed IP ranges ([#6188](https://github.com/kubermatic/dashboard/pull/6188))
- Update default CNI plugin to `Cilium` ([#6328](https://github.com/kubermatic/dashboard/pull/6328))
- VMware Cloud Director: Support for configuring placement and sizing policy for machines ([#6094](https://github.com/kubermatic/dashboard/pull/6094))
- Enforce Konnectivity value because OpenVPN support is now deprecated ([#6361](https://github.com/kubermatic/dashboard/pull/6361))

### Updates

- Update to Go 1.21.3 ([#6268](https://github.com/kubermatic/dashboard/pull/6268))
- Update web-terminal image to kubectl 1.27, Helm 3.12.3 and curl 8.4.0 ([#6283](https://github.com/kubermatic/dashboard/pull/6283))
