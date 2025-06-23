# Kubermatic 2.28

- [v2.28.0](#v2280)

## v2.28.0

**GitHub release: [v2.28.0](https://github.com/kubermatic/kubermatic/releases/tag/v2.28.0)**

Before upgrading, make sure to read the [general upgrade guidelines](https://docs.kubermatic.com/kubermatic/v2.28/installation/upgrading/). Consider tweaking `seedControllerManager.maximumParallelReconciles` to ensure user cluster reconciliations will not cause resource exhaustion on seed clusters. A [full upgrade guide is available from the official documentation](https://docs.kubermatic.com/kubermatic/v2.28/installation/upgrading/upgrade-from-2.27-to-2.28/).

### Breaking Changes

- The deprecated OAuth helm chart has been removed. Before proceeding with KKP upgrade, please follow the [dex migration procedure](https://docs.kubermatic.com/kubermatic/v2.27/installation/upgrading/upgrade-from-2.26-to-2.27/#dex-v242) to migrate from oauth/dex to the new upstream Dex chart ([#14657](https://github.com/kubermatic/kubermatic/pull/14657))
- `deploy-default-app-catalog` option from kubermatic-installer has been deprecated and would have no affect going forward. The field `.Spec.Applications.DefaultApplicationCatalog` can be used instead to manage default application catalog ([#14697](https://github.com/kubermatic/kubermatic/pull/14697))
- Support for multiple security groups has been dropped for OpenStack. If you are using multiple security groups for an OpenStack cluster, you need to manually adjust the `cluster.Spec.Cloud.Openstack.SecurityGroups` ([#14269](https://github.com/kubermatic/kubermatic/pull/14269))
- Node-exporter chart is now using the upstream helm chart. This means there are some adjustment required to the `values.yaml`. replace the `nodeExporter` with `node-exporter` ([#14176](https://github.com/kubermatic/kubermatic/pull/14176)) ([#14670](https://github.com/kubermatic/kubermatic/pull/14670))
- Alertmanager chart was replaced with upstream chart. This means, some of the old values might need migration. Please review documentation ([#14175](https://github.com/kubermatic/kubermatic/pull/14175))  ([#14675](https://github.com/kubermatic/kubermatic/pull/14675))
- Blackbox-exporter chart is now using the upstream helm chart indirectly. So the customizations for blackbox-exporter via helm values.yaml should be moved under key `blackbox-exporter` instead of `blackboxExporter` ([#14170](https://github.com/kubermatic/kubermatic/pull/14170))  ([#14675](https://github.com/kubermatic/kubermatic/pull/14675))
- Kube-state-metrics chart is now using the upstream helm chart with app version 2.15.0 ([#14357](https://github.com/kubermatic/kubermatic/pull/14357)) ([#14675](https://github.com/kubermatic/kubermatic/pull/14675))
- New roles for the Kubevirt csi driver have been added and users must add these role to KubeVirt infra cluster kubeconfig service account ([#14502]( https://github.com/kubermatic/kubermatic/pull/14502))

### ACTION REQUIRED

- Update cert-manager to v1.16.5. So the following updates should be done values.yaml ([#14400](https://github.com/kubermatic/kubermatic/pull/14400)): 
    - update `webhook.replicas` to `webhook.replicaCount` 
    - update `cainjector.replicas` to `webhook.replicaCount` 
    - remove `webhook.injectAPIServerCA` 

### API Changes

- The KKP APIs have been moved into a Go module named `k8c.io/kubermatic/v2/sdk` ([#14171](https://github.com/kubermatic/kubermatic/pull/14171))

### Supported Kubernetes Versions

- Remove support for Kubernetes 1.29 ([#14345](https://github.com/kubermatic/kubermatic/pull/14345))
- Add support for Kubernetes 1.33 ([#14419](https://github.com/kubermatic/kubermatic/pull/14419))
- Add 1.32.4/1.32.3/1.31.8/1.31.7/1.30.12/1.30.11 to the list of supported Kubernetes releases ([#14266](https://github.com/kubermatic/kubermatic/pull/14266)) ([#14385](https://github.com/kubermatic/kubermatic/pull/14385))

#### Supported Versions
* 1.30.11
* 1.30.12
* 1.31.7
* 1.31.8
* 1.32.3
* 1.32.4
* 1.33.0

### Cloud Providers 
#### KubeVirt
- Support KubeVirt Subnet CIDR in the Seed object ([#14604](https://github.com/kubermatic/kubermatic/pull/14604))
- The field `enabledDedicatedCPUs` in kubevirt provider spec is now deprecated. A new field called `useDomainCPU` is introduced which is for the same purpose. When set to `true` cpu will be assigned by `spec.domain.cpu` for a kubevirt virtual machine instead of using resource requests and limits ([#14621](https://github.com/kubermatic/kubermatic/pull/14621))
- Update KubeVirt CSI Driver Operator to v0.4.3 ([#14178](https://github.com/kubermatic/kubermatic/pull/14178))
- A bug that caused network policies to not be removed from the kubevirt infra cluster has been fixed ([#14429](https://github.com/kubermatic/kubermatic/pull/14429))
- Support `infra-csi-driver` as a `volumeProvisioner` for the KubeVirt CSI Driver ([#14199](https://github.com/kubermatic/kubermatic/pull/14199))
- Add ability to disable automatic installation of default kubevirt instance types and preferences ([#14426](https://github.com/kubermatic/kubermatic/pull/14426))
- A new field `spec.datacenters.<example-dc>.spec.kubevirt.enableDedicatedCpus` was added to seed crd to control whether kubevirt machine cpus are configured by `spec.template.spec.domain.resources` with requests and limits or `spec.template.spec.domain.cpu` . Later one is required to use kubevirt cpu allocation ratio feature ([#14298](https://github.com/kubermatic/kubermatic/pull/14298))
- A new field was introduced for kubevirt provider in namespacedmode in enterprise edition to configure a mode for the deployed network policy in kubevirt infrastructure clusters. Default mode is `allow` which allows by default all traffic except to other providernetwork subnets. The other option is `deny` which denies all traffic except to the usercluster apiserver, configured nameservers and between worker nodes ([#14390](https://github.com/kubermatic/kubermatic/pull/14390))
- Support KubeVirt CCM Load Balancer Interface Disabling ([#14521](https://github.com/kubermatic/kubermatic/pull/14521))

#### OpenStack
- Fix reconciliation loop for routers in Openstack provider by allowing updates to the `routerID` field ([#14420](https://github.com/kubermatic/kubermatic/pull/14420))
- User Clusters in Openstack can share a router, which is deleted only after all associated clusters are removed ([#14468](https://github.com/kubermatic/kubermatic/pull/14468))
- Fix auto-deletion of non-created OpenStack security group during cluster cleanup ([#14359](https://github.com/kubermatic/kubermatic/pull/14359))
- Fix router-subnet-link cleanup for OpenStack user clusters created with existing networks and subnets ([#14153](https://github.com/kubermatic/kubermatic/pull/14153))
- Add support for configuring multiple named `LoadBalancerClasses` for OpenStack cloud.cfg. This allows users to define and utilize different load balancer configurations within their OpenStack environments managed by Kubermatic ([#14362](https://github.com/kubermatic/kubermatic/pull/14362))
- Add revision label support to OpenStack CSI controller deployment to trigger pod restarts when cloud-config-csi secret is updated ([#14410](https://github.com/kubermatic/kubermatic/pull/14410))

### New Features

- EE: LocalAI is added to the default applications catalog ([#14700](https://github.com/kubermatic/kubermatic/pull/14700))
- The default apps from the enterprise catalog are now also part of the kubermatic-installer `mirror-images` command ([#14683](https://github.com/kubermatic/kubermatic/pull/14683))
- Add Kyverno images to `mirror-images` command ([#14681](https://github.com/kubermatic/kubermatic/pull/14681))
- Add `--insecure` flag to mirror-images command to bypass TLS verification ([#14635](https://github.com/kubermatic/kubermatic/pull/14635))
- Add support for configuring an Authorization Webhook for the User Clusters ([#13930](https://github.com/kubermatic/kubermatic/pull/13930))
- Mla-secrets chart can now be safely redeployed, existing secrets take precedence over provided values ([#14568](https://github.com/kubermatic/kubermatic/pull/14568))
- Gateway API CRDs are now automatically installed in the user clusters, when KubeLB integration is enabled ([#14368](https://github.com/kubermatic/kubermatic/pull/14368))
- Add support for scheduling options (tolerations, affinity, nodeSelector) to the KKP operator Helm chart, allowing users to control pod placement in their clusters ([#14574](https://github.com/kubermatic/kubermatic/pull/14574))
- Add nginx config to increase header size for Dashboard and Dex ([#14579](https://github.com/kubermatic/kubermatic/pull/14579))
- EE: The default Policy Template Catalog can be deployed via `--deploy-default-policy-template-catalog` flag  ([#14472](https://github.com/kubermatic/kubermatic/pull/14472))
- A new `--limit-apps` flag has been added to the Kubermatic Installer, allowing users to limit which AppDefinitions are installed during the setup process. This flag accepts a comma-separated list of AppDefinition names. If the flag is not provided or the list is empty, all available AppDefinitions will be installed—provided the default app catalog is enabled ([#14569](https://github.com/kubermatic/kubermatic/pull/14569)
- KubeLB: New field extraArgs has been introduced for KubeLB at the Seed and cluster level. This field can be used to configure extra arguments for the KubeLB CCM ([#14564](https://github.com/kubermatic/kubermatic/pull/14564))
- KubeLB: The configuration in the Seed seed.spec.kubelb.enableForAllDatacenters can be used to allow KubeLB installation for all the datacenters belonging to the Seed ([#14558](https://github.com/kubermatic/kubermatic/pull/14558))
- Install cluster-autoscaler by default as a system application via kubermatic-operator ([#14509](https://github.com/kubermatic/kubermatic/pull/14509))
- Add support for tagging cluster backup objects in KKP for improved management and traceability. It can help to identify, categorise, and track backup resources more effectively across multiple clusters and tenants ([#14373](https://github.com/kubermatic/kubermatic/pull/14373))
- Add `AuditLogging` configuration on seed level ([#14464](https://github.com/kubermatic/kubermatic/pull/14464))
- Add the ability to define the allowed IP Ranges for the API server for the user cluster on the Seed Level ([#14462](https://github.com/kubermatic/kubermatic/pull/14462))
- Add environment variable support for audit logging sidecar ([#14437](https://github.com/kubermatic/kubermatic/pull/14437))
- Add the ability to disable `UserSSHKey` feature in Kubermatic ([#14425](https://github.com/kubermatic/kubermatic/pull/14425))
- Add support for GlobalViewer role. Users marked with isGlobalViewer: true now gain read-only access to all projects without being explicitly added to them ([#14433](https://github.com/kubermatic/kubermatic/pull/14433))
- Allow users to set `--quota-backend-bytes` for etcd ([#14367](https://github.com/kubermatic/kubermatic/pull/14367))
- Add support for configuring `backup-interval` and `backup-count` at the seed, KubermaticConfiguration, and controller flag levels ([#14361](https://github.com/kubermatic/kubermatic/pull/14361))
- Prow jobs and E2E tests are added. There are prow jobs for each application from the default app catalog, and each job verifies that the ApplicationInstallation object has been installed successfully, its conditions are in a healthy state, a Helm release has been deployed, and the pods of the given application have been deployed successfully. Additionally, end-to-end (E2E) tests are added to verify that the pods for every application in the default application catalog are deployed successfully. These tests help ensure compatibility when updating the version of each application ([#14312](https://github.com/kubermatic/kubermatic/pull/14312))
- Add support for using a Proxy Between KKP and the nodeport-proxy ([#14159](https://github.com/kubermatic/kubermatic/pull/14159))
- Add a new field in KubermaticSettings to allow setting a default checksum algorithm for Velero through Dashboard ([#14253](https://github.com/kubermatic/kubermatic/pull/14253))
- Introduce the `mirror-binaries` in `kubermatic-installer` to mirror the kubernetes and container tools binaries for Offline setups ([#14251](https://github.com/kubermatic/kubermatic/pull/14251))
- Ensure `mirror-images` processes all images without blocking, logging failed images at the end for better visibility and debugging ([#14262](https://github.com/kubermatic/kubermatic/pull/14262))
- Include `cilium-envoy` image in the mirrored images ([#14238](https://github.com/kubermatic/kubermatic/pull/14238))
- Add a new field in `Cluster` to configure HTTP Proxy at User Cluster Level ([#14209](https://github.com/kubermatic/kubermatic/pull/14209))
- Support disabling default OperatingSystemProfiles in user clusters ([#14515](https://github.com/kubermatic/kubermatic/pull/14515))
- Add a new optional field called `args` under KonnectivityProxySettings to allow users to specify a set of arguments for Konnectivity deployments ([#14189](https://github.com/kubermatic/kubermatic/pull/14189))

### Bugfixes

- Fix `kubermatic-installer local kind` command for EE setups to set correct image pull secrets in `values.yaml` ([#14707](https://github.com/kubermatic/kubermatic/pull/14707))
- A bug was fixed where enforced annotation on application installation were not removed when enforcement for the related application definition was disabled ([#14706](https://github.com/kubermatic/kubermatic/pull/14706))
- Add the ability to skip charts in the `kubermatic-installer deploy usercluster-mla` command ([#14688](https://github.com/kubermatic/kubermatic/pull/14688))
- Fix references for dex host(dex.ingress.hosts[0].host) in example manifests, ([#14630](https://github.com/kubermatic/kubermatic/pull/14630))
- Remove redundant and undocumented/used `remove-oauth-release` flag for installer ([#14630](https://github.com/kubermatic/kubermatic/pull/14630))
- Add validation for checks in the installer for the new dex chart ([#14624](https://github.com/kubermatic/kubermatic/pull/14624))
- Fix `--skip-seed-validation` flag on the KKP installer ([#14585](https://github.com/kubermatic/kubermatic/pull/14585))
- Correctly mounts the custom CA bundle ConfigMap to fix reconciliation failures in custom CA environments ([#14575](https://github.com/kubermatic/kubermatic/pull/14575))
- Fix a bug where CSI Snapshot validating webhook was being deployed even if the CSI drivers are disabled for a cluster. When the csi driver is disabled after cluster creation the both mentioned resources will be cleaned up now ([#14466](https://github.com/kubermatic/kubermatic/pull/14466))
- Remove old warnings for new dex chart ([#14423](https://github.com/kubermatic/kubermatic/pull/14423))
- Fix the service account deletion process ([#14371](https://github.com/kubermatic/kubermatic/pull/14371))
- Ensure that etcd backup images are pulled from the overwrite Registry in air-gapped environments ([#14356](https://github.com/kubermatic/kubermatic/pull/14356))
- Fix a bug for KubeLB where disabling the ingress class for a user cluster was not working ([#14396](https://github.com/kubermatic/kubermatic/pull/14396))
- Node-local-dns in user clusters will now use `IfNotPresent` pull policy instead of `Always` ([#14309](https://github.com/kubermatic/kubermatic/pull/14309))
- Edge Provider: Fix a bug where clusters were stuck in `creating` phase due to wrongfully waiting for Machine Controller's health status ([#14257](https://github.com/kubermatic/kubermatic/pull/14257))
- Fix an issue where the CBSL status was not updating due to the missing cluster-backup-storage-controller in the master controller manager ([#14243](https://github.com/kubermatic/kubermatic/pull/14243))
- Fix mirroring the images of a single Kubernetes version ([#14248](https://github.com/kubermatic/kubermatic/pull/14248))
- It is now possible to configure the sidecar configuration for a given cluster while the auditLogging field is enabled at the Seed level. Previously, if the auditLogging field was enabled at the Seed level, it would override the same field at the Cluster level, resulting in the removal of the sidecar configuration ([#14145](https://github.com/kubermatic/kubermatic/pull/14145))
- Fix a Go panic when using git-source in Applications ([#14219](https://github.com/kubermatic/kubermatic/pull/14219))
- Include the etcd backup restore and delete images in the kubermatic-installer mirror-images command ([#14220](https://github.com/kubermatic/kubermatic/pull/14220))
- Add dex and gitops charts to the CI release pipeline for inclusion in the release tar ([#14192](https://github.com/kubermatic/kubermatic/pull/14192))
- Fix a bug that prevents configuring `resources` in KNP deployments ([#14205](https://github.com/kubermatic/kubermatic/pull/14205))
- Apply override registry configuration to cilium-envoy images ([#14164](https://github.com/kubermatic/kubermatic/pull/14164))
- The local kind command from the kubermatic-installer is now using helm values to deploy dex by the upstream helm chart. This was required due to removal of the old custom chart. ([#14704](https://github.com/kubermatic/kubermatic/pull/14704))
- A bug where images with digest instead of a tag are not properly parsed was fixed. This affected mirroring images and all parsing with an overwrite registry configured ([#14664](https://github.com/kubermatic/kubermatic/pull/14664))
- KubeLB: CCM will adjust the tenant kubeconfig to use API server endpoint and CA certificate from the management kubeconfig that is provided to KKP at the seed/datacenter level ([#14522](https://github.com/kubermatic/kubermatic/pull/14522))

### Updates

- Update go-jose to 3.0.4 (CVE-2025-27144) ([#14622](https://github.com/kubermatic/kubermatic/pull/14622))
- Update the default Kubernetes version to 1.32.4 ([#14634](https://github.com/kubermatic/kubermatic/pull/14634))
- Update the Helm values example files supplied with the release package to match the new Dex chart ([#14628](https://github.com/kubermatic/kubermatic/pull/14628))
- Update aikit application to v0.18.0 ([#14403](https://github.com/kubermatic/kubermatic/pull/14403))
- Update argocd application to v2.14.11 and v3.0.0 ([#14403](https://github.com/kubermatic/kubermatic/pull/14403))
- Update cert-manager application to v1.17.2 ([#14403](https://github.com/kubermatic/kubermatic/pull/14403))
- Update cluster-autoscaler application chart version to 9.46.6 ([#14403](https://github.com/kubermatic/kubermatic/pull/14403))
- Update falco application chart version to 4.21.2 ([#14403](https://github.com/kubermatic/kubermatic/pull/14403))
- Update flux2 application to 2.5.1 ([#14403](https://github.com/kubermatic/kubermatic/pull/14403))
- Update gpu-operator application to v25.3.0 ([#14403](https://github.com/kubermatic/kubermatic/pull/14403))
- Update ingress-nginx application to 4.12.2 ([#14403](https://github.com/kubermatic/kubermatic/pull/14403))
- Update k8sgpt-operator application chart version to 0.2.17 ([#14403](https://github.com/kubermatic/kubermatic/pull/14403))
- Update trivy application to 0.62.1 ([#14403](https://github.com/kubermatic/kubermatic/pull/14403))
- Update trivy-operator application to 0.26.0 ([#14403](https://github.com/kubermatic/kubermatic/pull/14403))
- Update fluent-bit container version to v4.0.0 ([#14427](https://github.com/kubermatic/kubermatic/pull/14427))
- Update etcd to 3.5.21 for all supported Kubernetes releases ([#14417](https://github.com/kubermatic/kubermatic/pull/14417))
- Update k8s-dns-node-cache to 1.25.0 ([#14409](https://github.com/kubermatic/kubermatic/pull/14409))
- Update oauth2-proxy to v7.8.2 ([#14388](https://github.com/kubermatic/kubermatic/pull/14388))
- Update the default ipv6 services range to `fd02::/108` ([#14369](https://github.com/kubermatic/kubermatic/pull/14369))
- Update the default application's nginx ingress controller to use the save and patched version of v1.12.1 ([#14341](https://github.com/kubermatic/kubermatic/pull/14341))
- Update operating-system-manager to [v1.7.0](https://github.com/kubermatic/operating-system-manager/releases/tag/v1.7.0) ([#14709](https://github.com/kubermatic/kubermatic/pull/14709))
- Update machine-controller to [v1.62.0](https://github.com/kubermatic/machine-controller/releases/tag/v1.62.0) ([#14711](https://github.com/kubermatic/kubermatic/pull/14711))
- Update etcd to 3.5.17 for all supported Kubernetes releases ([#14315](https://github.com/kubermatic/kubermatic/pull/14315))
- Update to Go 1.24.2 ([#14317](https://github.com/kubermatic/kubermatic/pull/14317))
- Update to controller-runtime 0.20.4 / Kubernetes 1.32 ([#14311](https://github.com/kubermatic/kubermatic/pull/14311))
- Update nginx-ingress-controller to 1.12.1 ([#14273](https://github.com/kubermatic/kubermatic/pull/14273))
- KubeLB: update CCM image to v1.1.5 ([#14609](https://github.com/kubermatic/kubermatic/pull/14609))
- The default CA bundle (provided by Mozilla) was updated from 2022-04-26 to 2025-02-25 ([#14439](https://github.com/kubermatic/kubermatic/pull/14439))
- Security: Update Cilium to 1.15.16 / 1.16.9 because the previous versions are affected by CVE-2025-32793 ([#14434](https://github.com/kubermatic/kubermatic/pull/14434))
- Add Cert-manager version v1.16.5 in the default applications catalog ([#14418](https://github.com/kubermatic/kubermatic/pull/14418))
- Support MatchSubnetAndStorageLocation and Subnets Regions and Zones ([#14414](https://github.com/kubermatic/kubermatic/pull/14414))
- Kube-state-metrics chart is now using the upstream helm chart with app version 2.15.0 ([#14174](https://github.com/kubermatic/kubermatic/pull/14174))



### Cleanups

- Cluster-autoscaler has been removed from the default accessible addons list ([#14689](https://github.com/kubermatic/kubermatic/pull/14689))
- Deprecate Equinix Metal provider ([#14448](https://github.com/kubermatic/kubermatic/pull/14448))

### Deprecations

- `deploy-default-app-catalog` for kubermatic-installer has been deprecated and would have no affect going forward. The field `.Spec.Applications.DefaultApplicationCatalog` can be used instead to manage default application catalog ([#14697](https://github.com/kubermatic/kubermatic/pull/14697))
- Default Application Catalog can now be managed via KubermaticConfiguration through the field `.Spec.Applications.DefaultApplicationCatalog` ([#14697](https://github.com/kubermatic/kubermatic/pull/14697))

### Miscellaneous

- The deprecated k8sgpt application has been removed and was replaced by the k8sgpt-operator app instead ([#14403](https://github.com/kubermatic/kubermatic/pull/14403))
- Disable cilium-envoy daemonset, if it was not specified in the chart values ([#14173](https://github.com/kubermatic/kubermatic/pull/14173))

### Dashboard and API 

#### Breaking Changes

- The `usePodResourcesCPU` feature will replace `enableDedicatedCPUs` flag. In the time of deprecation both are taking effect but the new value will have more priority. When `enableDedicatedCPUs` is set to `false` which is also the default value, you need to set `usePodResourcesCPU` to `true` to keep the same behaviour as before for new created machines. If `enableDedicatedCPUs` was set to `true` nothing needs to be changed ([#7413](https://github.com/kubermatic/dashboard/pull/7413))

#### Cloud Providers

##### KubeVirt

- Add the ability to disable the automatic installation of default kubevirt instance types and preferences ([#7304](https://github.com/kubermatic/dashboard/pull/7304))
- Display KubeVirt Subnet CIDRs in UI ([#7369](https://github.com/kubermatic/dashboard/pull/7369))
- The kkp api is now aware on how to configure cpus for kubevirt virtual machines based on a new introduced field in kkp seed crd called `spec.datacenters.<example-dc>.spec.kubevirt.enableDedicatedCpus` ([#7252](https://github.com/kubermatic/dashboard/pull/7252))

##### Openstack

- List all OpenStack networks in the UI wizard during cluster creation ([#7437](https://github.com/kubermatic/dashboard/pull/7437))
- Pass ConfigDrive value to JSON patch during machine updates for OpenStack ([#7299](https://github.com/kubermatic/dashboard/pull/7299))

##### VSphere

- Use infra management user credentials (if configured) to fetch data for vSphere ([#7397](https://github.com/kubermatic/dashboard/pull/7397))

#### New Features

- Allow manual installation of system applications except with type `cni` ([#7424](https://github.com/kubermatic/dashboard/pull/7424))
- Add Kyverno as the native Kubernetes policy solution in the dashboard ([#7323](https://github.com/kubermatic/dashboard/pull/7323))
- Display source of cluster backups ([#7348](https://github.com/kubermatic/dashboard/pull/7348))
- Cluster backups created by KKP controllers now include spec.labels to distinguish controller-initiated backups from those manually uploaded via the UI ([#7345](https://github.com/kubermatic/dashboard/pull/7345))
- Support for enabling KubeLB at a seed level for all datacenters ([#7350](https://github.com/kubermatic/dashboard/pull/7350))
- Add functionality to upload backups to cluster backup storage location ([#7335](https://github.com/kubermatic/dashboard/pull/7335))
- New page in the admin panel to manage the Global Viewer role ([#7337](https://github.com/kubermatic/dashboard/pull/7337))
- Add functionality to configure checksum algorithm for backup storage location ([#7346](https://github.com/kubermatic/dashboard/pull/7346))
- Add new feature gate to disable User SSH key ([#7324](https://github.com/kubermatic/dashboard/pull/7324))
- Users marked with globalViewer: true now receive read-only access to all projects and clusters via dynamic groups and roles injection. No need to create UserProjectBindings for them ([#7318](https://github.com/kubermatic/dashboard/pull/7318))
- Allow setting a default checksum algorithm for Velero ([#7231](https://github.com/kubermatic/dashboard/pull/7231))
- Add new API endpoints for Kyverno integration ([#7106](https://github.com/kubermatic/dashboard/pull/7106))
- Dashboard has been upgraded to use Angular 19 ([#7183](https://github.com/kubermatic/dashboard/pull/7183))
- Support infra storage classes and provider network subnets location compatibilities ([#7301](https://github.com/kubermatic/dashboard/pull/7301))

#### Bugfixes

- Unset backup sync period if value is empty ([#7444](https://github.com/kubermatic/dashboard/pull/7444))
- Fix clickable documentation links in hints for disabled checkboxes ([#7434](https://github.com/kubermatic/dashboard/pull/7434))
- Shows custom disk fields when a custom disk is configured in the Machine Deployment edit dialog ([#7415](https://github.com/kubermatic/dashboard/pull/7415))
- Cluster backup schedules created by KKP controllers now include backupSpec.labels to distinguish controller-initiated backups from those manually uploaded via the UI ([#7396](https://github.com/kubermatic/dashboard/pull/7396))
- Display system applications in cluster creation wizard and fix application type label for system applications ([#7388](https://github.com/kubermatic/dashboard/pull/7388))
- Make the Subnets field required when a VPC is selected, in both Wizard and Machine Deployment modes ([#7305](https://github.com/kubermatic/dashboard/pull/7305))
- Disable the Cluster Autoscaler option when the cluster autoscaler application is not defined in applications catalog ([#7283](https://github.com/kubermatic/dashboard/pull/7283))
- Add special characters restriction on Inputs and escape values to avoid rendering as HTML ([#7273](https://github.com/kubermatic/dashboard/pull/7273))
- Add role prioritization: Update logic to return the highest-priority role for members with multiple roles ([#7272](https://github.com/kubermatic/dashboard/pull/7272))
- Fix KKP login issue when the ID token is too large to be saved in a cookie, by splitting the token into multiple cookies ([#7206](https://github.com/kubermatic/dashboard/pull/7206))

#### Updates

- Update web-terminal image to v0.10.0 ([#7254](https://github.com/kubermatic/dashboard/pull/7254))
- Update to Go 1.24.2 ([#7253](https://github.com/kubermatic/dashboard/pull/7253))
- Update Dashboard API to use correct OSP which is selected while creating a cluster ([#7217](https://github.com/kubermatic/dashboard/pull/7217))

#### Deprecations

- Deprecate Equinix Metal provider ([#7376](https://github.com/kubermatic/dashboard/pull/7376))
