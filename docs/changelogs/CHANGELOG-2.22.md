# Kubermatic 2.22

- [v2.22.0](#v2220)

## [v2.22.0](https://github.com/kubermatic/kubermatic/releases/tag/v2.22.0)

Before upgrading, make sure to read the [general upgrade guidelines](https://docs.kubermatic.com/kubermatic/v2.22/tutorials-howtos/upgrading/). Consider tweaking `seedControllerManager.maximumParallelReconciles` to ensure usercluster reconciliations will not cause resource exhaustion on seed clusters.

### Supported Kubernetes Versions

- Add support for Kubernetes 1.24.8. Those Kubernetes patch releases fix CVE-2022-3162 and CVE-2022-3294, both in kube-apiserver. We strongly recommend upgrading to those Kubernetes patch releases as soon as possible ([#11340](https://github.com/kubermatic/kubermatic/pull/11340))
    - [CVE-2022-3162: Unauthorized read of Custom Resources](https://groups.google.com/g/kubernetes-announce/c/oR2PUBiODNA/m/tShPgvpUDQAJ)
    - [CVE-2022-3294: Node address isn't always verified when proxying](https://groups.google.com/g/kubernetes-announce/c/eR0ghAXy2H8/m/sCuQQZlVDQAJ)
- Add support for Kubernetes v1.24.9 ([#11553](https://github.com/kubermatic/kubermatic/pull/11553))
- Add support for Kubernetes 1.25 ([#11049](https://github.com/kubermatic/kubermatic/pull/11049))
- Add support for Kubernetes 1.26 ([#11621](https://github.com/kubermatic/kubermatic/pull/11621))
- Support for Kubernetes 1.22 and 1.23 user clusters has been removed; User clusters remaining on 1.23 will be automatically upgraded with this KKP version ([#11286](https://github.com/kubermatic/kubermatic/pull/11286), [#11767](https://github.com/kubermatic/kubermatic/pull/11767))
- Allow Kubernetes version upgrade for clusters with non-amd64 nodes & Canal CNI and IPVS for all Kubernetes versions ([#11765](https://github.com/kubermatic/kubermatic/pull/11765))

#### Supported Versions

- 1.24.8
- 1.24.9
- 1.25.4 
- 1.25.5
- 1.26.0

### Highlights

#### KubeVirt

KubeVirt cloud provider support is leaving the "technical preview" phase and is now considered GA.

- Fix to ensure that we do not raise an error when reconciling the namespace in the infrastructure KubeVirt cluster, until we get the value of the namespace to create, avoiding transient errors ([#10849](https://github.com/kubermatic/kubermatic/pull/10849))
- Bugfix for KubeVirt infra CSI token creation due to auto-creation disabled in k8s 1.24 ([#10908](https://github.com/kubermatic/kubermatic/pull/10908))
- Introduction of KubeVirt default instance types and instance preferences that will replace virtual machine presets ([#11025](https://github.com/kubermatic/kubermatic/pull/11025))
- Fix wrong quota filtering when `VirtualMachineInstancePreset.spec.cpu` has no quantity but only other fields ([#11046](https://github.com/kubermatic/kubermatic/pull/11046))
- Add support for ToplogySpreadConstraint for Kubevirt VM ([#11114](https://github.com/kubermatic/kubermatic/pull/11114))
- Add support for instancetype/preference ([#11182](https://github.com/kubermatic/kubermatic/pull/11182))
- Update KubeVirt CCM to v0.4.0 and kubevirt.io/api to v0.58.0 ([#11249](https://github.com/kubermatic/kubermatic/pull/11249))
- Add Images option in KubeVirt DataCenter which supports download and caching of VM images from HTTP source ([#11302](https://github.com/kubermatic/kubermatic/pull/11302))
- Add DataVolumeSize field to seed and ImageCloningEnabled to cluster ([#11651](https://github.com/kubermatic/kubermatic/pull/11651))
- Switch StorageClasses init configuration from annotation to DC ([#11716](https://github.com/kubermatic/kubermatic/pull/11716))
- Update KubeVirt DC: add additional network policies configuration ([#11659](https://github.com/kubermatic/kubermatic/pull/11659))
- Update seed and cluster to config StorageClasses from seed ([#11701](https://github.com/kubermatic/kubermatic/pull/11701))
- Graceful user cluster workload eviction in case of bare-metal infra node draining ([#11588](https://github.com/kubermatic/kubermatic/pull/11588))
- Split CSI driver deployment between user and infra cluster and remove user cluster CSI access to KubeVirt cluster ([#11370](https://github.com/kubermatic/kubermatic/pull/11370))
- Fix KubeVirt LB issue (wrong custer-isolation netpol): LB not accessible from outside user-cluster ([#11640](https://github.com/kubermatic/kubermatic/pull/11640))
- Change CustomNetworkPolicies type (extract name) ([#11666](https://github.com/kubermatic/kubermatic/pull/11666))

#### Konnectivity

Konnectivity is now GA.

- Set PriorityClassName of konnectivity-agent and openvpn-client to `system-cluster-critical` ([#11140](https://github.com/kubermatic/kubermatic/pull/11140))
- Add keepalive-time Konnectivity setting + set keepalive to 1m by default ([#11502](https://github.com/kubermatic/kubermatic/pull/11502))
- Remove `KonnectivityService` feature gate & make Konnectivity generally available ([#11643](https://github.com/kubermatic/kubermatic/pull/11643))
- Update Konnectivity version to v0.0.35 ([#11657](https://github.com/kubermatic/kubermatic/pull/11657))

#### Kubermatic Operator

- The KKP operator can now perform the complete initial setup for new seed clusters. If MinIO/S3-exporter are not required, the KKP installer does not need to be used for setting up new / updating existing seed clusters ([#10795](https://github.com/kubermatic/kubermatic/pull/10795))

#### Presets

- Add `.spec.projects` field to `Preset` resources to allow binding Presets to specific projects ([#11100](https://github.com/kubermatic/kubermatic/pull/11100))

#### OIDC Support

- Add groups in OIDC kubeconfig ([#11121](https://github.com/kubermatic/kubermatic/pull/11121))
- Add `OIDCProviderConfiguration` to Seed's spec allowing to configure dedicated OIDC provider for each Seed ([#11668](https://github.com/kubermatic/kubermatic/pull/11668))

#### OS Support

- KKP now defaults to Ubuntu 22.04 LTS when Ubuntu is selected as an operating system ([#11007](https://github.com/kubermatic/kubermatic/pull/11007))
- KKP no longer supports SLES operating system ([#11711](https://github.com/kubermatic/kubermatic/pull/11711))

#### Resource Quotas (EE)

- Add a default project resource quota setting which can be set in KKP's global `KubermaticSettings`. By managing the default quota, for all the projects which do not have a custom quota already set, their ResourceQuota is created/updated/deleted ([#11582](https://github.com/kubermatic/kubermatic/pull/11582))

### Breaking Changes

- Use `registry.k8s.io` instead of `k8s.gcr.io` for Kubernetes upstream images. It might be necessary to update firewall rules or mirror registries accordingly ([#11079](https://github.com/kubermatic/kubermatic/pull/11079))
- KubeVirt: Manual migration of existing MD required. For more information [see our docs](https://docs.kubermatic.com/kubermatic/v2.22/architecture/supported-providers/kubevirt/kubevirt/#migration-from-kkp-221-to-kkp-222) ([#11430](https://github.com/kubermatic/kubermatic/pull/11430))
- KubeVirt: remove Flavor handling in favour of Instancetype/Preference. action required: manual update of the MD (refer to [our documentation](https://docs.kubermatic.com/kubermatic/v2.22/architecture/supported-providers/kubevirt/kubevirt/#migration-from-kkp-221-to-kkp-222)) ([#11398](https://github.com/kubermatic/kubermatic/pull/11398))
- Remove experimental support for Thanos in the Prometheus Helm chart ([#11424](https://github.com/kubermatic/kubermatic/pull/11424))
- Cloud provider specific configurations are prohibited from seed-scoped default cluster templates; for existing seed-scoped default cluster templates these settings should be removed manually ([#11472](https://github.com/kubermatic/kubermatic/pull/11472))
- OperatingSystemProfiles and OperatingSystemConfigs have been moved to the user clusters ([#11710](https://github.com/kubermatic/kubermatic/pull/11710))
- CustomOperatingSystemProfile CRD is introduced for maintaining custom OSPs at the seed level. For more information [see our docs](https://docs.kubermatic.com/kubermatic/v2.22/tutorials-howtos/operating-system-manager/usage/#custom-operatingsystemprofiles) ([#11720](https://github.com/kubermatic/kubermatic/pull/11720))
    - OSP and OSC resources have been moved to user clusters. KKP will take care of migrating the existing resources
    - Custom OperatingSystemProfiles should now be created for the kind `CustomOperatingSystemProfile` instead of `OperatingSystemProfile` in the seed namespace

### API Changes

- Extend disk configuration for provider Anexia ([#10816](https://github.com/kubermatic/kubermatic/pull/10816))
- Add external cluster EKS/AKS/GKE provider configuration into the `ExternalCluster` CRD ([#10982](https://github.com/kubermatic/kubermatic/pull/10982))
- The `address` field in the Cluster CRD was deprecated in KKP 2.21 and removed in this release. Use `status.address` instead. Existing clusters were migrated automatically by the seed-controller-manager in release 2.21 ([#10906](https://github.com/kubermatic/kubermatic/pull/10906))
- Instead of an `apiv1.NodeDeployment`, a `clusterv1alpha1.MachineDeployment` must be stored in the `kubermatic.io/initial-machinedeployment-request` annotation on new clusters ([#11339](https://github.com/kubermatic/kubermatic/pull/11339))
- Seed spec no longer requires `defaultDestination` for `etcdBackupRestore`; Omitting it allows to disable default etcd backups ([#11594](https://github.com/kubermatic/kubermatic/pull/11594))
- `availabilityZone`, `dnsServers` and `nodeSizeRequirements` are now optional in the Openstack datacenter spec ([#11605](https://github.com/kubermatic/kubermatic/pull/11605))

### Cloud Providers

- The cloud-config and cloud-config-csi for user clusters is now stored in a `Secret` instead of `ConfigMap` on the seed cluster (in each user cluster namespace) ([#10759](https://github.com/kubermatic/kubermatic/pull/10759))
- By default, the `externalCloudProvider` feature will be turned on for all newly created clusters (if the provider supports it). Previously this only applied to Anexia/Kubevirt clusters (and to VSphere/Openstack/Azure clusters when using the KKP dashboard). Now the same rules apply no matter how the cluster is provisioned. The feature gate can still be explicitly set to `false`, but eventually all clusters must be migrated to the external CCM/CSI as in-tree support in Kubernetes is phased out ([#11095](https://github.com/kubermatic/kubermatic/pull/11095))

#### AWS

- The AWS and EKS cloud provider implementations now use the AWS Go SDK v2 to communicate with AWS ([#10922](https://github.com/kubermatic/kubermatic/pull/10922))
- Add support for the AWS External CCM & EBS CSI driver for Kubernetes 1.24+; newly created AWS clusters will use the `externalCloudProvider` feature flag automatically, existing clusters must be migrated manually ([#11102](https://github.com/kubermatic/kubermatic/pull/11102))
- Introduce a new field `disableIAMReconciling` in AWS cloud spec to disable IAM reconciliation ([#11272](https://github.com/kubermatic/kubermatic/pull/11272))
- Add support for dual-stack node IPs with AWS CCM ([#11285](https://github.com/kubermatic/kubermatic/pull/11285))

#### vSphere

- Remove the `overwriteCloudSpec` field from vSphere Machine Deployment ([#11315](https://github.com/kubermatic/kubermatic/pull/11315))
- Defaulting vSphere tag category from seed, when it is not specified in user cluster ([#11411](https://github.com/kubermatic/kubermatic/pull/11411))
- Rework vSphere user clusters tagging mechanism to adjust tags and tag categories creation and deletion, as part of the vSphere cloud provider resources reconciliation ([#11665](https://github.com/kubermatic/kubermatic/pull/11665))
- Update vSphere CSI driver to v2.7.0 ([#11724](https://github.com/kubermatic/kubermatic/pull/11724))
- Cleanup default tag categories creation and only create tags in the tag category which assigned on the user cluster level ([#11790](https://github.com/kubermatic/kubermatic/pull/11790))

#### GCP

- Add support for GCP CSI Driver in Kubernetes 1.25+ clusters ([#11268](https://github.com/kubermatic/kubermatic/pull/11268))

#### OpenStack

- Support for using server groups with OpenStack ([#11298](https://github.com/kubermatic/kubermatic/pull/11298))
- Add support for enforcing custom disk for OpenStack in KubermaticSettings ([#11338](https://github.com/kubermatic/kubermatic/pull/11338))
- Update OpenStack Cinder CSI to v1.24.5 and v1.25.3 ([#11454](https://github.com/kubermatic/kubermatic/pull/11454))

#### DigitalOcean

- Add support for DigitalOcean external CCM ([#11464](https://github.com/kubermatic/kubermatic/pull/11464))

#### Anexia

- Update Anexia CCM (cloud-controller-manager) to version 1.5.1 ([#11656](https://github.com/kubermatic/kubermatic/pull/11656))

### MLA

- Installation of UserCluster MLA is now integrated with KKP installer via `kubermatic-installer deploy usercluster-mla` command ([#11008](https://github.com/kubermatic/kubermatic/pull/11008))
- Consul now uses the `kubermatic-fast` StorageClass by default ([#11291](https://github.com/kubermatic/kubermatic/pull/11291))
- UserCluster MLA: grafana-agent is now used instead of Prometheus inside the user clusters. Custom rules ConfigMaps should now be prefixed with `monitoring-scraping-` instead of `prometheus-scraping-` ([#11387](https://github.com/kubermatic/kubermatic/pull/11387))
- UserCluster MLA: grafana-agent is now used instead of Promtail inside the user clusters. No action is required ([#11426](https://github.com/kubermatic/kubermatic/pull/11426))
- Update MLA components ([#11580](https://github.com/kubermatic/kubermatic/pull/11580))
    - Update Consul to 1.14.2
    - Update Cortex to 1.13.1
    - Update Grafana to 9.3.1
    - Update Loki to 2.6.1
    - Update minio to RELEASE.2022-09-17T00-09-45Z 

### Applications

- Update `ApplicationDefinition` CRD to handle credentials at "templating" time. This allows downloading helm dependencies from an authenticated registry when application's source is git ([#11452](https://github.com/kubermatic/kubermatic/pull/11452))
- Add new field  `ReconciliationInterval` in `ApplicationInstallation` to force reconciliation, even if the `ApplicationInstallation` CR has not changed ([#11467](https://github.com/kubermatic/kubermatic/pull/11467))
- Extend `ApplicationDefinition` and `ApplicationInstallation` CRD with `DeployOptions.HelmDeployOptions` to control how applications are deployed with `Helm`([#11608](https://github.com/kubermatic/kubermatic/pull/11608))
    - ApplicationInstallation: set condition ready to `unknown` with reason `InstallationInProgress` before starting the installation
    - ApplicationInstallation: don't try to install / upgrade the application if the max number of retries is exceeded 
- Use string Version type for `ApplicationInstallation` CRD ([#11359](https://github.com/kubermatic/kubermatic/pull/11359))
- Make uninstall for Applications idempotent ([#11622](https://github.com/kubermatic/kubermatic/pull/11622))
- Add validating and defaulting webhook for Application deployOptions ([#11633](https://github.com/kubermatic/kubermatic/pull/11633))

### etcd-launcher

- A race condition bug in `etcd-launcher` that can trigger on user cluster initialisation and that prevents the last etcd node from joining the etcd cluster has been fixed ([#10932](https://github.com/kubermatic/kubermatic/pull/10932))
- EtcdRestores are moved to a '`EtcdLauncherNotEnabled` phase if required etcd-launcher is not enabled ([#11115](https://github.com/kubermatic/kubermatic/pull/11115))
- Feature flag `EtcdLauncher` is enabled by default for `KubermaticConfiguration` ([#11684](https://github.com/kubermatic/kubermatic/pull/11684))

### Metrics & Alerts

- Fix missing etcd metrics in Grafana etcd dashboards and master/seed Prometheus by renaming to: `etcd_mvcc_db_total_size_in_bytes`, `etcd_mvcc_delete_total`, `etcd_mvcc_put_total`, `etcd_mvcc_range_total`, `etcd_mvcc_txn_total` ([#11434](https://github.com/kubermatic/kubermatic/pull/11434))
- Monitoring: added etcd database size alerts: EtcdDatabaseQuotaLowSpace, EtcdExcessiveDatabaseGrowth, EtcdDatabaseHighFragmentationRatio ([#11507](https://github.com/kubermatic/kubermatic/pull/11507))
- Add unified event monitoring Grafana dashboard ([#11402](https://github.com/kubermatic/kubermatic/pull/11402))
- Enable alert-management using Grafana ([#11031](https://github.com/kubermatic/kubermatic/pull/11031))

### Expose Strategy

- Remove `TunnelingExposeStrategy` feature gate, Tunneling expose strategy promoted to generally available (GA) ([#11680](https://github.com/kubermatic/kubermatic/pull/11680))
- Fix setting exposeStrategy via KKP cluster API endpoint ([#11061](https://github.com/kubermatic/kubermatic/pull/11061))
- Allow expose strategy migration for existing user clusters ([#11157](https://github.com/kubermatic/kubermatic/pull/11157))
- Fix duplicate SourceRange entries for front-loadbalancer Service ([#11308](https://github.com/kubermatic/kubermatic/pull/11308))
- Add TunnelingAgentIP into ClusterNetwork part of the cluster API ([#11381](https://github.com/kubermatic/kubermatic/pull/11381))
- Implemented network-interface-manager for enhancing Tunneling expose strategy reliability ([#11432](https://github.com/kubermatic/kubermatic/pull/11432))
- Fix default tunnelingAgentIP for Tunneling expose strategy ([#11443](https://github.com/kubermatic/kubermatic/pull/11443))
- Change the Tunneling agent interface default IP from `192.168.30.10` to `100.64.30.10` ([#11504](https://github.com/kubermatic/kubermatic/pull/11504))
- Prioritise public IP over private IP in front LoadBalancer service ([#11512](https://github.com/kubermatic/kubermatic/pull/11512))
- Fix external cluster address in cluster's status.address.ip for the Tunneling expose strategy ([#11687](https://github.com/kubermatic/kubermatic/pull/11687))

### Installer

- Add `--registry-prefix` flag to `kubermatic-installer mirror-images` command ([#11705](https://github.com/kubermatic/kubermatic/pull/11705))
- The `kubermatic-installer` will now reject installing a different KKP edition over an existing one (for example installing the Community Edition over a previous Enterprise Edition). This safety check can be disabled by adding `--allow-edition-change` to the installer. Installing the EE over the CE (i.e. upgrading) is supported, this change just prevents accidental mixups ([#11128](https://github.com/kubermatic/kubermatic/pull/11128))
- Updating KKP requires to change container runtime for all user clusters to containerd beforehand ([#11781](https://github.com/kubermatic/kubermatic/pull/11781))
- The KKP installer will ensure that neither the master nor any seed violate the KKP version skew policy, i.e. skipping a minor release during an upgrade is not permitted. Additionally, all seeds must be healthy for an upgrade to be possible. These changes are to ensure that smaller issues now do not lead to bigger problems during upgrades and migrations ([#10907](https://github.com/kubermatic/kubermatic/pull/10907))
- `kubermatic-installer mirror-images` correctly picks up konnectivity and Kubernetes dashboard images ([#11148](https://github.com/kubermatic/kubermatic/pull/11148))
- Fix `--config` flag not being validated in `kubermatic-installer mirror-images` command in the KKP installer ([#11146](https://github.com/kubermatic/kubermatic/pull/11146))
- Fix `kubermatic-installer convert-kubeconfig` installer command not generating a SA token ([#11197](https://github.com/kubermatic/kubermatic/pull/11197))
- Fix `kubermatic-installer print` always printing the CE version of the example YAMLs ([#11129](https://github.com/kubermatic/kubermatic/pull/11129))
- Installer subcommand `mirror-images` correctly mirrors image `kubernetesui/metrics-scraper` now ([#11207](https://github.com/kubermatic/kubermatic/pull/11207))
- Observe configured addons tag suffix when extracting addon images in `kubermatic-installer mirror-images` command ([#11702](https://github.com/kubermatic/kubermatic/pull/11702))
- Remove dependency on `docker` binary when using `kubermatic-installer mirror-images` (removes the `--docker-binary` flag) ([#11717](https://github.com/kubermatic/kubermatic/pull/11717))
- Do not require addons flags in `kubermatic-installer mirror-images` and fall back to default addons image ([#11135](https://github.com/kubermatic/kubermatic/pull/11135))
- Ignore repository overrides in `KubermaticConfiguration` by default when mirroring images with `kubermatic-installer mirror-images` (can be disabled with `--ignore-repository-overrides=false`) ([#11703](https://github.com/kubermatic/kubermatic/pull/11703))

### New Features

- Add `spec.version` and `spec.cloudSpec.kubeone.region` fields in External cluster CRD ([#11644](https://github.com/kubermatic/kubermatic/pull/11644))
- Add `types` parameter, defining report types, to the `/api/v1/admin/metering/configurations/reports` POST and PUT ([#10889](https://github.com/kubermatic/kubermatic/pull/10889))
- Add Cilium CNI values validation ([#11506](https://github.com/kubermatic/kubermatic/pull/11506))
- Add option for configuring OCI Helm repository for storing system Applications (e.g. Cilium CNI) ([#11708](https://github.com/kubermatic/kubermatic/pull/11708))
- Add support for kube-dns configmap for NodeLocal DNSCache to allow customization of dns.Fixes an issue with a wrong mounted Corefile in NodeLocal DNSCache ([#11664](https://github.com/kubermatic/kubermatic/pull/11664))
- External clusters on EKS now support assume role ([#11259](https://github.com/kubermatic/kubermatic/pull/11259))
- Manage Cilium CNI via Applications infra ([#11414](https://github.com/kubermatic/kubermatic/pull/11414))
- OPA integration: allow to define enforcementAction in KKP's constraint.EnforcementAction defines the action to take in response to a constraint being violated.By default, EnforcementAction is set to deny as the default behavior is to deny admission requests with any violation ([#11723](https://github.com/kubermatic/kubermatic/pull/11723))
- seed proxy: increase memory limit from 32Mi to 64Mi ([#10984](https://github.com/kubermatic/kubermatic/pull/10984))
- Support for setting default OperatingSystemProfiles at the seed level ([#11105](https://github.com/kubermatic/kubermatic/pull/11105))
- Support to filter machines based on resources (CPU, RAM, GPU) per datacenter ([#11130](https://github.com/kubermatic/kubermatic/pull/11130))
- Add Canal CNI version v3.24 ([#11575](https://github.com/kubermatic/kubermatic/pull/11575))

### New Features (EE)

- Add support for GroupProjectBindings in MLA Grafana ([#11076](https://github.com/kubermatic/kubermatic/pull/11076))

### Bugfixes

- No resources are created for the default addon `pod-security-policy` when applied to Kubernetes 1.25 or higher ([#11373](https://github.com/kubermatic/kubermatic/pull/11373))
    - Remove `PodSecurityPolicy` resource from `aws-node-termination-handler` addon
    - Ensure `PodDisruptionBudget` resource in `aws-ebs-csi-driver` addon is created via `policy/v1` API
- Actually make `enableWebTerminal` an optional field ([#11362](https://github.com/kubermatic/kubermatic/pull/11362))
- Container runtime configuration is properly validated while creating or upgrading clusters ([#11780](https://github.com/kubermatic/kubermatic/pull/11780))
- CRDs were missing in the KKP Docker images, making it hard to use the installer in Docker. This was now fixed and the CRDs are available ([#11210](https://github.com/kubermatic/kubermatic/pull/11210))
- Disable promtail initContainer that was overriding system `fs.inotify.max_user_instances` configuration ([#11382](https://github.com/kubermatic/kubermatic/pull/11382))
- Fix an issue where creating Clusters through ClusterTemplates failed without leaving a trace (the ClusterTemplateInstance got deleted as if all was good) ([#11601](https://github.com/kubermatic/kubermatic/pull/11601))
- Fix kubermatic-webhook failing to start on external seed clusters ([#10951](https://github.com/kubermatic/kubermatic/pull/10951))
- Fix kubermatic-webhook panic on providerName mismatch from CloudSpec ([#11236](https://github.com/kubermatic/kubermatic/pull/11236))
- Fix rare CRD conflict when installing old KKP versions into user clusters created by a different KKP version ([#10903](https://github.com/kubermatic/kubermatic/pull/10903))
- Fix rendering error of the metallb addon causing missing L2Advertisement ([#11233](https://github.com/kubermatic/kubermatic/pull/11233))
- Fix seed-proxy ServiceAccount token not being generated ([#11190](https://github.com/kubermatic/kubermatic/pull/11190))
- Fix the issue where AllowedRegistry ConstraintTemplate was not being reconiciled by Gatekeeper because it's `spec.crd` OpenAPI spec was missing a type ([#11327](https://github.com/kubermatic/kubermatic/pull/11327))
- Fix user-ssh-keys-agent Docker image for arm64 containing the amd64 binary ([#11606](https://github.com/kubermatic/kubermatic/pull/11606))
- Fix DNAT controller not installing NAT rules for big clusters ([#11267](https://github.com/kubermatic/kubermatic/pull/11267))
- Improve validation for versioning/update configuration in `KubermaticConfiguration` ([#11749](https://github.com/kubermatic/kubermatic/pull/11749))
- OPA integration: enable status operation by that audit Pod ([#11722](https://github.com/kubermatic/kubermatic/pull/11722))
- Prefer InternalIP when connecting to Kubelet for Hetzner dual-stack clusters ([#10937](https://github.com/kubermatic/kubermatic/pull/10937))
- Prevent index out-of-bounds issue when querying GKE external cluster status ([#11213](https://github.com/kubermatic/kubermatic/pull/11213))
- Properly clean up Kubernetes Dashboard resources in the user cluster if the Kubernetes Dashboard is disabled ([#11574](https://github.com/kubermatic/kubermatic/pull/11574))
- Repo Digests are now dropped when a docker image is changed by using the overwrite-registry mechanism. Previously kept digests could lead to issues with mirrored Docker images ([#11147](https://github.com/kubermatic/kubermatic/pull/11147))
- Set nginx ingress `proxy-read-timeout` and `proxy-send-timeout` to 1 hour to support long-lasting connections (e.g. websocket) ([#11756](https://github.com/kubermatic/kubermatic/pull/11756))
- Use seed proxy configuration for seed-controller-manager ([#11561](https://github.com/kubermatic/kubermatic/pull/11561))
- Paused external clusters will not be processed ([#11447](https://github.com/kubermatic/kubermatic/pull/11447))

### Updates

- Update to Cilium v1.12.2 and v1.11.9 ([#11013](https://github.com/kubermatic/kubermatic/pull/11013))
- Update KubeOne image to v1.5.3 for KubeOne external cluster controller ([#11388](https://github.com/kubermatic/kubermatic/pull/11388))
- Update MetalLB version to v0.13.7 ([#11252](https://github.com/kubermatic/kubermatic/pull/11252))
- Update cert-manager to 1.10.1 ([#11412](https://github.com/kubermatic/kubermatic/pull/11412))
- Update Dex to 2.35.3 ([#11413](https://github.com/kubermatic/kubermatic/pull/11413))
- Update k8s-dns-node-cache to 1.22.13 ([#11287](https://github.com/kubermatic/kubermatic/pull/11287))
- Update nginx-ingress-controller to 1.5.1 ([#11415](https://github.com/kubermatic/kubermatic/pull/11415))
- Update node\_exporter to 1.4.0 ([#11425](https://github.com/kubermatic/kubermatic/pull/11425))
- Update OPA integration Gatekeeper to 3.7.2 ([#11188](https://github.com/kubermatic/kubermatic/pull/11188))
- Update Prometheus to 2.40.2 ([#11423](https://github.com/kubermatic/kubermatic/pull/11423))
- Update to etcd 3.5.6 to prevent potential data inconsistency issues during online defragmentation ([#11403](https://github.com/kubermatic/kubermatic/pull/11403))

### Updates (EE)

- Update metering to version 1.0.1 ([#11282](https://github.com/kubermatic/kubermatic/pull/11282))
    - Add average-used-cpu-millicores to Cluster and Namespace reports
    - Add average-available-cpu-millicores add average-cluster-machines field to Cluster reports
    - Fix a bug that causes wrong values if metric is not continuously present for the aggregation window 
- Fix a bug in metering that lead to outdated Project and/or Cluster labels in reports ([#11743](https://github.com/kubermatic/kubermatic/pull/11743))

### Dashboard & API

- The KKP API is run based on the Dashboard's Docker image now ([#11229](https://github.com/kubermatic/kubermatic/pull/11229))
- Add GET `/api/v2/providers/eks/clusterroles` endpoint to list EKS Cluster Roles ([#10778](https://github.com/kubermatic/kubermatic/pull/10778))
- Add GET `/api/v2/providers/eks/noderoles` endpoint to list EKS Worker Node Roles ([#10939](https://github.com/kubermatic/kubermatic/pull/10939))
- Add GET endpoint `/api/v2/providers/aks/resourcegroups` to list AKS resource groups ([#10921](https://github.com/kubermatic/kubermatic/pull/10921))
- Replace GET `api/v2/providers/eks/noderoles` endpoint with `/api/v2/projects/{project\_id}/kubernetes/clusters/{cluster\_id}/providers/eks/noderoles` endpoint to list EKS NodeRoles ([#10975](https://github.com/kubermatic/kubermatic/pull/10975))
- Add API endpoints to create service accounts and get associated kubeconfig ([#11120](https://github.com/kubermatic/kubermatic/pull/11120))
    - GET, POST `/api/v2/projects/{project_id}/clusters/{cluster_id}/serviceaccount`
    - DELETE `/api/v2/projects/{project_id}/clusters/{cluster_id}/serviceaccount/{namespace}/{service_account_id}`
    - GET `/api/v2/projects/{project_id}/clusters/{cluster_id}/serviceaccount/{namespace}/{service_account_id}/kubeconfig`
- Fix API error in extended disk configuration for provider Anexia ([#11030](https://github.com/kubermatic/kubermatic/pull/11030))
- Fix Openstack `api/v1/providers/openstack/tenants` API endpoint for some cases where "couldn't get projects: couldn't get tenants for region XX: couldn't get identity endpoint: No suitable endpoint could be found in the service catalog." was wrongly returned ([#10968](https://github.com/kubermatic/kubermatic/pull/10968))
- Fix issue in KKP API where deleting all datacenters from a Seed and then trying to add a new one would cause a panic ([#10953](https://github.com/kubermatic/kubermatic/pull/10953))
- Enhance cluster rbac to allow to bind service account to clusterRole and Role. ([#11096](https://github.com/kubermatic/kubermatic/pull/11096)) Following endpoint has been updated:
    - GET `/api/v2/projects/{project_id}/clusters/{cluster_id}/bindings`
    - POST `/api/v2/projects/{project_id}/clusters/{cluster_id}/roles/{namespace}/{role_id}/bindings`
    - DELETE `/api/v2/projects/{project_id}/clusters/{cluster_id}/roles/{namespace}/{role_id}/bindings`
    - GET `/api/v2/projects/{project_id}/clusters/{cluster_id}/clusterbindings`
    - POST `/api/v2/projects/{project_id}/clusters/{cluster_id}/clusterroles/role_id/clusterbindings`
    - DELETE `/api/v2/projects/{project_id}/clusters/{cluster_id}/clusterroles/role_id/clusterbindings`
- KubeVirt: add new API endpoints to list instancetypes and preferences ([#11085](https://github.com/kubermatic/kubermatic/pull/11085))
- Add API endpoints for GKE that allow using project-scoped Presets as credentials ([#11156](https://github.com/kubermatic/kubermatic/pull/11156))

### Miscellaneous

- Update codebase to Kubernetes 1.26 ([#11600](https://github.com/kubermatic/kubermatic/pull/11600))
- Update Go to version 1.19.4 ([#11530](https://github.com/kubermatic/kubermatic/pull/11530))
- Update to controller-runtime 0.13.0 ([#10957](https://github.com/kubermatic/kubermatic/pull/10957))
- Enhanced AKS external cluster status information on errors or warning states ([#10913](https://github.com/kubermatic/kubermatic/pull/10913))
- The `quay.io/kubermatic/util:2.2.0` Docker image does not ship with `yq3` and `yq4` anymore, but only `yq` (version 4.x) ([#10665](https://github.com/kubermatic/kubermatic/pull/10665))
- User deletion will not be allowed if the user is the sole owner of project which has resources i.e, clusters or external clusters ([#11289](https://github.com/kubermatic/kubermatic/pull/11289))
- Change primary Git branch from `master` to `main` ([#11206](https://github.com/kubermatic/kubermatic/pull/11206))
- Move mutation to add default OSP annotations to `MachineDeployments` to a new operating-system-manager webhook ([#11325](https://github.com/kubermatic/kubermatic/pull/11325))
- Stop overriding upstream chart tolerations for logging/promtail by default, adding `node-role.kubernetes.io/control-plane` toleration ([#11592](https://github.com/kubermatic/kubermatic/pull/11592))
- Use `topology.kubernetes.io/zone` instead of `failure-domain.beta.kubernetes.io/zone` for KKP components ([#10808](https://github.com/kubermatic/kubermatic/pull/10808))

### Cleanup

- Remove `metrics-server` addon. The addon was deprecated in v2.9 and should not be part of any usercluster anymore ([#11184](https://github.com/kubermatic/kubermatic/pull/11184))
- Remove unused `Cluster.Status.CloudProviderRevision` field which carried no useful information anymore ([#11299](https://github.com/kubermatic/kubermatic/pull/11299))
