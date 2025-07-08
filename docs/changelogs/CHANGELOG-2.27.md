# Kubermatic 2.27

- [v2.27.0](#v2270)
- [v2.27.1](#v2271)
- [v2.27.2](#v2272)
- [v2.27.3](#v2273)
- [v2.27.4](#v2274)
- [v2.27.5](#v2275)
- [v2.27.6](#v2276)

## v2.27.6

**GitHub release: [v2.27.6](https://github.com/kubermatic/kubermatic/releases/tag/v2.27.6)**

### New Features

- KubeLB: KKP defaulting will now enable KubeLB for a cluster if it's enforced at the datacenter level ([#14747](https://github.com/kubermatic/kubermatic/pull/14747))

### Design

- Fix clickable documentation links in hints for disabled checkboxes ([#7434](https://github.com/kubermatic/dashboard/pull/7434))

### Bugfixes

- Fix KubeLB checkbox state management and UI flickering issues in cluster creation wizard/edit cluster dialog ([#7460](https://github.com/kubermatic/dashboard/pull/7460))
- Fix validation error when switching expose strategy from Tunneling to LoadBalancer by clearing tunnelingAgentIP automatically ([#7422](https://github.com/kubermatic/dashboard/pull/7422))
- KubeLB: Fix a bug where enforcement on a datacenter was not enabling KubeLB for the user clusters in the dashboard ([#7455](https://github.com/kubermatic/dashboard/pull/7455))
- List all OpenStack networks in the UI wizard during cluster creation ([#7437](https://github.com/kubermatic/dashboard/pull/7437))
- Project viewers can now only view cluster templates. Create, update, and delete actions are restricted except deletion by the owner ([#7482](https://github.com/kubermatic/dashboard/pull/7482))
- Shows custom disk fields when a custom disk is configured in the Machine Deployment edit dialog ([#7415](https://github.com/kubermatic/dashboard/pull/7415))
- Unset backup sync period if value is empty ([#7444](https://github.com/kubermatic/dashboard/pull/7444))

### Updates

- Update machine-controller(MC) to [v1.61.3](https://github.com/kubermatic/machine-controller/releases/tag/v1.61.3) ([#14729](https://github.com/kubermatic/kubermatic/pull/14729))
- Update operating-system-manager(OSM) to [v1.6.7](https://github.com/kubermatic/operating-system-manager/releases/tag/v1.6.7) ([#14794](https://github.com/kubermatic/kubermatic/pull/14794))
- Update to Go 1.23.10 ([#14667](https://github.com/kubermatic/kubermatic/pull/14667),[#7450](https://github.com/kubermatic/dashboard/pull/7450))

### Cleanup

- By default the oauth2-proxy disables Dex's approval screen now. To return to the old behaviour, set `approval_prompt = "force"` for each IAP deployment in your Helm values.yaml ([#14751](https://github.com/kubermatic/kubermatic/pull/14751))


## v2.27.5

**GitHub release: [v2.27.5](https://github.com/kubermatic/kubermatic/releases/tag/v2.27.5)**

### Bugfixes

- Add validation for checks in the installer for the new Dex Helm chart ([#14624](https://github.com/kubermatic/kubermatic/pull/14624))
- Correctly mounts the custom CA bundle ConfigMap to fix reconciliation failures in custom CA environments ([#14575](https://github.com/kubermatic/kubermatic/pull/14575))
- Fix `--skip-seed-validation` flag on the KKP installer ([#14589](https://github.com/kubermatic/kubermatic/pull/14589))
- Fix a bug that caused network policies to not be removed from the KubeVirt infra cluster ([#14639](https://github.com/kubermatic/kubermatic/pull/14639))
- Fix a bug where CSI Snapshot validating webhook was being deployed even if the CSI drivers are disabled for a cluster. When the csi driver is disabled after cluster creation the both mentioned resources will be cleaned up now ([#14466](https://github.com/kubermatic/kubermatic/pull/14466))
- KubeLB: CCM will adjust the tenant kubeconfig to use API server endpoint and CA certificate from the management kubeconfig that is provided to KKP at the seed/datacenter level ([#14522](https://github.com/kubermatic/kubermatic/pull/14522))
- Remove redundant and undocumented/unused `remove-oauth-release` flag for installer ([#14631](https://github.com/kubermatic/kubermatic/pull/14631))
- Use infra management user credentials (if configured) to fetch data for vSphere ([#7397](https://github.com/kubermatic/dashboard/pull/7397))

### Miscellaneous

- Support KubeVirt CCM Load Balancer Interface Disabling ([#14641](https://github.com/kubermatic/kubermatic/pull/14641))

### Updates

- Update KubeVirt CSI Driver to commit `9ad38f9e49c296acfe7b9d3301ebff8a1056fa68` ([#14640](https://github.com/kubermatic/kubermatic/pull/14640))
- Update machine controller to version v1.61.2 ([#14644](https://github.com/kubermatic/kubermatic/pull/14644))


## v2.27.4

**GitHub release: [v2.27.4](https://github.com/kubermatic/kubermatic/releases/tag/v2.27.4)**

### ACTION REQUIRED
- Update cert-manager to v1.16.5. In the cert-manager values.yaml, following updates should be done ([#14400](https://github.com/kubermatic/kubermatic/pull/14400))
    - update  `webhook.replicas` to `webhook.replicaCount`
    - update  `cainjector.replicas` to `webhook.replicaCount`
    - remove `webhook.injectAPIServerCA`

### Supported Kubernetes versions

- Add 1.32.4/1.31.8/1.30.12 to the list of supported Kubernetes releases ([#14385](https://github.com/kubermatic/kubermatic/pull/14385))

### New Features

- Support infra storage classes and provider network subnets location compatibilities ([#7303](https://github.com/kubermatic/dashboard/pull/7303))

### Bugfixes

- Ensure that etcd backup images are pulled from the overwrite Registry in air-gapped environments ([#14356](https://github.com/kubermatic/kubermatic/pull/14356))
- Fix a bug for KubeLB where disabling the ingress class for a user cluster was not working ([#14396](https://github.com/kubermatic/kubermatic/pull/14396))
- Remove old warnings for new dex chart ([#14423](https://github.com/kubermatic/kubermatic/pull/14423))
- Add role prioritization: Update logic to return the highest-priority role for members with multiple roles ([#7272](https://github.com/kubermatic/dashboard/pull/7272))
- Add special characters restriction on Inputs and escape values to avoid rendering as HTML ([#7273](https://github.com/kubermatic/dashboard/pull/7273))
- Disable the Cluster Autoscaler option when the cluster autoscaler application is not defined in applications catalog ([#7283](https://github.com/kubermatic/dashboard/pull/7283))
- Make the Subnets field required when a VPC is selected, in both Wizard and Machine Deployment modes ([#7305](https://github.com/kubermatic/dashboard/pull/7305))

### Updates

- Add Cert-manager version v1.16.5 in the default applications catalog ([#14418](https://github.com/kubermatic/kubermatic/pull/14418))
- Security: Update Cilium to 1.15.16 / 1.16.9 because the previous versions are affected by CVE-2025-32793 ([#14436](https://github.com/kubermatic/kubermatic/pull/14436))
- Support MatchSubnetAndStorageLocation  and Subnets Regions and Zones ([#14414](https://github.com/kubermatic/kubermatic/pull/14414))
- Update oauth2-proxy to v7.8.2 ([#14388](https://github.com/kubermatic/kubermatic/pull/14388))
- Update OSM version to [v1.6.5](https://github.com/kubermatic/operating-system-manager/releases/tag/v1.6.5) ([#14412](https://github.com/kubermatic/kubermatic/pull/14412))
- Update KubeLB CCM to [v1.1.4](https://docs.kubermatic.com/kubelb/v1.1/release-notes/#v114) ([#14366](https://github.com/kubermatic/kubermatic/pull/14366))

## v2.27.3

**GitHub release: [v2.27.3](https://github.com/kubermatic/kubermatic/releases/tag/v2.27.3)**

### Supported Kubernetes Versions

- Add 1.32.3/1.31.7/1.30.11 to the list of supported Kubernetes releases ([#14266](https://github.com/kubermatic/kubermatic/pull/14266))

### New Features

- A new field `spec.datacenters.<example-dc>.spec.kubevirt.enableDedicatedCpus` is added to seed crd to control whether KubeVirt machine cpus are configured by `spec.template.spec.domain.resources` with requests and limits or `spec.template.spec.domain.cpu` . Later one is required to use KubeVirt cpu allocation ratio feature ([#14298](https://github.com/kubermatic/kubermatic/pull/14298))
- Ensure `mirror-images` processes all images without blocking, logging failed images at the end for better visibility and debugging ([#14279](https://github.com/kubermatic/kubermatic/pull/14279))
- The KKP API is now aware on how to configure cpus for KubeVirt virtual machines based on a new introduced field in kkp seed crd called `spec.datacenters.<example-dc>.spec.kubevirt.enableDedicatedCpus` ([#7264](https://github.com/kubermatic/dashboard/pull/7264))

### Bugfixes

- Node-local-dns in user clusters will now use `IfNotPresent` pull policy instead of `Always` ([#14309](https://github.com/kubermatic/kubermatic/pull/14309))

### Updates

- Update etcd to 3.5.17 for all supported Kubernetes releases ([#14338](https://github.com/kubermatic/kubermatic/pull/14338))
- Update MC version to [v1.61.1](https://github.com/kubermatic/machine-controller/releases/tag/v1.61.1) ([#14339](https://github.com/kubermatic/kubermatic/pull/14339))
- Update OSM to [1.6.4](https://github.com/kubermatic/operating-system-manager/releases/tag/v1.6.4) ([#14332](https://github.com/kubermatic/kubermatic/pull/14332))
- Update the default application's nginx ingress controller to use the save and patched version of v1.12.1 ([#14341](https://github.com/kubermatic/kubermatic/pull/14341))

## v2.27.2

**GitHub release: [v2.27.2](https://github.com/kubermatic/kubermatic/releases/tag/v2.27.2)

### Bugfixes

- Edge Provider: Fix a bug where clusters were stuck in `creating` phase due to wrongfully waiting for Machine Controller's health status ([#14257](https://github.com/kubermatic/kubermatic/pull/14257))
- Fix a bug that prevents configuring `resources` in KNP deployments ([#14205](https://github.com/kubermatic/kubermatic/pull/14205))
- Fix a Go panic when using git-source in Applications ([#14230](https://github.com/kubermatic/kubermatic/pull/14230))
- Fix an issue where the CBSL status was not updating due to the missing cluster-backup-storage-controller in the master controller manager ([#14256](https://github.com/kubermatic/kubermatic/pull/14256))
- Fix mirroring the images of a single Kubernetes version ([#14252](https://github.com/kubermatic/kubermatic/pull/14252))
- It is now possible to configure the sidecar configuration for a given cluster while the auditLogging field is enabled at the Seed level. Previously, if the auditLogging field was enabled at the Seed level, it would override the same field at the Cluster level, resulting in the removal of the sidecar configuration ([#14145](https://github.com/kubermatic/kubermatic/pull/14145))
- Update Dashboard API to use correct OSP which is selected while creating a cluster ([#7217](https://github.com/kubermatic/dashboard/pull/7217))

### Updates

- Security: Update nginx-ingress-controller to 1.11.5, fixing CVE-2025-1097, CVE-2025-1098, CVE-2025-1974, CVE-2025-24513, CVE-2025-24514 ([#14274](https://github.com/kubermatic/kubermatic/pull/14274))

## v2.27.1

**GitHub release: [v2.27.1](https://github.com/kubermatic/kubermatic/releases/tag/v2.27.1)**

### New Features

- Support `infra-csi-driver` as a `volumeProvisioner` for the KubeVirt CSI Driver ([#14199](https://github.com/kubermatic/kubermatic/pull/14199))

### Bugfixes

- Add dex and gitops charts to the CI release pipeline for inclusion in the release tar ([#14192](https://github.com/kubermatic/kubermatic/pull/14192))
- Apply override registry configuration to cilium-envoy images ([#14164](https://github.com/kubermatic/kubermatic/pull/14164))
- Include the etcd backup restore and delete images in the kubermatic-installer mirror-images command ([#14220](https://github.com/kubermatic/kubermatic/pull/14220))

### Updates

- Disable cilium-envoy daemonset, if it was not specified in the chart values ([#14203](https://github.com/kubermatic/kubermatic/pull/14203))
- Update KubeVirt CSI Driver Operator to v0.4.3 ([#14178](https://github.com/kubermatic/kubermatic/pull/14178))

## v2.27.0

**GitHub release: [v2.27.0](https://github.com/kubermatic/kubermatic/releases/tag/v2.27.0)**

Before upgrading, make sure to read the [general upgrade guidelines](https://docs.kubermatic.com/kubermatic/v2.27/installation/upgrading/). Consider tweaking `seedControllerManager.maximumParallelReconciles` to ensure user cluster reconciliations will not cause resource exhaustion on seed clusters. A [full upgrade guide is available from the official documentation](https://docs.kubermatic.com/kubermatic/v2.27/installation/upgrading/upgrade-from-2.26-to-2.27/).

### Breaking Changes

- Remove CentOS as a supported operating system, since it has reached EOL ([#13906](https://github.com/kubermatic/kubermatic/pull/13906))
- VSphere credentials are now handled properly. For existing user cluster, this will change the credentials in machine-controller and OSM to `infraManagementUser` and  `infraManagementPassword` instead of `username` and `password` when specified. The latter one was always mounted to the before mentioned deployments in the past. ([#14087](https://github.com/kubermatic/kubermatic/pull/14087))


### ACTION REQUIRED

- A regression in 2.26.0 started overriding the `floatingIPPool` fields of OpenStack clusters with the default external network. If you are using a floating IP pool that is not the default external network, you might have to update `Cluster` objects manually after upgrading KKP to set the correct floating IP pool again ([#13834](https://github.com/kubermatic/kubermatic/pull/13834))

### API Changes

- Add `spec.componentsOverride.prometheus` to allow overriding Prometheus replicas and tolerations ([#13893](https://github.com/kubermatic/kubermatic/pull/13893))
- Tagged KKP releases will no longer tag the KKP images twice (with the Git tag and the Git hash), but only once with the Git tag. This is so that existing hash-based container images do not suddenly change when a Git tag is set and the release job is run. Users of tagged KKP releases are not affected by this change ([#13763](https://github.com/kubermatic/kubermatic/pull/13763))

### Supported Kubernetes Versions

- Remove support for Kubernetes 1.28 ([#13968](https://github.com/kubermatic/kubermatic/pull/13968))
- Add support for Kubernetes 1.32 ([#13984](https://github.com/kubermatic/kubermatic/pull/13984))
- Add 1.32.1/1.31.5/1.30.9/1.29.13 to the list of supported Kubernetes releases ([#14059](https://github.com/kubermatic/kubermatic/pull/14059))

#### Supported Versions

- 1.29.13
- 1.30.9
- 1.31.5
- 1.32.1

### Cloud Providers

#### AWS

- Update AWS cloud-controller-manager to 1.31.1 ([#13838](https://github.com/kubermatic/kubermatic/pull/13838))
- Update AWS EBS CSI addon to 1.32.0 ([#13508](https://github.com/kubermatic/kubermatic/pull/13508))

#### Azure

- Add `images` field to DatacenterSpecAzure, allowing configuration of default OS images for Azure datacenters in Seeds ([#13924](https://github.com/kubermatic/kubermatic/pull/13924))


#### GCP

- Bump GCP CSI driver to 1.15.0 ([#13800](https://github.com/kubermatic/kubermatic/pull/13800))

#### KubeVirt

- Add KubeVirt DS in the charts repo to generate images for the mirrored images command ([#14064](https://github.com/kubermatic/kubermatic/pull/14064))
- Add `VolumeProvisioner` field in the InfraStorageClasses for the KubeVirt provider in the seed, to indicate whether the storage class can be used by the CSI or the CDI ([#14111](https://github.com/kubermatic/kubermatic/pull/14111))
- Support `CSIDriverOperator` field in the seed object to customize csi driver images in the KubeVirt user cluster ([#14147](https://github.com/kubermatic/kubermatic/pull/14147))
- Support KubeVirt VMs LiveMigrate as an eviction strategy ([#14076](https://github.com/kubermatic/kubermatic/pull/14076))
- Bump the new KubeVirt csi controller operator and add supports to infra storage class labels ([#13827](https://github.com/kubermatic/kubermatic/pull/13827))
- Include KubeVirt CCM and Fluent-Bit images in the mirror-images command ([#14063](https://github.com/kubermatic/kubermatic/pull/14063))
- Remove storage classes filtration in KubeVirt Namespaced mode ([#13985](https://github.com/kubermatic/kubermatic/pull/13985))
- Bump KubeVirt CSI Driver operator to v0.4.2 in KKP ([#13833](https://github.com/kubermatic/kubermatic/pull/13833), [#14011](https://github.com/kubermatic/kubermatic/pull/14011), [#14142](https://github.com/kubermatic/kubermatic/pull/14142))
    * Bump KubeVirt csi-driver-operator to support zone-aware topologies
    * Add support for KubeVirt csi driver operator images overwrite
- Add support for KubeVirt provider network ([#13791](https://github.com/kubermatic/kubermatic/pull/13791))
- Setup KubeVirt network controller in the seed-controller-manager ([#13858](https://github.com/kubermatic/kubermatic/pull/13858))
- Support KubeVirt VolumeBindingMode in the tenant storage class ([#13821](https://github.com/kubermatic/kubermatic/pull/13821))
- Update kubevirt csi driver ([#13818](https://github.com/kubermatic/kubermatic/pull/13818))

#### OpenStack

- Remove redundant storage classes from OpenStack CSI addon ([#13920](https://github.com/kubermatic/kubermatic/pull/13920))
- Upgrade cloud-provider-openstack for v1.30 and v1.31 ([#13899](https://github.com/kubermatic/kubermatic/pull/13899))
- Mark `domain` as optional field for OpenStack preset ([#13948](https://github.com/kubermatic/kubermatic/pull/13948))
- Add `NodePortsAllowedIPRanges` option on the seed datacenter level, giving an option to override default setting for all user clusters on OpenStack provider ([#14029](https://github.com/kubermatic/kubermatic/pull/14029))

#### Packet

- Use official Equinix SDK instead of deprecated packethost/packngo ([#13851](https://github.com/kubermatic/kubermatic/pull/13851))

#### VSphere

- Default datastore validation is skipped, if datastore cluster is specified for the user clusters on vSphere ([#13708](https://github.com/kubermatic/kubermatic/pull/13708))
- Update vSphere CSI Driver to v3.3.1, use official container images from registry.k8s.io again ([#13801](https://github.com/kubermatic/kubermatic/pull/13801))

### New Features

- Add `mirrorImages` field to KubermaticConfiguration for Image Mirroring ([#14140](https://github.com/kubermatic/kubermatic/pull/14140))
- Add `mla-skip-logging` option into `kubermatic-installer` to exclude the logging stack from installation ([#14032](https://github.com/kubermatic/kubermatic/pull/14032))
- Add AIKit in the default application catalogue ([#14121](https://github.com/kubermatic/kubermatic/pull/14121))
- Add an early testing version of ArgoCD based GitOps management for various deployments in Seed ([#13705](https://github.com/kubermatic/kubermatic/pull/13705))
- Control plane components will now automatically be rotated when the user cluster cloud credentials are changed ([#13703](https://github.com/kubermatic/kubermatic/pull/13703))
- If autoscaler should continue to be managed by KKP, the cluster autoscaler app have to be installed to the user cluster. A purge of addon resources will be done before installing the app with helm ([#14057](https://github.com/kubermatic/kubermatic/pull/14057))
- [EE] It is now possible to use pre defined values in the KKP Enterprise Edition for applicationdefinition and applicationinstallation resources to specify cluster related values in the helm release value configuration ([#13945](https://github.com/kubermatic/kubermatic/pull/13945))
- Support `ZoneAndRegionEnable` field in the CCM cloud config ([#13876](https://github.com/kubermatic/kubermatic/pull/13876))
- Support Provider Network VPCs and Subnets in the Seed Object ([#13857](https://github.com/kubermatic/kubermatic/pull/13857))
- The download archives on GitHub now include the dependencies of all included Helm charts ([#13954](https://github.com/kubermatic/kubermatic/pull/13954))

### Bugfixes

- Turned off the Velero backups as well as snapshots by default ([#13940](https://github.com/kubermatic/kubermatic/pull/13940))
    * If you have provided configuration for new velero chart in KKP 2.26 for creating etcd backups and/or volume backups, please ensure to set `velero.backupsEnabled: true` and `velero.snapshotsEnabled: true` explicitly in your custom `values.yaml`
    * The node-agent daemonset is by default disabled. If you had configured volume backups via `velero.snapshotsEnabled: true`, you also need to enabled `velero.deployNodeAgent: true` for volume backups to work
- Mirror system applications helm chart images ([#14126](https://github.com/kubermatic/kubermatic/pull/14126))
    * Update aws-node-termination-handler to v1.24.0
    * `kubermatic-installer mirror-images` command now also mirrors system applications helm charts
- [EE] Fix ClusterBackupStorageLocation sync on the remote seed clusters ([#13955](https://github.com/kubermatic/kubermatic/pull/13955))
- [EE] Fix KubeLB cleanup not being performed when clusters are deleted ([#13960](https://github.com/kubermatic/kubermatic/pull/13960))
- Add validation for IP that's shown in the terminal for ingress and DNS configuration. Hostname is preferred, if it is not present then public IP is preferred over private IP ([#13865](https://github.com/kubermatic/kubermatic/pull/13865))
- Change to a cluster object were not always applied immediately to the user cluster resources ([#13795](https://github.com/kubermatic/kubermatic/pull/13795))
- Disable `/metrics` endpoint in the master/seed MLA and user cluster MLA charts for Grafana ([#13939](https://github.com/kubermatic/kubermatic/pull/13939))
- Do not add `InTree*Unregister` feature gates to the clusters on Kubernetes 1.30+ ([#13983](https://github.com/kubermatic/kubermatic/pull/13983))
- Fix a bug where ca-bundle was not being used to communicate to MinIO for metering ([#14072](https://github.com/kubermatic/kubermatic/pull/14072))
- Fix CA bundle for kubermatic-operator ([#14146](https://github.com/kubermatic/kubermatic/pull/14146))
- Fix Cluster Backup feature not provisioning Velero inside the user clusters ([#13897](https://github.com/kubermatic/kubermatic/pull/13897))
- Fix cluster credentials not being synced into the cluster namespaces, whenever a Secret is updated in the KKP namespace ([#13819](https://github.com/kubermatic/kubermatic/pull/13819))
- CustomOperatingSystemProfiles are now applied more consistently and earlier, when creating new user clusters ([#13831](https://github.com/kubermatic/kubermatic/pull/13831))
- Fix initial sync for CustomOperatingSystemProfiles, when creating new user clusters (follow-up to #13831) ([#13895](https://github.com/kubermatic/kubermatic/pull/13895))
- Fix KKP version check for CRD upgrades, when using commit-based images ([#13762](https://github.com/kubermatic/kubermatic/pull/13762))
- Fix misleading CNI application check error ([#14034](https://github.com/kubermatic/kubermatic/pull/14034))
- Fix node label overwriting issue with the initial Machine Deployment ([#14033](https://github.com/kubermatic/kubermatic/pull/14033))
- Fix seed controller panic, while creating `nodeport-proxy-envoy` deployment for the user clusters ([#13835](https://github.com/kubermatic/kubermatic/pull/13835))
- Fix Seed-MLA's Prometheus trying to scrape Kopia job Pods from Velero ([#14007](https://github.com/kubermatic/kubermatic/pull/14007))
- Fix the issue with blocked cluster provisioning, when selecting default applications that conflicted with Cilium system application and user-cluster-controller-manager was stuck ([#13870](https://github.com/kubermatic/kubermatic/pull/13870))
- Helm releases managed by application installation resources are first deployed when there is at least one node registered to the user cluster ([#14109](https://github.com/kubermatic/kubermatic/pull/14109))
- Ignore reading the storage classes from the infra cluster and only roll out the ones from the seed object ([#14023](https://github.com/kubermatic/kubermatic/pull/14023))
- KubeLB: rely only on cluster spec for `enable-gateway-api` and `use-loadbalancer-class` flags for KubeLB CCM ([#13947](https://github.com/kubermatic/kubermatic/pull/13947))
- Mount correct `ca-bundle` ConfigMap in kubermatic-seed-controller-manager Deployment on dedicated master/seed environments ([#13938](https://github.com/kubermatic/kubermatic/pull/13938))
- Only applicable, if custom update rules in `KubermaticConfiguration.spec.versions.updates` were defined ([#13750](https://github.com/kubermatic/kubermatic/pull/13750))
    * Custom update rules with `automaticNodeUpdate: true` and `automatic` either absent or explicitly set to "false" will be treated as an automatic update rule.
    * A `KubermaticConfiguration` with automatic update rules (as described above) to a target version that would immediately trigger another automatic update is invalid and will be rejected.
- The created RBAC Role for the csi-driver now grants get for VirtualMachineInstances ([#13967](https://github.com/kubermatic/kubermatic/pull/13967))
- The rollback revision is now set explicit to the last deployed revision, when a helm release managed by an application installation fails and a rollback is executed to avoid errors, when the last deployed revision is not the current minus 1 and history limit is set to 1 ([#13953](https://github.com/kubermatic/kubermatic/pull/13953))
- Refactor Cluster Backup controllers ([#13807](https://github.com/kubermatic/kubermatic/pull/13807))
    * Fix ClusterBackupStorageLocations not being synchronized from the master to seed clusters.
    * Refactor Cluster Backups: The controllers for this feature now run in the master-controller-manager and usercluster-controller-manager instead of the seed-controller-manager
- K8sgpt-operator has been introduced to replace the now deprecated k8sgpt(non-operator) application. K8sgpt application will be removed in the future releases ([#14113](https://github.com/kubermatic/kubermatic/pull/14113))
- Update Cluster-Backup Velero CRDs ([#13898](https://github.com/kubermatic/kubermatic/pull/13898))

### Updates

- Bump MC to [v1.61.0](https://github.com/kubermatic/machine-controller/releases/tag/v1.61.0), OSM to [1.6.0](https://github.com/kubermatic/operating-system-manager/releases/tag/v1.6.0)  ([#13811](https://github.com/kubermatic/kubermatic/pull/13811), [#14092](https://github.com/kubermatic/kubermatic/pull/14092))
    * Add support for Ubuntu 24.04.
    * Update machine-controller to v1.61.0
    * Update operating-system-manager to v1.6.0
- Update Konnectivity version tags to match corresponding Kubernetes cluster versions ([#13852](https://github.com/kubermatic/kubermatic/pull/13852))
- Update the applications from the default application catalog to newer versions ([#14067](https://github.com/kubermatic/kubermatic/pull/14https://github.com/kubermatic/kubermatic/pull/14067))
- [EE] The Applications in the Default Application Catalog now use the `defaultValuesBlock` instead of the `defaultValues` ([#13820](https://github.com/kubermatic/kubermatic/pull/13820))
- Add support for Canal in 3.29 version, deprecating v3.26 ([#14051](https://github.com/kubermatic/kubermatic/pull/14051))
- Add support for Cilium in 1.16.6 version, deprecating 1.13 version ([#14048](https://github.com/kubermatic/kubermatic/pull/14048))
- Update Cilium and Trivy ([#13832](https://github.com/kubermatic/kubermatic/pull/13832))
    * Security: Update Cilium to 1.14.16 / 1.15.10 because the previous versions are affected by CVE-2024-47825
- Update cloud-controller-managers to their latest releases. Azure and OpenStack now use the 1.31.x CCMs for 1.31 clusters ([#13854](https://github.com/kubermatic/kubermatic/pull/13854))
- Bump golang.org/x/net to 0.33.0 (CVE-2024-45338) ([#13961](https://github.com/kubermatic/kubermatic/pull/13961))
- Bump KubeLB to v1.1.2 ([#13809](https://github.com/kubermatic/kubermatic/pull/13809))
- Bump oauth2-proxy to 7.7.0 ([#13788](https://github.com/kubermatic/kubermatic/pull/13788))
- Bump Go to 1.23.6 ([#14124](https://github.com/kubermatic/kubermatic/pull/14124))

### Cleanup

- Remove redundant resources and code references for cluster-autoscaler ([#14114](https://github.com/kubermatic/kubermatic/pull/14114))
- Set `METAL_` environment variables instead of `PACKET_` for machine-controller and KubeOne ([#13825](https://github.com/kubermatic/kubermatic/pull/13825))
### Deprecation

- Add new `dex` Helm chart to replace the `oauth` Chart; the new chart uses the official upstream Dex chart, but is preconfigured for use in KKP ([#13486](https://github.com/kubermatic/kubermatic/pull/13486))
- Remove deprecated v3.19 and v3.20 Canal addons ([#14075](https://github.com/kubermatic/kubermatic/pull/14075))
- Remove a long deprecated and ineffective flag (`--docker-binary`) from kubermatic-installer `mirror-images` subcommand ([#14110](https://github.com/kubermatic/kubermatic/pull/14110))


### Dashboard and API

#### Breaking Changes

- Remove CentOS as a supported operating system since it has reached EOL ([#7026](https://github.com/kubermatic/dashboard/pull/7026))

#### Cloud Providers

##### Edge

- Fix datacenter creation for Edge provider ([#7165](https://github.com/kubermatic/dashboard/pull/7165))

##### GCP

- Fix wrong GCP machine deployment values in Edit Machine Deployment dialog ([#7169](https://github.com/kubermatic/dashboard/pull/7169))

##### KubeVirt

- Filter out KubeVirt storage classes in the API/UI based on the the volume provisioner field ([#7144](https://github.com/kubermatic/dashboard/pull/7144))
- Support EvictionStrategy field in KubeVirt provider ([#7129](https://github.com/kubermatic/dashboard/pull/7129))
- Support for Kube-OVN subnet and VPCs for KubeVirt ([#6941](https://github.com/kubermatic/dashboard/pull/6941))
- Support Kube-OVN provider networks for VPCs and Subnets ([#6915](https://github.com/kubermatic/dashboard/pull/6915))
- Use infra namespace from datacenter configuration, if specified ([#6949](https://github.com/kubermatic/dashboard/pull/6949))
- Read KubeVirt StorageClasses from the seed object instead of reading them from the infra cluster directly ([#7000](https://github.com/kubermatic/dashboard/pull/7000))
- Support reading OVN VPCs and Subnets from the seed CR ([#6982](https://github.com/kubermatic/dashboard/pull/6982))

##### Openstack

- Make `Domain` field optional when using application credentials for OpenStack provider ([#7044](https://github.com/kubermatic/dashboard/pull/7044))

##### VMWare Cloud Director
- Select correct template value when editing MD of VCD provider ([#6927](https://github.com/kubermatic/dashboard/pull/6927))

##### VSphere

- Fix a bug where updating replicas of machine deployments was causing machine rotation ([#7130](https://github.com/kubermatic/dashboard/pull/7130))

#### New Features

- Add a new `postfix_page_title` property in the UI config to allow setting a custom postfix for the page title ([#6980](https://github.com/kubermatic/dashboard/pull/6980))
- Add a new admin announcement feature. Admins can broadcast messages to users, displayed as banners on all pages, with a full list accessible through the help panel ([#7067](https://github.com/kubermatic/dashboard/pull/7067))
- Add API endpoints for managing Backup Storage Locations in the user clusters ([#7108](https://github.com/kubermatic/dashboard/pull/7108))
- Add functionality to handle default namespaces for application installation and application resources ([#7146](https://github.com/kubermatic/dashboard/pull/7146))
- Add functionality to import cluster backups from S3 using cluster backup storage location ([#7111](https://github.com/kubermatic/dashboard/pull/7111))
- Add new option for cluster autoscaling in the initial nodes step of cluster wizard to add cluster autoscaler application ([#7128](https://github.com/kubermatic/dashboard/pull/7128))
- KubeLB: Enable gateway API and use load balancer class values will now be picked from the datacenter configuration during cluster creation ([#7055](https://github.com/kubermatic/dashboard/pull/7055))

#### Design

- Add search field in options dropdown of autocomplete element ([#7069](https://github.com/kubermatic/dashboard/pull/7069))
- Display only the admin-allowed OS's on project add/edit dialog ([#6962](https://github.com/kubermatic/dashboard/pull/6962))

#### Bugfixes

- [EE] Fix Cluster Backups failing because of empty label selectors ([#6971](https://github.com/kubermatic/dashboard/pull/6971))
- Fix a bug where `groups` scope was missing in authentication request for kubernetes-dashboard ([#7014](https://github.com/kubermatic/dashboard/pull/7014))
- Fix an issue where selecting "Backup All Namespaces" in the create backup/schedule dialog for cluster backups caused new namespaces to be excluded ([#7034](https://github.com/kubermatic/dashboard/pull/7034))
- Fix editing project from project overview page ([#6991](https://github.com/kubermatic/dashboard/pull/6991))
- Fix list images in kubevirt and tinkerbell ([#7049](https://github.com/kubermatic/dashboard/pull/7049))
- Fix missing AMI value in edit machine deployment dialog ([#7002](https://github.com/kubermatic/dashboard/pull/7002))
- Fix Velero logs command in error message of failed backups and restores ([#7154](https://github.com/kubermatic/dashboard/pull/7154))
- In the cluster backup feature, fix the issue with restoring a backup that includes all namespaces and add the option to restore all namespaces from a backup ([#7168](https://github.com/kubermatic/dashboard/pull/7168))

#### Updates

- Update stylelint dependencies ([#6961](https://github.com/kubermatic/dashboard/pull/6961))
- Update to Angular version 18 ([#6958](https://github.com/kubermatic/dashboard/pull/6958))
- Bump Go to 1.23.6 ([#7147](https://github.com/kubermatic/dashboard/pull/7147))
