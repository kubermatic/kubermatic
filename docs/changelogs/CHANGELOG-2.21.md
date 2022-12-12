# Kubermatic 2.21

- [v2.21.0](#v2210)
- [v2.21.1](#v2211)
- [v2.21.2](#v2212)
- [v2.21.3](#v2213)
- [v2.21.4](#v2214)

## [v2.21.4](https://github.com/kubermatic/kubermatic/releases/tag/v2.21.4)

### Action Required

- ACTION REQUIRED: Use `registry.k8s.io` instead of `k8s.gcr.io` for Kubernetes upstream images. It might be necessary to update firewall rules or mirror registries accordingly ([#11079](https://github.com/kubermatic/kubermatic/pull/11079))

### API Changes

- Add the option to set autoscaler min and max replicas for a machine deployment through the KKP API. They are only relevant if the autoscaler addon is installed ([#11544](https://github.com/kubermatic/kubermatic/pull/11544))

### New Feature

- Defaulting vSphere tag category from seed, when it is not specified in user cluster ([#11460](https://github.com/kubermatic/kubermatic/pull/11460))

### Bugfixes

- Disable promtail initContainer that was overriding system `fs.inotify.max_user_instances` configuration ([#11382](https://github.com/kubermatic/kubermatic/pull/11382))
- Fix duplicate SourceRange entries for front-loadbalancer Service ([#11371](https://github.com/kubermatic/kubermatic/pull/11371))
- Fix the issue where AllowedRegistry ConstraintTemplate was not being reconiciled by Gatekeeper because it's `spec.crd` OpenAPI spec was missing a type ([#11327](https://github.com/kubermatic/kubermatic/pull/11327))
- Monitoring: fixes missing etcd metrics in Grafana etcd dashboards and master/seed Prometheus by renaming to: `etcd_mvcc_db_total_size_in_bytes`, `etcd_mvcc_delete_total`, `etcd_mvcc_put_total`, `etcd_mvcc_range_total`, `etcd_mvcc_txn_total` ([#11438](https://github.com/kubermatic/kubermatic/pull/11438))
- Prioritise public IP over private IP in front LoadBalancer service ([#11512](https://github.com/kubermatic/kubermatic/pull/11512))

### Updates

- Update KubeVirt CSI driver operator version to v0.1.3 ([#11399](https://github.com/kubermatic/kubermatic/pull/11399))
- Update to etcd 3.5.6 for Kubernetes 1.22+ to prevent potential data inconsistency issues during online defragmentation ([#11404](https://github.com/kubermatic/kubermatic/pull/11404))
- Update nginx-ingress to 1.5.1 ([#11416](https://github.com/kubermatic/kubermatic/pull/11416))
- Update Dex to 2.35.3 ([#11419](https://github.com/kubermatic/kubermatic/pull/11419))
- Update OpenStack Cinder CSI to v1.24.5, v1.23.4, and v1.22.2 ([#11455](https://github.com/kubermatic/kubermatic/pull/11455))
- Update Anexia CCM (cloud-controller-manager) to version 1.5.0 ([#11503](https://github.com/kubermatic/kubermatic/pull/11503))
- Update Go to version 1.18.9 ([#11532](https://github.com/kubermatic/kubermatic/pull/11532))
- Update machine-controller to v1.54.3 ([#11545](https://github.com/kubermatic/kubermatic/pull/11545))
- Add support for Kubernetes v1.22.17, v1.23.15 and v1.24.9 ([#11554](https://github.com/kubermatic/kubermatic/pull/11554))
- Add etcd database size alerts `EtcdDatabaseQuotaLowSpace`, `EtcdExcessiveDatabaseGrowth`, `EtcdDatabaseHighFragmentationRatio`([#11560](https://github.com/kubermatic/kubermatic/pull/11560))

### Dashboard

- Provide options for autoscaling nodes ([#5402](https://github.com/kubermatic/dashboard/pull/5402))

## [v2.21.3](https://github.com/kubermatic/kubermatic/releases/tag/v2.21.3)

This release includes updated Kubernetes versions that fix CVE-2022-3162 and CVE-2022-3294. For more information, see below. We strongly recommend upgrading to those Kubernetes patch releases as soon as possible.

### Bugfixes

- Fix kubermatic-webhook panic on providerName mismatch from CloudSpec ([#11247](https://github.com/kubermatic/kubermatic/pull/11247))
- Fix rendering error of the metallb addon causing missing L2Advertisement ([#11233](https://github.com/kubermatic/kubermatic/pull/11233))
- Remove digests from Docker images in addon manifests to fix issues with Docker registry mirrors / local registries. KKP 2.22  will restore the digests and properly support them ([#11239](https://github.com/kubermatic/kubermatic/pull/11239))

### New Feature

- Introduce a new field `disableIAMReconciling` in AWS cloud spec to disable IAM reconciliation ([#11280](https://github.com/kubermatic/kubermatic/pull/11280))

### Updates

- Update MetalLB version to v0.13.7 ([#11256](https://github.com/kubermatic/kubermatic/pull/11256))
- Add support for Kubernetes 1.24.8, 1.23.14, and 1.22.16 and automatically upgrade existing clusters ([#11341](https://github.com/kubermatic/kubermatic/pull/11341))
    * Those Kubernetes patch releases fix CVE-2022-3162 and CVE-2022-3294, both in kube-apiserver: [CVE-2022-3162: Unauthorized read of Custom Resources](https://groups.google.com/g/kubernetes-announce/c/oR2PUBiODNA/m/tShPgvpUDQAJ) and [CVE-2022-3294: Node address isn't always verified when proxying](https://groups.google.com/g/kubernetes-announce/c/eR0ghAXy2H8/m/sCuQQZlVDQAJ).

#### Metering (EE)

- Update metering to version 1.0.1 ([#11293](https://github.com/kubermatic/kubermatic/pull/11293))
    * Add average-used-cpu-millicores to Cluster and Namespace reports
    * Add average-available-cpu-millicores add average-cluster-machines field to Cluster reports
    * Fix a bug that causes wrong values if metric is not continuously present for the aggregation window 

### Upcoming Changes

- For the next series of KKP patch releases, image references will move from `k8s.gcr.io` to `registry.k8s.io`. This will be done to keep up with [latest upstream changes](https://github.com/kubernetes/enhancements/tree/master/keps/sig-release/3000-artifact-distribution). Please ensure that any mirrors you use are going to host `registry.k8s.io` and/or that firewall rules are going to allow access to `registry.k8s.io` to pull images before applying the next KKP patch releases. **This is not included in this patch release but just a notification of future changes.**

## [v2.21.2](https://github.com/kubermatic/kubermatic/releases/tag/v2.21.2)

### Bugfixes

- Fix wrong quota filtering when VirtualMachineInstancePreset.spec.cpu has no quantity but only other fields ([#11046](https://github.com/kubermatic/kubermatic/issues/11046))
- Fix API error in extended disk configuration for provider Anexia ([#11050](https://github.com/kubermatic/kubermatic/issues/11050))
- Fix setting exposeStrategy via KKP cluster API endpoint ([#11061](https://github.com/kubermatic/kubermatic/issues/11061))
- Fix `--config` flag not being validated in `mirror-images` command in the KKP installer ([#11146](https://github.com/kubermatic/kubermatic/issues/11146))
- `kubermatic-installer mirror-images` correctly picks up konnectivity and Kubernetes dashboard images ([#11148](https://github.com/kubermatic/kubermatic/issues/11148))
- Fix Seed-Proxy ServiceAccount token not being generated ([#11190](https://github.com/kubermatic/kubermatic/issues/11190))
- Fix `convert-kubeconfig` installer command not generating a SA token ([#11197](https://github.com/kubermatic/kubermatic/issues/11197))
- installer subcommand `mirror-images` correctly mirrors image `kubernetesui/metrics-scraper` now ([#11208](https://github.com/kubermatic/kubermatic/issues/11208))
- Prevent index out-of-bounds issue when querying GKE external cluster status ([#11213](https://github.com/kubermatic/kubermatic/issues/11213))

### Misc

- Added support for GroupProjectBindings in MLA Grafana ([#11076](https://github.com/kubermatic/kubermatic/issues/11076))
- Do not require addons flags in `kubermatic-installer mirror-images` and fall back to default addons image ([#11135](https://github.com/kubermatic/kubermatic/issues/11135))
- Set PriorityClassName of konnectivity-agent and openvpn-client to system-cluster-critical ([#11140](https://github.com/kubermatic/kubermatic/issues/11140))

### Updates

- Upgrade to cilium v1.12.2 and v1.11.9 ([#11013](https://github.com/kubermatic/kubermatic/issues/11013))
- Add support for Ubuntu 22.04 ([#11072](https://github.com/kubermatic/kubermatic/issues/11072))
- Update konnectivity to v0.0.33 ([#11080](https://github.com/kubermatic/kubermatic/issues/11080))
- Upgrade to machine-controller v1.54.2 ([#11090](https://github.com/kubermatic/kubermatic/issues/11090))
- Upgrade to operating-system-manager v1.1.1 ([#11090](https://github.com/kubermatic/kubermatic/issues/11090))


## [v2.21.1](https://github.com/kubermatic/kubermatic/releases/tag/v2.21.1)

### API Changes

- Extend disk configuration for Anexia provider ([#10916](https://github.com/kubermatic/kubermatic/pull/10916))

### New Feature

- Seed-proxy: increase memory limit from 32Mi to 64Mi ([#10984](https://github.com/kubermatic/kubermatic/pull/10984))

### Bugfixes

- A race condition bug in `etcd-launcher` that can trigger on user cluster initialisation and that prevents the last etcd node from joining the etcd cluster has been fixed ([#10932](https://github.com/kubermatic/kubermatic/pull/10932))
- Fix Openstack `api/v1/providers/openstack/tenants` API endpoint for some cases where "couldn't get projects: couldn't get tenants for region XX: couldn't get identity endpoint: No suitable endpoint could be found in the service catalog." was wrongly returned ([#10968](https://github.com/kubermatic/kubermatic/pull/10968))
- Fix for listing Operating System Profiles for Equinix Metal ([#4969](https://github.com/kubermatic/dashboard/pull/4969))
- Fix issue in KKP API where deleting all datacenters from a Seed and then trying to add a new one would cause a panic ([#10953](https://github.com/kubermatic/kubermatic/pull/10953))
- Fix kubermatic-webhook failing to start on external seed clusters ([#10958](https://github.com/kubermatic/kubermatic/pull/10958))
- Fix upgrades for external seeds that have clusters with no `enableOperatingSystemManager` flag yet, resulting in the seed-operator not being able to fully upgrade the seed cluster to 2.21 ([#10948](https://github.com/kubermatic/kubermatic/pull/10948))
- Prefer InternalIP when connecting to Kubelet for Hetzner dual-stack clusters ([#10937](https://github.com/kubermatic/kubermatic/pull/10937))
- Update OpenStack version for k8s 1.23 to fix services ports mapping issue ([#11022](https://github.com/kubermatic/kubermatic/pull/11022))

### Chore

- Add support for Kubernetes 1.22.15, 1.23.12 and 1.24.6; existing clusters using these Kubernetes releases will be automatically updated as any previous version is affected by CVEs ([#11027](https://github.com/kubermatic/kubermatic/pull/11027))

### Updates

- Update Cilium 1.12 to v1.12.1 ([#10952](https://github.com/kubermatic/kubermatic/pull/10952))


## [v2.21.0](https://github.com/kubermatic/kubermatic/releases/tag/v2.21.0)

Before upgrading, make sure to read the [general upgrade guidelines](https://docs.kubermatic.com/kubermatic/v2.21/tutorials-howtos/upgrading/). Consider tweaking `seedControllerManager.maximumParallelReconciles` to ensure usercluster reconciliations will not cause resource exhaustion on seed clusters.

### Supported Kubernetes Versions

- Add support for Kubernetes 1.23; Kubernetes 1.23 is currently not supported on ARM64 clusters running Canal and kube-proxy in the IPVS mode. KKP will allow the creation of new 1.23 clusters with the ARM64 nodes, Canal, and kube-proxy in the IPVS mode, but those clusters will not be usable. In this case, you can delete the cluster and create a 1.22 cluster, or switch to the AMD64 nodes. Upgrades of the existing clusters 1.22 clusters with ARM64 nodes, Canal, and kube-proxy in the IPVS mode is forbidden using the newly-added `nonAMD64WithCanalAndIPVS` incompatibility ([#8455](https://github.com/kubermatic/kubermatic/pull/8455))
- Add support for Kubernetes 1.24 ([#9736](https://github.com/kubermatic/kubermatic/pull/9736))
- Remove support for Kubernetes 1.20 ([#9384](https://github.com/kubermatic/kubermatic/pull/9384))
- Remove support for Kubernetes 1.21, auto-upgrade existing clusters to 1.22.11 ([#10147](https://github.com/kubermatic/kubermatic/pull/10147))

Supported Versions:

- v1.22.5
- v1.22.9
- v1.22.12
- v1.23.6
- v1.23.9
- v1.24.3

### Highlights

- Remove the `/api/v1/projects/{project_id}/clusters/{cluster_id}/dashboard/proxy` endpoint. Update the `/api/v2/projects/{project_id}/clusters/{cluster_id}/dashboard/proxy` to use the OIDC based authentication flow to access K8S Dashboard. Add the `api/v2/dashboard/login` endpoint to initiate the OIDC login flow ([#10072](https://github.com/kubermatic/kubermatic/pull/10072))
- Support RockyLinux as an operating system ([#9800](https://github.com/kubermatic/kubermatic/pull/9800), [#4624](https://github.com/kubermatic/dashboard/pull/4624), [#4515](https://github.com/kubermatic/dashboard/pull/4515))
- Support for VMware Cloud Director as a cloud provider ([#9933](https://github.com/kubermatic/kubermatic/pull/9933))
- The KKP Operator now updates CRDs on seed clusters. If the KKP Minio chart is not used and the legacy etcd backup configuration is also not used anymore, the KKP Installer does not need to be used for *updating* seed clusters anymore (however first setups of new seed clusters must still be done using the KKP installer) ([#9748](https://github.com/kubermatic/kubermatic/pull/9748))
- Update KubeVirt logo to mark technology preview ([#4810](https://github.com/kubermatic/dashboard/pull/4810))

#### Resource Quotas (EE)

- Add controllers for calculating resource quota global and local seed usage ([#10160](https://github.com/kubermatic/kubermatic/pull/10160))
- Clusters will now have resource usage in in their `status` ([#10070](https://github.com/kubermatic/kubermatic/pull/10070))
- Add a Machine validating webhook which checks the Machine resource requests (CPU, Memory, Storage) against its project's resource quota (if set) ([#9650](https://github.com/kubermatic/kubermatic/pull/9650))
- Introduce new API endpoints for CRUD operations on KKP Resource Quotas ([#10079](https://github.com/kubermatic/kubermatic/pull/10079))
- Add admin setting page for adding resource quota for projects ([#4641](https://github.com/kubermatic/dashboard/pull/4641))
- Add quota widget for projects in the dashboard ([#4731](https://github.com/kubermatic/dashboard/pull/4731))
- Add support for managing project resource quotas in the dashboard ([#4680](https://github.com/kubermatic/dashboard/pull/4680), [#4690](https://github.com/kubermatic/dashboard/pull/4690))

#### Applications
​
- Add application installation and management from Git/Helm sources ([#9977](https://github.com/kubermatic/kubermatic/issues/9977), [#10363](https://github.com/kubermatic/kubermatic/issues/10363))
- Add API endpoints for ApplicationInstallations/ApplicationDefinitions ([#10286](https://github.com/kubermatic/kubermatic/issues/10286), [#10341](https://github.com/kubermatic/kubermatic/issues/10341))
- Add authentication with RegistryConfigFile in HelmCredentials ([#10564](https://github.com/kubermatic/kubermatic/issues/10564), [#10570](https://github.com/kubermatic/kubermatic/issues/10570))

#### Operating System Manager

- Operating System Manager is enabled by default and it's responsible for creating and managing the required configurations for worker nodes ([#10415](https://github.com/kubermatic/kubermatic/pull/10415))
- Containerd container runtime mirror registries support ([#10134](https://github.com/kubermatic/kubermatic/pull/10134))
- OSM Deployment Docker image repository and tag can be overwritten using the `KubermaticConfiguration` ([#10123](https://github.com/kubermatic/kubermatic/pull/10123))

#### Group Project Bindings (EE)

- Add `GroupProjectBinding` CRD ([#10158](https://github.com/kubermatic/kubermatic/pull/10158))
- Add REST API for interacting with GroupProjectBinding resources ([#10303](https://github.com/kubermatic/kubermatic/pull/10303), [#4712](https://github.com/kubermatic/dashboard/pull/4712))
- Add support for GroupProjectBindings in Alertmanager authorization server ([#10574](https://github.com/kubermatic/kubermatic/pull/10574))

#### Encryption At Rest

- Add experimental support for encryption-at-rest with secretbox for static key encryption ([#9654](https://github.com/kubermatic/kubermatic/pull/9654))

#### External Clusters

- Add support to create Azure Kubernetes Cluster (AKS) ([#8884](https://github.com/kubermatic/kubermatic/pull/8884))
- Add support to create AWS Elastic Kubernetes Service Cluster (EKS) ([#8883](https://github.com/kubermatic/kubermatic/pull/8883))
- Allows to delete External Cluster from the Cloud Provider ([#10330](https://github.com/kubermatic/kubermatic/pull/10330))
- Allow to create an EKS Nodepool ([#8976](https://github.com/kubermatic/kubermatic/pull/8976))
- Display the GKE cluster details ([#9144](https://github.com/kubermatic/kubermatic/pull/9144))
- `GET /api/v2/providers/gke/versions` to list GKE versions list ([#10511](https://github.com/kubermatic/kubermatic/pull/10511))
- GET endpoint for AMI types, Capacity types, Subnets, VPCs and Instance Types ([#9002](https://github.com/kubermatic/kubermatic/pull/9002))
- GET VMSizes, NodePool Modes, Kubernetes Version ([#8925](https://github.com/kubermatic/kubermatic/pull/8925))
- KubermaticConfiguration contains version configuration for providers like EKS ([#10537](https://github.com/kubermatic/kubermatic/pull/10537))
- Add Kubernetes versions drop down list when creating new external Cluster ([#4747](https://github.com/kubermatic/dashboard/pull/4747))
- Add support for updating/deleting MachineDeployments of external Cluster (AKS/EKS/GKE) ([#4657](https://github.com/kubermatic/dashboard/pull/4657), [#4660](https://github.com/kubermatic/dashboard/pull/4660))
- The `CloudSpec` of `ExternalClusters` must be set and use the new `bringyourown` provider when previously no cloud provider was configured. This is identical to how regular `Clusters` behave. Existing `ExternalClusters` are automatically migrated when using the kubermatic-installer, manual setups need to manually set `spec.cloudSpec.providerName = "bringyourown"` and `spec.cloudSpec.bringyourown: {}` after the CRDs were updated ([#10762](https://github.com/kubermatic/kubermatic/pull/10762))

#### Dual-Stack Support

- Add dual-stack support for AWS security groups ([#9133](https://github.com/kubermatic/kubermatic/pull/9133))
- Add dual-stack support for Azure provider resources ([#9443](https://github.com/kubermatic/kubermatic/pull/9443))
- Add dual-stack support for Canal CNI ([#9730](https://github.com/kubermatic/kubermatic/pull/9730))
- Add dual-stack support for Equnix Metal, DigitalOcean and bringyourown cloud providers ([#10344](https://github.com/kubermatic/kubermatic/pull/10344))
- Add dual-stack support for GCP firewall rules ([#9400](https://github.com/kubermatic/kubermatic/pull/9400))
- Add dual-stack support for Hetzner ([#10037](https://github.com/kubermatic/kubermatic/pull/10037))
- Add dual-stack support for OpenStack provider ([#9532](https://github.com/kubermatic/kubermatic/pull/9532))
- Add dual-stack support for vSphere user clusters ([#10424](https://github.com/kubermatic/kubermatic/pull/10424))
- Add support for dual-stack pods & services CIDR ([#9103](https://github.com/kubermatic/kubermatic/pull/9103))
- Allow rendering dual-stack IPAddressPool in metallb addon ([#10763](https://github.com/kubermatic/kubermatic/pull/10763))

#### Declarative KKP Preview

This release offers limited support for managing KKP resources directly using the Kubernetes API (e.g. by using `kubectl`). As KKP is working on polishing the previously private CRDs, administrators should expect rough edges (for example fields that are not yet defaulted or missing validation).

Note that in the current state, declarative working skips KKP authentication and is therefore primarily suited for smaller setups where permissions are handled by an external review workflow (e.g. pull requests on GitHub). Using this feature requires access to the master and seed clusters and is not recommended for end users.

### Breaking Changes

- Operating System Manager (OSM) is enabled by default and it's responsible for creating and managing the required configurations for worker nodes; for existing clusters, admins need to set `enableOperatingSystemManager` to true in the cluster spec to enable OSM. Existing `MachineDeployments` will not be rotated automatically. To use OSM for existing `MachineDeployments`, the user needs to update the `MachineDeployments` manually. An example for a somewhat benign change that could trigger rotation is to update the `.spec.templates.metadata.annotations` field of a MachineDeployment. This would result in the annotation being added to the machines and the machines would be rotated ([#10415](https://github.com/kubermatic/kubermatic/pull/10415))
- Secret name for S3 credentials updated to `kubermatic-s3-credentials`. If the secret `s3-credentials` was manually created instead of using the `minio` Helm chart, new Secret `kubermatic-s3-credentials` must be created ([#9230](https://github.com/kubermatic/kubermatic/pull/9230), [#4700](https://github.com/kubermatic/dashboard/pull/4700))
- Restore correct labels on nodeport-proxy-envoy Deployment. Deleting the existing Deployment for each cluster with the `LoadBalancer` expose strategy if upgrading from affected version is necessary ([#9060](https://github.com/kubermatic/kubermatic/pull/9060))
- Update blackbox-exporter to 0.21.0; HTTP probe: `no_follow_redirects` has been renamed to `follow_redirects`; disabled support for TLS 1.0/1.1 by default and rejects certificates signed with SHA-1. Please refer to the [documentation](https://prometheus.io/docs/prometheus/latest/configuration/configuration/#tls_config) for more information ([#9638](https://github.com/kubermatic/kubermatic/pull/9638), [#10084](https://github.com/kubermatic/kubermatic/pull/10084))
- Update cert-manager to 1.9.1; any API version earlier than v1 is not available anymore ([#9645](https://github.com/kubermatic/kubermatic/pull/9645))
- Update Helm-Exporter to 1.2.2; Helm 2 is not supported anymore, on clusters with many Helm releases, performance tweaks might be necessary ([#9642](https://github.com/kubermatic/kubermatic/pull/9642))
- Update Promtail to v2.5.0; the `config.client` configuration has been replaced by `config.clients` instead. If you overwrote the Promtail client (=Loki), please adjust your `values.yaml` accordingly ([#10082](https://github.com/kubermatic/kubermatic/pull/10082))
- Update Velero to 1.9.0; removed `velero.defaultBackupStorageLocation` from the Helm values, set `spec.default=true` on your `BackupStorageLocation` instead (see `values.yaml` for an example) ([#9643](https://github.com/kubermatic/kubermatic/pull/9643))
- Fix inconsistent casing for `floatingIPPool` in cloud spec. This affects the API endpoints used to list and get clusters; for OpenStack, the field `floatingIpPool` was replaced with `floatingIPPool`. Endpoints for creation and update are not affected ([#10423](https://github.com/kubermatic/kubermatic/pull/10423))
- Update OPA Gatekeeper to 3.6.0: OPA ConstraintTemplates are upgraded from v1beta1 to v1. When creating new Kubermatic ConstraintTemplates the `spec.crd.spec.validation.openAPIV3Schema` needs to be structurally correct. The old ConstraintTemplates will have a `legacySchema` flag in the `spec.crd.spec.validation` so they won't need to be migrated yet, although we suggest editing them and fixing the schema to be structurally correct. More info about change from v1beta1 to v1 in [gatekeeper docsumentation](https://open-policy-agent.github.io/gatekeeper/website/docs/constrainttemplates#v1-constraint-template) ([#8973](https://github.com/kubermatic/kubermatic/pull/8973))
- Update Metering to version 1.0. This changes the report csv format and the method of data collection. All previous generated reports will be still accessible via the dashboard. The new metering has a slightly different format and uses a different data source. This means that data collection will start from the beginning at the time of upgrading. Going back in time is not possible due to the change of the data source ([#10721](https://github.com/kubermatic/kubermatic/pull/10721))

### Cloud Providers

- Support Amazon Linux 2 as an operating system ([#10683](https://github.com/kubermatic/kubermatic/pull/10683), [#4794](https://github.com/kubermatic/dashboard/pull/4794))
- Support RockyLinux as an operating system ([#9800](https://github.com/kubermatic/kubermatic/pull/9800), [#4624](https://github.com/kubermatic/dashboard/pull/4624), [#4515](https://github.com/kubermatic/dashboard/pull/4515))
- Cloud provider credentials (`.spec.cloud` in a Cluster object) are transferred into a Secret by a controller, something previously only done during cluster creation when using the KKP dashboard. Now this procedure affects every Cluster object. The cluster credentials are now also mirrored into the usercluster namespace ([#10505](https://github.com/kubermatic/kubermatic/pull/10505))
- Cloud provider credentials are not put into environment variables for Deployments (like the kube-apiserver) anymore, but instead Deployments reference Secrets ([#10506](https://github.com/kubermatic/kubermatic/pull/10506))
- Cloud provider spec changes fail validation for fields that are not supported by in-place updates (mostly cloud resources that can be auto-generated by KKP) ([#9868](https://github.com/kubermatic/kubermatic/pull/9868))
- The flag `--kubelet-certificate-authority` (introduced in KKP 2.19) is not set for "kubeadm" / "bringyourown" user clusters anymore ([#9674](https://github.com/kubermatic/kubermatic/pull/9674))
- Validate Alibaba, Anexia, and vSphere provider credentials in the cluster webhook ([#9287](https://github.com/kubermatic/kubermatic/pull/9287))
- Add allowed IP range override support to the GCP, Azure, AWS, and OpenStack providers ([#4314](https://github.com/kubermatic/dashboard/pull/4314))

#### Anexia

- Anexia now supports LoadBalancer Services ([#10507](https://github.com/kubermatic/kubermatic/pull/10507))

#### AWS

- Fix cloud provider cleanup sometimes getting stuck when cleaning up tags ([#8879](https://github.com/kubermatic/kubermatic/pull/8879))
- Flatcar on AWS will default to ignition as the provisioning utility ([#10604](https://github.com/kubermatic/kubermatic/pull/10604))

#### Azure

- Add CSI drivers for Azure Disk and Azure File ([#10049](https://github.com/kubermatic/kubermatic/pull/10049))
- Allow migrating existing Azure clusters to the external CCM ([#9963](https://github.com/kubermatic/kubermatic/pull/9963))
- Attach previously unattached Azure route table to generated subnet ([#9963](https://github.com/kubermatic/kubermatic/pull/9963))
- Fix potential race in cleanup of Azure resources ([#9553](https://github.com/kubermatic/kubermatic/pull/9553))
- ICMP rules migration only runs on Azure NSGs created by KKP ([#8843](https://github.com/kubermatic/kubermatic/pull/8843))
- New Azure clusters use external CCM by default ([#10049](https://github.com/kubermatic/kubermatic/pull/10049))
- Updated OS default disk size to 64GB when RHEL OS is selected ([#4314](https://github.com/kubermatic/dashboard/pull/4314))
- When using the "standard" load balancer SKU for Azure clusters, MachineDeployments use the same SKU for public IP addresses ([#10678](https://github.com/kubermatic/kubermatic/pull/10678))

#### DigitalOcean

- Add support for the DigitalOcean CSI driver ([#10375](https://github.com/kubermatic/kubermatic/pull/10375))
- Option to configure IPv6 has been removed from node settings since it can now be configured using dual-stack network configuration in cluster creation wizard ([#4613](https://github.com/kubermatic/dashboard/pull/4613))

#### Equinix Metal (formerly Packet)

- Add metro support ([#10328](https://github.com/kubermatic/kubermatic/pull/10328))

#### GCP

- GCP cloud resources are periodically reconciled ([#8810](https://github.com/kubermatic/kubermatic/pull/8810))

#### Hetzner

- If a network is set in the Hetzner cluster spec, it is now correctly applied to generated machines ([#8872](https://github.com/kubermatic/kubermatic/pull/8872))

#### KubeVirt

- Add support for storage classes initialization on KubeVirt user clusters that users can use hot-pluggable disks ([#10006](https://github.com/kubermatic/kubermatic/pull/10006), [#9898](https://github.com/kubermatic/kubermatic/pull/9898))
- Initialisation of VirtualMachineInstancePresets in a dedicated namespace in the infra KubeVirt Cluster ([#9296](https://github.com/kubermatic/kubermatic/pull/9296))
- Reconcile the VirtualMachineInstancePresets from `default` namespace into the dedicated namespace `cluster-xxyy` in the update cluster flow ([#9700](https://github.com/kubermatic/kubermatic/pull/9700))
- Add support for KubeVirt pre-allocated data volumes ([#4722](https://github.com/kubermatic/dashboard/pull/4722))
- Configure pod affinity/anti-affinity and node affinity preset settings for KubeVirt provider ([#4720](https://github.com/kubermatic/dashboard/pull/4720))

#### Nutanix

- Add Nutanix CSI driver ([#8865](https://github.com/kubermatic/kubermatic/pull/8865), [#4251](https://github.com/kubermatic/dashboard/pull/4251))
- Correctly handle the 'default' Nutanix project in API calls ([#9332](https://github.com/kubermatic/kubermatic/pull/9332))

#### Openstack

- Add EnableIngressHostname and IngressHostnameSuffix options (enables workaround in Openstack CCM for PROXY protocol client IP preservation) ([#10751](https://github.com/kubermatic/kubermatic/pull/10751))
- Add IPv6 subnet ID and IPv6 subnet pool for OpenStack cluster provider ([#4682](https://github.com/kubermatic/dashboard/pull/4682))
- Add external snapshotter for Cinder CSI; add default VolumeSnapshotClass for supported provider ([#9893](https://github.com/kubermatic/kubermatic/pull/9893))
- Add support for OpenStack CCM v1.24.0 (for Kubernetes 1.24 clusters)Add support for OpenStack Cinder CSI driver v1.24.0 (for Kubernetes 1.24 clusters)Update CSI components in OpenStack Cinder CSI driver ([#9935](https://github.com/kubermatic/kubermatic/pull/9935))
- Allow volume expansion on OpenStack Cinder CSI StorageClass ([#9433](https://github.com/kubermatic/kubermatic/pull/9433))
- Fix missing snapshot CRD's for Cinder CSI ([#9042](https://github.com/kubermatic/kubermatic/pull/9042))
- Support for `network:ha_router_replicated_interface` ports when discovering existing subnet router in Openstack ([#9164](https://github.com/kubermatic/kubermatic/pull/9164))
- Support for application credentials in OpenStack preset ([#4192](https://github.com/kubermatic/dashboard/pull/4192))

#### VMware Cloud Director

- Support for VMware Cloud Director as a cloud provider ([#9933](https://github.com/kubermatic/kubermatic/pull/9933), [#4644](https://github.com/kubermatic/dashboard/pull/4644))
- Add CSI driver support ([#10080](https://github.com/kubermatic/kubermatic/pull/10080))

#### vSphere

- Add `vsphereCSIClusterID` feature flag for the cluster object. This feature flag changes the cluster-id in the vSphere CSI config to the cluster name instead of the vSphere Compute Cluster name provided via Datacenter config. Migrating the cluster-id requires [manual steps](https://docs.kubermatic.com/kubermatic/v2.20/cheat_sheets/vsphere_cluster_id/) ([#9202](https://github.com/kubermatic/kubermatic/pull/9202))
- Add support for vSphere tags ([#9568](https://github.com/kubermatic/kubermatic/pull/9568))
- Add vSphere Snapshotter ([#9113](https://github.com/kubermatic/kubermatic/pull/9113))
- Bring back vSphere cluster field and make it required ([#8993](https://github.com/kubermatic/kubermatic/pull/8993))
- Enable the `vsphereCSIClusterID` feature flag when running the CCM/CSI migration ([#9557](https://github.com/kubermatic/kubermatic/pull/9557))
- Extend vSphere provider for default tag category ([#9327](https://github.com/kubermatic/kubermatic/pull/9327))
- Support latest vSphere Cloud Controller Manager and CSI driver for Kubernetes 1.23 ([#9750](https://github.com/kubermatic/kubermatic/pull/9750))

### CRD Changes

- `Cluster`
  - Add `nodeCidrMaskSizeIPv4` and `nodeCidrMaskSizeIPv6` to the networkConfig of Clusters ([#9344](https://github.com/kubermatic/kubermatic/pull/9344))
  - Add `nodePortsAllowedIPRanges` option to specify multiple IP ranges from which access to NodePort services is allowed in AWS, Azure, GCP and OpenStack ([#9571](https://github.com/kubermatic/kubermatic/pull/9571))
  - Add `spec.auditLogging.sidecar` to `Cluster` and `ClusterTemplate` resources to allow configuring fluent-bit outputs and resource overrides; update   fluent-bit audit logging sidecar to 1.9.5 ([#10140](https://github.com/kubermatic/kubermatic/pull/10140))
  - Add new `ClusterVersionsStatus` to the `ClusterStatus` which represents the currently active control plane versions. `cluster.spec.version` should now always be treated as the intended, eventual version, not the current version ([#9337](https://github.com/kubermatic/kubermatic/pull/9337))
  - Add optional `ipFamily` option to the `clusterNetwork` ([#9652](https://github.com/kubermatic/kubermatic/pull/9652))
  - Cluster object is kept around until the cluster namespace has been entirely removed from etcd (using a new finalizer) ([#10359](https://github.com/kubermatic/kubermatic/pull/10359))
  - Clusters now have a phase (creating, updating, running, terminating) to allow getting a quick overview over the health on a seed cluster ([#9414](https://github.com/kubermatic/kubermatic/pull/9414))
  - Fix inconsistent casing for `floatingIPPool` in cloud spec. This affects the API endpoints used to list and get clusters; for OpenStack, the field `floatingIpPool` was replaced with `floatingIPPool`. Endpoints for creation and update are not affected ([#10423](https://github.com/kubermatic/kubermatic/pull/10423))
  - Support for disabling kubernetes-dashboard ([#9511](https://github.com/kubermatic/kubermatic/pull/9511))
  - The `ClusterAddress` for user clusters was moved to the `ClusterStatus`; the old `address` field remains only for the migration and should not be relied upon anymore ([#9668](https://github.com/kubermatic/kubermatic/pull/9668))
  - `spec.pause` does not need to be set for Clusters anymore (defaults to `false`) ([#10473](https://github.com/kubermatic/kubermatic/pull/10473))
  - It's now possible to reference a secret with container registry credentials on Cluster resources by setting `spec.imagePullSecret`. These credentials are implicitly available on every node of the cluster ([#10031](https://github.com/kubermatic/kubermatic/pull/10031))
- `Seed`
  - Add support for configuration annotations and loadBalancerSourceRanges for front-loadbalancer service of node port proxy; for Seed CR, `spec.NodeportProxy.Annotations` is deprecated and `spec.NodeportProxy.Envoy.LoadBalancerService.Annotations` should be used instead ([#9476](https://github.com/kubermatic/kubermatic/pull/9476))
  - The etcd-backup-related containers are now loaded dynamically from the KubermaticConfiguration, the relevant CLI flags like `-backup-container=<file>` have been removed.The deprecated configuration options `KubermaticConfiguration.spec.seedController.backupRestore` and `Seed.spec.backupRestore` have been removed. Please migrate to `Seed.spec.etcdBackupRestore` ([#9003](https://github.com/kubermatic/kubermatic/pull/9003))
  - Seed resources now make use of the `status` subresource to keep track of Seed version, health and other conditions ([#9706](https://github.com/kubermatic/kubermatic/pull/9706))
- `KubermaticConfiguration`
  - Add `Status` to `KubermaticConfiguration` ([#10029](https://github.com/kubermatic/kubermatic/pull/10029))
  - It is now possible to disable all user accessible addons in the operator by setting `spec.api.accessibleAddons=[]` in the `KubermaticConfiguration` ([#9198](https://github.com/kubermatic/kubermatic/pull/9198))
- `User`
  - `spec.project` was added to signal the Service Account <-> Project relationship ([#9441](https://github.com/kubermatic/kubermatic/pull/9441))
  - `spec.isAdmin` now defaults to `false` ([#9538](https://github.com/kubermatic/kubermatic/pull/9538))
  - `spec.id` is marked as deprecated/optional, as this field is not used anymore ([#9538](https://github.com/kubermatic/kubermatic/pull/9538))
- `UserSSHKey`
  - `spec.owner` is marked as deprecated/optional, as this field is not used anymore ([#9538](https://github.com/kubermatic/kubermatic/pull/9538))
  - `spec.fingerprint` is marked as optional because the KKP webhook automatically (re)calculates the fingerprint ([#9538](https://github.com/kubermatic/kubermatic/pull/9538))
  - `spec.project` was added, making it easier to manage SSH keys declaratively. Existing UserSSHKey objects must be migrated, the kubermatic-installer takes care of that during the upgrade ([#9421](https://github.com/kubermatic/kubermatic/pull/9421))
- `ExternalCluster`
  - The `CloudSpec` must be set and use the new `bringyourown` provider when previously no cloud provider was configured. This is identical to how regular `Clusters` behave. Existing `ExternalClusters` are automatically migrated when using the kubermatic-installer, manual setups need to manually set `spec.cloudSpec.providerName = "bringyourown"` and `spec.cloudSpec.bringyourown: {}` after the CRDs were updated ([#10762](https://github.com/kubermatic/kubermatic/pull/10762))
- Remove deprecated fields from Cluster CRD ([#8961](https://github.com/kubermatic/kubermatic/pull/8961))
  - `Cluster.spec.masterVersion`
  - `Cluster.status.kubermaticVersion`
  - `Cluster.status.rootCA`
  - `Cluster.status.apiserverCert`
  - `Cluster.status.kubeletCert`
  - `Cluster.status.apiserverSSHKey`
  - `Cluster.status.serviceAccountKey`
  - `Cluster.spec.cloud.aws.roleName`
- Add `GroupProjectBinding` CRD ([#10158](https://github.com/kubermatic/kubermatic/pull/10158))
- Add `isDefault` flag to the RuleGroup API ([#8936](https://github.com/kubermatic/kubermatic/pull/8936))

### API Changes

- Add endpoint to list operating system profiles ([#10532](https://github.com/kubermatic/kubermatic/pull/10532))
- Add endpoints for querying Nutanix category data ([#9466](https://github.com/kubermatic/kubermatic/pull/9466))
- Add endpoints to list storage profiles for VMware Cloud Director ([#10217](https://github.com/kubermatic/kubermatic/pull/10217))
- Add endpoints to list networks, catalogs, and templates for VMware Cloud Director ([#9982](https://github.com/kubermatic/kubermatic/pull/9982))
- Add endpoints to list networks, catalogs, storage profiles, and templates for VMware Cloud Director based on the project and cluster ID ([#10268](https://github.com/kubermatic/kubermatic/pull/10268))
- Add endpoints to list EKS Subnets, VPCs, Regions and SecurityGroups ([#8896](https://github.com/kubermatic/kubermatic/pull/8896))
- Add preset stats endpoint: `GET /api/v2/presets/{preset_name}/stats` ([#9596](https://github.com/kubermatic/kubermatic/pull/9596))
- Allow listing invalidated ServiceAccount tokens ([#9371](https://github.com/kubermatic/kubermatic/pull/9371))
- Do not reference Nutanix cluster in endpoint path for subnets ([#8906](https://github.com/kubermatic/kubermatic/pull/8906))
- etcd backup API now requires destination to be set for etcdbackupconfig, etcdrestore and backupcredentials endpoints ([#9139](https://github.com/kubermatic/kubermatic/pull/9139))
- New endpoint for seed creation ([#9962](https://github.com/kubermatic/kubermatic/pull/9962))
- Remove the `/api/v1/projects/{project_id}/clusters/{cluster_id}/dashboard/proxy` endpoint. Update the `/api/v2/projects/{project_id}/clusters/{cluster_id}/dashboard/proxy` to use the OIDC based authentication flow to access K8S Dashboard. Add the `api/v2/dashboard/login` endpoint to initiate the OIDC login flow ([#10072](https://github.com/kubermatic/kubermatic/pull/10072))
- Update `PATCH` seed endpoint to support kubeconfig ([#9985](https://github.com/kubermatic/kubermatic/pull/9985))
- Update list feature gates endpoint to include OIDCKubeCfgEndpoint ([#9034](https://github.com/kubermatic/kubermatic/pull/9034))

### Metrics

- `kubermatic_cluster_info` Prometheus metric was updated: `type` label was removed, `master_version` renamed to `spec_version` and `current_version` and `phase` labels were added ([#9794](https://github.com/kubermatic/kubermatic/pull/9794))
- Add `kubermatic_external_cluster_info` metric with `name`, `display_name`, `provider` and `phase` labels (note that external cluster metrics are provided by the master-controller-manager) ([#9794](https://github.com/kubermatic/kubermatic/pull/9794))
- Add a `kubermatic_cluster_labels` metric that contains all Kubernetes labels on Cluster objects (similar to kube-state-metrics),* adds a `kubermatic_project_labels` metric that contains all Kubernetes labels on Projects objects ([#10605](https://github.com/kubermatic/kubermatic/pull/10605))
- Add a `kubermatic_project_info` metric with `name`, `display_name`, `owner` and `phase` labels.
- Add a `project` label to the `kubermatic_cluster_info` metric, containing the project name for which the cluster belongs to ([#10605](https://github.com/kubermatic/kubermatic/pull/10605))
- The seed-controller-manager is now providing Prometheus metrics regarding etcd backups (only for the new etcd backup/restore controllers) ([#9765](https://github.com/kubermatic/kubermatic/pull/9765))

### KKP Dashboard

- Add additional header to prevent being shown in an iframe ([#4796](https://github.com/kubermatic/dashboard/pull/4796))
- Add additional stats for each Provider Preset ([#4412](https://github.com/kubermatic/dashboard/pull/4412))
- Add custom metering report schedules ([#4403](https://github.com/kubermatic/dashboard/pull/4403))
- Add extra detail in external cluster page ([#4681](https://github.com/kubermatic/dashboard/pull/4681))
- Add new option in user settings to set the default project landing page to navigate to when open a project ([#4643](https://github.com/kubermatic/dashboard/pull/4643))
- Add quota widget to project overview page ([#4867](https://github.com/kubermatic/dashboard/pull/4867))
- Add retention parameter to metering schedule configuration ([#4478](https://github.com/kubermatic/dashboard/pull/4478))
- Add support for cluster applications ([#4694](https://github.com/kubermatic/dashboard/pull/4694))
- Add support for creating external clusters on AKS/EKS/GKE ([#4642](https://github.com/kubermatic/dashboard/pull/4642), [#4589](https://github.com/kubermatic/dashboard/pull/4589), [#4672](https://github.com/kubermatic/dashboard/pull/4672))
- Add validations for backup destination names ([#4661](https://github.com/kubermatic/dashboard/pull/4661))
- Add warning icon with a message for invalid service account tokens ([#4337](https://github.com/kubermatic/dashboard/pull/4337))
- Addition of a new field in the "additional cluster information > MISC" section related to the external CCM/CSI setting for the current cluster ([#4255](https://github.com/kubermatic/dashboard/pull/4255))
- Allow arbitrary human readable cluster names ([#4611](https://github.com/kubermatic/dashboard/pull/4611))
- Allow the user to specify the operating system profile when creating machine deployment ([#4602](https://github.com/kubermatic/dashboard/pull/4602))
- Configure pod affinity/anti-affinity and node affinity preset settings for KubeVirt provider ([#4720](https://github.com/kubermatic/dashboard/pull/4720))
- Disable OIDC Kubeconfig setting if feature gates is disabled ([#4734](https://github.com/kubermatic/dashboard/pull/4734))
- Disallow IPVS proxy mode when `Cilium` CNI is selected ([#4705](https://github.com/kubermatic/dashboard/pull/4705))
- Display configured secondary disks in edit machine deployment dialog ([#4726](https://github.com/kubermatic/dashboard/pull/4726))
- Display multiple external IPs in node details of machine deployment ([#4671](https://github.com/kubermatic/dashboard/pull/4671))
- Display share kubeconfig button even if OIDC Kubeconfig setting is enabled ([#4765](https://github.com/kubermatic/dashboard/pull/4765))
- Enable metering report removal ([#4438](https://github.com/kubermatic/dashboard/pull/4438))
- Fix a bug where the edit cluster view was not loading the correct configuration for the event rate limit plugin ([#4802](https://github.com/kubermatic/dashboard/pull/4802))
- Fix alignment of CPU/Memory usage on cluster details ([#4837](https://github.com/kubermatic/dashboard/pull/4837))
- Fix empty default selection in fallback case of Kubelet version selector ([#4619](https://github.com/kubermatic/dashboard/pull/4619))
- Fix settings defaulting after first settings update ([#4242](https://github.com/kubermatic/dashboard/pull/4242))
- Fix sorting metering reports by modification date ([#4359](https://github.com/kubermatic/dashboard/pull/4359))
- Fix: disallow to disable Konnectivity for CNI "Cilium" for proxy mode "ebpf" ([#4538](https://github.com/kubermatic/dashboard/pull/4538))
- Fix: on the machine deployment details page, show the correct nodes instead of all nodes ([#4577](https://github.com/kubermatic/dashboard/pull/4577))
- Hide MLA section if it is not enabled on seed level ([#4488](https://github.com/kubermatic/dashboard/pull/4488))
- MachineDeployment health is only considered "Running" if replicas are all updated ([#4504](https://github.com/kubermatic/dashboard/pull/4504))
- Show placeholder when no nodes exists inside external machine deployment details page ([#4869](https://github.com/kubermatic/dashboard/pull/4869))
- Support for disabling kubernetes-dashboard for user clusters ([#4729](https://github.com/kubermatic/dashboard/pull/4729))
- Support for dual stack cluster network ([#4604](https://github.com/kubermatic/dashboard/pull/4604))
- Update KubeVirt machine deployment settings ([#4178](https://github.com/kubermatic/dashboard/pull/4178))
- Update KubeVirt logo to mark technology preview ([#4810](https://github.com/kubermatic/dashboard/pull/4810))
- ApplicationDefinition can define default values to show in the UI when creation an applicationInstallation ([#10794](https://github.com/kubermatic/kubermatic/pull/10794))

### Bugfixes

- etcd-launcher is now capable of automatically rejoining the etcd ring when a member is removed during the peer TLS migration ([#9322](https://github.com/kubermatic/kubermatic/pull/9322))
- Fix addon variables not being persisted ([#10010](https://github.com/kubermatic/kubermatic/pull/10010))
- Fix an issue where Helm invocations by the kubermatic-installer ignored most environment variables ([#9876](https://github.com/kubermatic/kubermatic/pull/9876))
- Fix an issue with vsphere csi driver using improved-csi-idempotency that's currently not supported by KKP ([#10771](https://github.com/kubermatic/kubermatic/pull/10771))
- Fix applying resource requirements when using incomplete overrides (e.g. specifying only limits, but no requests for a container) ([#9045](https://github.com/kubermatic/kubermatic/pull/9045))
- Fix automatic Canal version upgrade for clusters using Kubernetes 1.23+ ([#10296](https://github.com/kubermatic/kubermatic/pull/10296))
- Fix deprecated nodePortProxy annotations (in `spec.nodePortProxy.annotations` in a Seed object) being ignored ([#10008](https://github.com/kubermatic/kubermatic/pull/10008))
- Fix etcdbackup controller constantly updating the EtcdBackup status ([#10650](https://github.com/kubermatic/kubermatic/pull/10650))
- Fix finalizers on clusters sometimes getting overwritten by the cloud controller or cluster-credentials controller ([#10536](https://github.com/kubermatic/kubermatic/pull/10536))
- Fix handling custom annotations for the front-loadbalancer Service ([#10436](https://github.com/kubermatic/kubermatic/pull/10436))
- Fix handling insecure HTTP endpoints for etcd backups ([#10189](https://github.com/kubermatic/kubermatic/pull/10189))
- Fix Konnectivity authentication issue in some scenarios by fixing cluster-external-addr-allow apiserver network policy ([#9187](https://github.com/kubermatic/kubermatic/pull/9187))
- Fix Mutating webhook for None CNI ([#9733](https://github.com/kubermatic/kubermatic/pull/9733))
- Fix `OpenVPNServerDown` alerting rule to work as expected and not fire if Konnectivity is enabled ([#9216](https://github.com/kubermatic/kubermatic/pull/9216))
- Fix Preset API Body for preset creation and update API calls ([#7856](https://github.com/kubermatic/kubermatic/pull/7856))
- Fix probes, resources and allow overriding resource requests/limits for Konnectivity proxy via components override in the cluster resource ([#9911](https://github.com/kubermatic/kubermatic/pull/9911))
- Fix reconcile loop in AllowedRegistry controller ([#10644](https://github.com/kubermatic/kubermatic/pull/10644))
- Fix Seeds being deleted on the master cluster not being cleaned up in the seed clusters themselves ([#9838](https://github.com/kubermatic/kubermatic/pull/9838))
- Fix telemetry CronJob not producing data ([#9740](https://github.com/kubermatic/kubermatic/pull/9740))
- Fix user cluster owner when you create a cluster from the template ([#9388](https://github.com/kubermatic/kubermatic/pull/9388))
- Make sure that kubelet-configmap(s) are up-to-date after updating KKP ([#9744](https://github.com/kubermatic/kubermatic/pull/9744))
- The `mirror-images` command in the kubermatic-installer loads more images that were missing before (OpenStack CSI, user-ssh-keys-agent, operating-system-manager) in the `image-loader` ([#9871](https://github.com/kubermatic/kubermatic/pull/9871))
- Fix: Consider components override for etcd PDB ([#9998](https://github.com/kubermatic/kubermatic/pull/9998))

### Deprecations

- In the Seed CRD, `spec.NodeportProxy.Annotations` is deprecated and `spec.NodeportProxy.Envoy.LoadBalancerService.Annotations` should be used instead ([#9476](https://github.com/kubermatic/kubermatic/pull/9476))
- The deprecated configuration options `KubermaticConfiguration.spec.seedController.backupRestore` and `Seed.spec.backupRestore` have been removed. Please migrate to `Seed.spec.etcdBackupRestore` ([#9003](https://github.com/kubermatic/kubermatic/pull/9003))
- Deprecate Canal CNI v3.19 ([#10289](https://github.com/kubermatic/kubermatic/pull/10289))
- `User.spec.id` is marked as deprecated/optional, as this field is not used anymore ([#9538](https://github.com/kubermatic/kubermatic/pull/9538))
- `UserSSHKey.spec.owner` is marked as deprecated/optional, as this field is not used anymore ([#9538](https://github.com/kubermatic/kubermatic/pull/9538))

### Miscellaneous

- A new `HeadlessInstallation` (preview, not yet ready for production) feature flag can be used to disable the KKP UI, API and Ingress, which will also skip installing nginx/Dex/cert-manager. Use this if you intend to only access the master/seed clusters directly and need no user separation ([#9544](https://github.com/kubermatic/kubermatic/pull/9544))
- A webhook now validates `MLAAdminSetting` resources and restricts their creation to cluster namespaces ([#9318](https://github.com/kubermatic/kubermatic/pull/9318))
- Add API endpoints for managing IPAM pools ([#10229](https://github.com/kubermatic/kubermatic/pull/10229))
- Add Canal CNI v3.22 support & make it the default CNI. NOTE: Automatically upgrades Canal to v3.22 in clusters with k8s v1.23 and higher and older Canal version ([#9258](https://github.com/kubermatic/kubermatic/pull/9258))
- Add Deployment with CLI tools for the user cluster web terminal ([#9696](https://github.com/kubermatic/kubermatic/pull/9696))
- Add MetalLB addon integrated with multi-cluster IPAM ([#10426](https://github.com/kubermatic/kubermatic/pull/10426))
- Add `--skip-dependencies` flag to kubermatic-installer that skips downloading Helm chart dependencies (requires chart dependencies to be downloaded already) ([#10348](https://github.com/kubermatic/kubermatic/pull/10348))
- Add `configuration_name` parameter for the metering report delete endpoint: `DELETE /api/v1/admin/metering/reports/${reportName}` ([#9699](https://github.com/kubermatic/kubermatic/pull/9699))
- Add a controller to monitor preset deletion. All affected clusters will be annotated. The end-user can make the decision to migrate the cluster credentials ([#9545](https://github.com/kubermatic/kubermatic/pull/9545))
- Add a validation webhook for `ClusterTemplate` CRs ([#10007](https://github.com/kubermatic/kubermatic/pull/10007))
- Add an endpoint for OIDC kubeconfig secret for the web terminal ([#10102](https://github.com/kubermatic/kubermatic/pull/10102))
- Add credential validation for Hetzner and Equinixmetal ([#9051](https://github.com/kubermatic/kubermatic/pull/9051))
- Add darwin-arm64 to platforms supported by release binaries ([#8964](https://github.com/kubermatic/kubermatic/pull/8964))
- Add handling for the creation of applications at cluster creation time and from cluster templates ([#9655](https://github.com/kubermatic/kubermatic/pull/9655))
- Add missing credentials reference for cluster templates ([#9865](https://github.com/kubermatic/kubermatic/pull/9865))
- Add optional parameter allowing to set retention for metering configurations ([#9787](https://github.com/kubermatic/kubermatic/pull/9787))
- Add possibility to import and export cluster templates from/to file ([#8864](https://github.com/kubermatic/kubermatic/pull/8864))
- Add preset sync controller ([#9478](https://github.com/kubermatic/kubermatic/pull/9478))
- Add support for Canal CNI v3.23 & make it the default CNI, deprecate Canal CNI v3.19 ([#10289](https://github.com/kubermatic/kubermatic/pull/10289))
- Add support for Cilium CNI & Hubble v1.12 ([#10434](https://github.com/kubermatic/kubermatic/pull/10434))
- Add webhook to validate `KubermaticConfiguration` objects ([#9326](https://github.com/kubermatic/kubermatic/pull/9326))
- All pods created by KKP are assigned the `RuntimeDefault` seccomp profile ([#9053](https://github.com/kubermatic/kubermatic/pull/9053))
- All webhooks have been moved from the controller-managers into a standalone webhook Deployment; it is now possible again to scale up the seed/master controller-managers to more than 1 replica without running into webhook-related issues ([#8566](https://github.com/kubermatic/kubermatic/pull/8566))
- Audit logging presets `recommended` and `minimal` now include ResponseRequest level logging for `Machine`, `MachineSets` and `MachineDeployments`, any Gatekeeper template resources, and the user-ssh-keys-agent secret for SSH keys ([#9807](https://github.com/kubermatic/kubermatic/pull/9807))
- Auto generated names for the MachineDeployments now contain the cluster name as prefix, instead of the cluster human readable name ([#9586](https://github.com/kubermatic/kubermatic/pull/9586))
- Dynamic kubelet configuration is rejected by the KKP API for `NodeDeployments` with Kubernetes 1.24 or higher ([#9892](https://github.com/kubermatic/kubermatic/pull/9892))
- Extend web terminal Pod for dedicated in-memory storage ([#9902](https://github.com/kubermatic/kubermatic/pull/9902))
- For Kubernetes 1.22 and higher, etcd is updated to v3.5.3 to fix data consistency issues as reported by upstream developers ([#9604](https://github.com/kubermatic/kubermatic/pull/9604))
- For user clusters that use etcd 3.5 (Kubernetes 1.22 clusters), etcd corruption checks are turned on to detect [etcd data consistency issues](https://github.com/etcd-io/etcd/issues/13766). Checks run at etcd startup and every 4 hours ([#9477](https://github.com/kubermatic/kubermatic/pull/9477))
- Improve cluster deletion by emitting events to make it easier to disagnose stuck clusters ([#10359](https://github.com/kubermatic/kubermatic/pull/10359))
- Improve log verbosity ([#10325](https://github.com/kubermatic/kubermatic/pull/10325))
- Init-container `etcd-running` that check if etcd is ready before starting api-server has now a retry limit ([#9403](https://github.com/kubermatic/kubermatic/pull/9403))
- KKP API does not omit `replicas` field from `NodeDeployment` responses if set to zero ([#9679](https://github.com/kubermatic/kubermatic/pull/9679))
- KKP will create default `AddonConfig` objects for the addons it ships. To customize them, remove the `app.kubernetes.io/managed-by` label from an AddonConfig, after which it will no longer be reconciled by KKP ([#10753](https://github.com/kubermatic/kubermatic/pull/10753))
- KKP now updates clusters according to the Kubernetes version skew policy ([#9375](https://github.com/kubermatic/kubermatic/pull/9375))
- Link preset with the user cluster ([#9455](https://github.com/kubermatic/kubermatic/pull/9455))
- Lowered the default `defaultNodeSize` which is set in KubermaticSetting during KKP installation from 10 to 2. This only affects new KKP installations ([#10838](https://github.com/kubermatic/kubermatic/pull/10838))
- Make Cilium non-exclusive CNI for compatibility with Multus ([#8915](https://github.com/kubermatic/kubermatic/pull/8915))
- Make user cluster kubelet resource metrics available ([#10603](https://github.com/kubermatic/kubermatic/pull/10603))
- Making telemetry UUID field optional ([#9900](https://github.com/kubermatic/kubermatic/pull/9900))
- Monitoring: `KubeCPUOvercommit` and `KubeMemOvercommit` alerts now calculate available resources more accurately ([#9739](https://github.com/kubermatic/kubermatic/pull/9739))
- New flag for the UserSettings to allow users saving their view preferences ([#9926](https://github.com/kubermatic/kubermatic/pull/9926))
- New redirect URIs introduced in Dex configuration for web terminal and Kubernetes dashboard ([#10104](https://github.com/kubermatic/kubermatic/pull/10104))
- Remove ipv6 from the dnat-controller of openvpn-client ([#9552](https://github.com/kubermatic/kubermatic/pull/9552))
- Support custom pod resources for NodePortProxy pod for the user cluster ([#8859](https://github.com/kubermatic/kubermatic/pull/8859))
- The KKP API does not use `cluster-admin` permissions anymore ([#10113](https://github.com/kubermatic/kubermatic/pull/10113))
- The KKP Operator now respects worker-name labels, making development on shared clusters much easier ([#9138](https://github.com/kubermatic/kubermatic/pull/9138))
- The KKP webhook now ensures that Addons are only created in cluster namespaces and assigns a proper Cluster reference ([#9205](https://github.com/kubermatic/kubermatic/pull/9205))
- The KKP webhook now ensures that `UserSSHKey` fingerprints always match their public key ([#9200](https://github.com/kubermatic/kubermatic/pull/9200))
- The Master/Seed MLA Prometheus from `charts/monitoring/prometheus` supports annotating Pods with `prometheus.io/scheme=https` to use HTTPS ([#9662](https://github.com/kubermatic/kubermatic/pull/9662))
- The `clusters.k8s.io/Cluster` CRD is not being reconciled into userclusters anymore, as it served no purpose ([#9844](https://github.com/kubermatic/kubermatic/pull/9844))
- The `image-loader` utility has been removed and its functionality is available via the installer's `mirror-images` subcommand instead; a `--docker-binary` flag has been added to `kubermatic-installer mirror-images` to specify a custom docker binary ([#10129](https://github.com/kubermatic/kubermatic/pull/10129))
- The `kubermatic-installer` now rejects `--config` files that are not actually valid `KubermaticConfiguration` objects ([#10737](https://github.com/kubermatic/kubermatic/pull/10737))
- The cluster validation webhook now validates that Cluster objects have a proper `project-id` label pointing to an existing project ([#9292](https://github.com/kubermatic/kubermatic/pull/9292))
- The default CA bundle (provided by Mozilla) was updated from 2021-04-13 to 2022-04-26 ([#10052](https://github.com/kubermatic/kubermatic/pull/10052))
- The kubermatic-installer uses Cobra instead of urfave/cli, but the CLI flags are identical, so no changes to scripts or automation should be necessary ([#9398](https://github.com/kubermatic/kubermatic/pull/9398))
- The legacy `owner-remover` tool was removed from the codebase ([#9474](https://github.com/kubermatic/kubermatic/pull/9474))
- The version of the installed Kubernetes dashboard now depends on the usercluster version ([#8746](https://github.com/kubermatic/kubermatic/pull/8746))
- Unused flatcar update resources are removed when no Flatcar `Nodes` are in a user cluster ([#8745](https://github.com/kubermatic/kubermatic/pull/8745))
- Update example values and KubermaticConfiguration to respect OIDC settings ([#8851](https://github.com/kubermatic/kubermatic/pull/8851))
- Update machine-controller CRDs additionalPrinterColumns to match upstream ([#10354](https://github.com/kubermatic/kubermatic/pull/10354))
- Use batch/v1 API for CronJob resources ([#10219](https://github.com/kubermatic/kubermatic/pull/10219))
- Use policy/v1 API for PodDisruptionBudget resources (Seed minimum Kubernetes version is now 1.21) ([#10162](https://github.com/kubermatic/kubermatic/pull/10162))
- Use quay.io as the default registry for Canal CNI images ([#10305](https://github.com/kubermatic/kubermatic/pull/10305))
- User Cluster MLA version updates: Prometheus v2.36.2, promtail v2.5.0 ([#10322](https://github.com/kubermatic/kubermatic/pull/10322))
- When updating an existing KKP installation, `--config` must not be specified for the kubermatic-installer. Instead the current configuration is loaded from the KKP master cluster ([#10533](https://github.com/kubermatic/kubermatic/pull/10533))
- `Cluster` resources are validated against supported Kubernetes versions as defined in `KubermaticConfiguration` or as defaulted by the respective KKP release ([#9912](https://github.com/kubermatic/kubermatic/pull/9912))
- `kubermatic-installer` output can be formatted as json via a new `--output` flag ([#10137](https://github.com/kubermatic/kubermatic/pull/10137))
- `oauth` (Dex) Helm chart supports mounting extra volumes into Dex Deployment to supply data from outside the chart ([#10364](https://github.com/kubermatic/kubermatic/pull/10364))
- etcd backup files are named differently (`foo-YYYY-MM-DDThh:mm:ss` to `foo-YYYY-MM-DDThhmmss.db`) to improve compatibility with different storage solutions ([#10143](https://github.com/kubermatic/kubermatic/pull/10143))
- etcd-launcher now recovers from a PVC deletion when restarting with a fresh data volume ([#9600](https://github.com/kubermatic/kubermatic/pull/9600))
- kubermatic-installer: improve error handling when building Helm chart dependencies ([#9851](https://github.com/kubermatic/kubermatic/pull/9851))
- kubermatic-operator is deployed with leader election by default, can be disabled from Chart values ([#9722](https://github.com/kubermatic/kubermatic/pull/9722))
- s3-storeuploader uses Cobra instead of urfave/cli, but the CLI flags are identical, so no changes to backup containers should be necessary ([#9394](https://github.com/kubermatic/kubermatic/pull/9394))
- ClusterRole to list namespaces is shipped with the RBAC addon ([#10729](https://github.com/kubermatic/kubermatic/pull/10729))

### Updates

- Update Anexia CCM (cloud-controller-manager) to version 1.4.3 ([#10507](https://github.com/kubermatic/kubermatic/pull/10507))
- Update aws-node-termination-handler addon to v1.16.2 ([#9716](https://github.com/kubermatic/kubermatic/pull/9716))
- Update Canal 3.20 version to v3.20.5 ([#10305](https://github.com/kubermatic/kubermatic/pull/10305))
- Update Canal 3.21 version to v3.21.6 ([#10491](https://github.com/kubermatic/kubermatic/pull/10491))
- Update Canal 3.22 version to v3.22.4 ([#10499](https://github.com/kubermatic/kubermatic/pull/10499))
- Update Canal 3.23 version to v3.23.3 ([#10531](https://github.com/kubermatic/kubermatic/pull/10531))
- Update Cilium to v1.11.6 & Hubble to v0.9.0 ([#10331](https://github.com/kubermatic/kubermatic/pull/10331))
- Update controller-runtime to 0.12.3 / Kubernetes 1.24 ([#9826](https://github.com/kubermatic/kubermatic/pull/9826))
- Update dns-node-cache to 1.21.1 ([#9850](https://github.com/kubermatic/kubermatic/pull/9850))
- Update etcd for Kubernetes 1.22+ to 3.5.4 ([#9832](https://github.com/kubermatic/kubermatic/pull/9832))
- Update Go to 1.18.4 ([#10416](https://github.com/kubermatic/kubermatic/pull/10416))
- Update Helm in Docker image to 3.8.1 ([#9780](https://github.com/kubermatic/kubermatic/pull/9780))
- Update Konnectivity version to v0.0.31 ([#10112](https://github.com/kubermatic/kubermatic/pull/10112))
- Update Kubernetes dashboard to 2.6.0 for Kubernetes 1.24 ([#9948](https://github.com/kubermatic/kubermatic/pull/9948))
- Update kubevirt CCM to v0.1.0 ([#9404](https://github.com/kubermatic/kubermatic/pull/9404))
- Update machine-controller to v1.54.0 (image location moved from `docker.io/kubermatic/machine-controller` to `quay.io/kubermatic/machine-controller`) ([#9825](https://github.com/kubermatic/kubermatic/pull/9825), [#10856](https://github.com/kubermatic/kubermatic/pull/10856))
- Update metrics-scraper to 1.0.8 ([#9948](https://github.com/kubermatic/kubermatic/pull/9948))
- Update metrics-server to 0.6.1 ([#9849](https://github.com/kubermatic/kubermatic/pull/9849))
- Update Multus to v3.8.1, enable Multus auto-configuration ([#8900](https://github.com/kubermatic/kubermatic/pull/8900))
- Update OPA Gatekeeper to 3.6.0 ([#8973](https://github.com/kubermatic/kubermatic/pull/8973))
- Update OSM to v1.0.0 ([#10856](https://github.com/kubermatic/kubermatic/pull/10856))
- Update usercluster Prometheus to 2.33.3 ([#8992](https://github.com/kubermatic/kubermatic/pull/8992))
- Update Vertical Pod Autoscaler to 0.11 ([#10395](https://github.com/kubermatic/kubermatic/pull/10395))
- Update vSphere CCM to 1.23.1 (for Kubernetes 1.23.x) and 1.24.0 (for Kubernetes 1.24.x) ([#10203](https://github.com/kubermatic/kubermatic/pull/10203))

### Helm Chart Updates

- Alertmanager 0.24.0 ([#9636](https://github.com/kubermatic/kubermatic/pull/9636))
- blackbox exporter 0.21.1 ([#10396](https://github.com/kubermatic/kubermatic/pull/10396))
- cert-manager 1.9.1 ([#10518](https://github.com/kubermatic/kubermatic/pull/10518))
- Dex 2.32.0 ([#10083](https://github.com/kubermatic/kubermatic/pull/10083))
- Grafana 9.0.1 ([#10195](https://github.com/kubermatic/kubermatic/pull/10195))
- Helm-Exporter 1.2.2 ([#9642](https://github.com/kubermatic/kubermatic/pull/9642))
- ingress-nginx 1.2.1 ([#10036](https://github.com/kubermatic/kubermatic/pull/10036))
- karma v0.103 ([#10085](https://github.com/kubermatic/kubermatic/pull/10085))
- kube-state-metrics 2.5.0 ([#9974](https://github.com/kubermatic/kubermatic/pull/9974))
- Loki 2.5 ([#9648](https://github.com/kubermatic/kubermatic/pull/9648))
- Minio RELEASE.2022-06-25T15-50-16Z ([#10193](https://github.com/kubermatic/kubermatic/pull/10193))
- nginx 1.3.0 ([#10441](https://github.com/kubermatic/kubermatic/pull/10441))
- node-exporter 1.3.1 ([#9640](https://github.com/kubermatic/kubermatic/pull/9640))
- oauth2-proxy (IAP) v7.3.0 ([#10081](https://github.com/kubermatic/kubermatic/pull/10081))
- Prometheus 2.37 (disables support for SHA-1 certificates and TLS 1.0/1.1) ([#10396](https://github.com/kubermatic/kubermatic/pull/10396))
- Promtail v2.5.0 ([#10082](https://github.com/kubermatic/kubermatic/pull/10082))
- Velero 1.9.0 ([#10192](https://github.com/kubermatic/kubermatic/pull/10192))
- The long deprecated `kubermatic` Helm chart has finally been removed. It was unusable for quite some time and every KKP setup should use the KKP Operator instead (from the `kubermatic-operator` chart) ([#9110](https://github.com/kubermatic/kubermatic/pull/9110))
