# Kubermatic 2.30

- [v2.30.0](#v2300)

## v2.30.0

**GitHub release: [v2.30.0](https://github.com/kubermatic/kubermatic/releases/tag/v2.30.0)**

### Breaking Changes

This release contains changes that require additional attention, please read the following items carefully.

- Potential BREAKING CHANGE: Add checksum-based restart for kube-proxy on cluster network changes ([#15533](https://github.com/kubermatic/kubermatic/pull/15533))
- Fix cluster-autoscaler RBAC permissions. ⚠️ Potentially BREAKING Change: cluster-autoscaler application needs to be re-installed to force recreating the ApplicationInstallation resource in order to get the new updated default values.yaml ([#15152](https://github.com/kubermatic/kubermatic/pull/15152))
- Update oauth2-proxy to appversion v7.14.2. ⚠️ Potentially BREAKING Change: Major Alpha Config YAML parsing has been revamped for better extensibility in preparation for v8. Please review https://github.com/oauth2-proxy/oauth2-proxy/releases/tag/v7.14.0 for more details ([#15499](https://github.com/kubermatic/kubermatic/pull/15499))
- Update to User-Cluster MLA Cortex and Consul charts. All User-Cluster MLA Cortex and Consul pods will restart and be upgraded to the latest versions. Action required: If you had configured a startupProbe for the Cortex compactor in your values.yaml, that entire configuration must be removed, as the latest version of Cortex compactor does not include a startupProbe ([#15356](https://github.com/kubermatic/kubermatic/pull/15356))
- Update oauth2-proxy to appversion v7.13.0. Potentially BREAKING Change: If your configuration relies on matching query parameters in `skip_auth_routes` patterns, you must update your regex patterns to match paths only. Review all `skip_auth_routes` entries for potential impact. For detailed information, migration guidance, and security implications, see the upstream [security advisory](https://github.com/oauth2-proxy/oauth2-proxy/security/advisories/GHSA-7rh7-c77v-6434) ([#15174](https://github.com/kubermatic/kubermatic/pull/15174))

### Supported Kubernetes Versions

- Add support for Kubernetes version 1.35 ([#15347](https://github.com/kubermatic/kubermatic/pull/15347))
- Drop support for Kubernetes 1.31 ([#15294](https://github.com/kubermatic/kubermatic/pull/15294))
- Add support for k8s patch releases v1.35.2/v1.34.5/v1.33.9/v1.32.13 ([#15538](https://github.com/kubermatic/kubermatic/pull/15538))
- Add support for the latest k8s patch releases v1.35.1/v1.34.4/v1.33.8/v1.32.12 ([#15465](https://github.com/kubermatic/kubermatic/pull/15465))
- Add support for the latest k8s patch releases v1.34.3/v1.33.7 ([#15239](https://github.com/kubermatic/kubermatic/pull/15239))
- Add support for k8s patch releases v1.34.2/v1.33.6/v1.32.10/v1.31.14 ([#15170](https://github.com/kubermatic/kubermatic/pull/15170))

### Cloud Providers

#### KubeVirt

- Support VolumeSnapshot creation for the KubeVirt cloud provider. The VolumeSnapshot feature is only supported in Namespaced Mode ([#15549](https://github.com/kubermatic/kubermatic/pull/15549))
- KubeVirt CSI driver snapshotting and disk resizing support; update KubeVirt provider RBAC to include: `snapshot.storage.k8s.io/volumesnapshots` (get, create, delete) and `persistentvolumeclaims` (get, list, watch) ([#15526](https://github.com/kubermatic/kubermatic/pull/15526))
- Add new environment variables for the KubeVirt provider in machine controller ([#15253](https://github.com/kubermatic/kubermatic/pull/15253))

#### OpenStack

- Add `loadBalancerFloatingIPPool` field to the OpenStack cloud spec to allow specifying a dedicated floating IP pool for LoadBalancer services ([#15508](https://github.com/kubermatic/kubermatic/pull/15508))
- Configure NodeVolumeAttachLimit for OpenStack at the Datacenter and Cluster level ([#15309](https://github.com/kubermatic/kubermatic/pull/15309))

### New Features

- Add `--provider-filter` flag to the mirror-images command to mirror only images for the desired provider ([#15314](https://github.com/kubermatic/kubermatic/pull/15314))
- Add Falco v0.41.3, v0.42.1, and v0.43.0 to the default application catalog ([#15548](https://github.com/kubermatic/kubermatic/pull/15548))
- Add a new Envoy Gateway application to the default catalog ([#15416](https://github.com/kubermatic/kubermatic/pull/15416))
- Add Gateway API support to the IAP chart, allowing IAP deployments to use HTTPRoute resources instead of Ingress when migrating to Gateway API ([#15365](https://github.com/kubermatic/kubermatic/pull/15365))
- Embed Gateway API CRDs in kubermatic-installer ([#15361](https://github.com/kubermatic/kubermatic/pull/15361))
- Konnectivity server and agent now default to `--xfr-channel-size=150`. This can be overridden via ComponentsOverride ([#15328](https://github.com/kubermatic/kubermatic/pull/15328))
- Introduce nftables proxy mode ([#15321](https://github.com/kubermatic/kubermatic/pull/15321))
- Kubermatic now supports Gateway API for external traffic routing as an alternative to NGINX Ingress. It can be enabled via the `--enable-gateway-api` operator flag (set via `migrateGatewayAPI: true` in Helm values and `--migrate-gateway-api` in `kubermatic-installer`) ([#15293](https://github.com/kubermatic/kubermatic/pull/15293))
- Konnectivity server metrics are now scraped and federated to seed Prometheus. New alerts notify administrators when Konnectivity agents are unavailable or experiencing high dial failure rates ([#15286](https://github.com/kubermatic/kubermatic/pull/15286))
- Replace Grafana Agent with Grafana Alloy in user clusters ([#15302](https://github.com/kubermatic/kubermatic/pull/15302))
- Add the ability to configure Helm registry connection options (TLS skip verification and PlainHTTP) for the default catalog and system applications in KubermaticConfiguration ([#15316](https://github.com/kubermatic/kubermatic/pull/15316))
- Add Envoy Agent in Component Settings ([#15306](https://github.com/kubermatic/kubermatic/pull/15306))
- The installer now uses a consistent scheme setup for both `deploy` and `local kind` commands ([#15288](https://github.com/kubermatic/kubermatic/pull/15288))
- Add Kubernetes MCP Server as a default application ([#15224](https://github.com/kubermatic/kubermatic/pull/15224))
- Add `EnableThrottling` option to KubermaticSettings to allow rate-limiting of dashboard notifications ([#15274](https://github.com/kubermatic/kubermatic/pull/15274))
- Add support for Kyverno enforcement at Datacenter, Seed, and Global levels. Administrators can now enforce Kyverno policy engine deployment on user clusters with configurable precedence (Datacenter > Seed > Global). When enforced, Kyverno is automatically enabled for clusters ([#15198](https://github.com/kubermatic/kubermatic/pull/15198))
- `kubermatic-installer`: `--helm-values` can now be specified multiple times to load multiple values files in the specified order (later files overwrite earlier ones). As is standard with Helm, lists and arrays are not merged but replaced by later files ([#15235](https://github.com/kubermatic/kubermatic/pull/15235))
- Set Tolerations overrides for all control plane components ([#15248](https://github.com/kubermatic/kubermatic/pull/15248))
- Set emulation for non-Linux local development environments ([#15245](https://github.com/kubermatic/kubermatic/pull/15245))
- Add status conditions for policy binding resources ([#15209](https://github.com/kubermatic/kubermatic/pull/15209))
- The image for KubeLB CCM can be overridden using `.spec.userCluster.kubelb` in the KubermaticConfiguration ([#15159](https://github.com/kubermatic/kubermatic/pull/15159))
- The application catalog management can now be handled by an external application-catalog-manager service as an opt-in feature. Enable the `ExternalApplicationCatalogManager` feature gate to use the new architecture ([#15333](https://github.com/kubermatic/kubermatic/pull/15333))
- Add HTTPRoute-Gateway sync controller to enable automatic certificate provisioning via cert-manager for KKP components ([#15497](https://github.com/kubermatic/kubermatic/pull/15497))
- Introduce audit logging enforcement for user clusters via a new controller in the seed-controller-manager. Depending on the datacenter configuration, audit logging is either enforced from the seed configuration or explicitly disabled for user clusters. Enforcement is skipped when the seed does not define audit logging or when a user cluster is annotated with `kubermatic.k8c.io/skip-audit-logging-enforcement: "true"` ([#15330](https://github.com/kubermatic/kubermatic/pull/15330))
- Users can now configure additional arguments for oauth2-proxy pods (useful for seed and user MLA) ([#15241](https://github.com/kubermatic/kubermatic/pull/15241))
- Default kube-proxy mode to nftables for Kubernetes clusters running version v1.35 and above ([#15537](https://github.com/kubermatic/kubermatic/pull/15537))
- Add global settings for the EventRateLimit admission plugin in KubermaticConfiguration. Admins can now enable EventRateLimit by default, enforce it for all clusters, and provide default configuration values via `spec.userCluster.admissionPlugins.eventRateLimit` ([#15300](https://github.com/kubermatic/kubermatic/pull/15300))
- MachineController deployment resources can now be customized via `cluster.spec.componentsOverride.machineController` ([#15202](https://github.com/kubermatic/kubermatic/pull/15202))
- Make local kind installation idempotent ([#15185](https://github.com/kubermatic/kubermatic/pull/15185))


### Bugfixes

- Fix idempotency of local kind installer ([#15541](https://github.com/kubermatic/kubermatic/pull/15541))
- EE: Upgraded Kyverno to v1.15.3 to address CVE-2026-22039 and regenerated user-cluster Kyverno CRDs ([#15540](https://github.com/kubermatic/kubermatic/pull/15540))
- Fix Alloy crashloop ([#15513](https://github.com/kubermatic/kubermatic/pull/15513))
- Fix alertmanager service port name reference after upstream chart migration ([#15512](https://github.com/kubermatic/kubermatic/pull/15512))
- Set `hostname` on Gateway while using cert-manager ([#15496](https://github.com/kubermatic/kubermatic/pull/15496))
- Add missing envoy-gateway-controller chart in release artifacts ([#15491](https://github.com/kubermatic/kubermatic/pull/15491))
- Add Seed-level `spec.nodeportProxy.envoy.connectionSettings` for nodeport-proxy Envoy idle timeout and TCP keepalive tuning. Existing clusters keep their current behavior by default; if fields are unset or `0`, Envoy defaults are used ([#15462](https://github.com/kubermatic/kubermatic/pull/15462))
- Add optional Seed setting `spec.nodeportProxy.envoy.replicas` to configure the `nodeport-proxy-envoy` replica count. If unset, the existing default behavior remains (`3` replicas) ([#15464](https://github.com/kubermatic/kubermatic/pull/15464))
- Add support for the `apps.kubermatic.k8c.io/reconciliation-interval` annotation on ApplicationDefinition to automatically set `spec.reconciliationInterval` on all ApplicationInstallations created from that definition ([#15355](https://github.com/kubermatic/kubermatic/pull/15355))
- Fix Helm 4 compatibility ([#15357](https://github.com/kubermatic/kubermatic/pull/15357))
- Fix undeleted etcd-launcher ServiceAccount in kube-system ([#15303](https://github.com/kubermatic/kubermatic/pull/15303))
- Add validation to prevent user creation with duplicate email addresses ([#15218](https://github.com/kubermatic/kubermatic/pull/15218))
- Fix local kind command ignoring global flags ([#15276](https://github.com/kubermatic/kubermatic/pull/15276))
- Fix dashboard-metrics-scraper security context with `runAsNonRoot` and dropped `capabilities` for improved CIS compliance ([#15275](https://github.com/kubermatic/kubermatic/pull/15275))
- Upgrade Cortex to 1.16.1, fixing an issue where cortex-ingester was consuming excessive storage space ([#15242](https://github.com/kubermatic/kubermatic/pull/15242))
- Velero backup hook annotations have been corrected to use proper JSON format and ASCII quotes, fixing backup failures caused by invalid exec commands ([#15217](https://github.com/kubermatic/kubermatic/pull/15217))
- Delete orphaned UserProjectBinding resources on User or Project deletion ([#15181](https://github.com/kubermatic/kubermatic/pull/15181))
- Fix Operating System Manager args for flags like `containerd-registry-mirrors` ([#15154](https://github.com/kubermatic/kubermatic/pull/15154))
- Add `omitempty` to component settings fields to allow partial configuration ([#15182](https://github.com/kubermatic/kubermatic/pull/15182))
- Add HTTP/2 keepalive functionality enabling Envoy-Agent to detect broken or half-open TCP connections to the nodeport-proxy and re-establish them automatically ([#15062](https://github.com/kubermatic/kubermatic/pull/15062))
- Fix azurefile-csi with Kubernetes 1.31 and 1.32 ([#15162](https://github.com/kubermatic/kubermatic/pull/15162))
- Fix policy template selector targeting with empty target selectors ([#15145](https://github.com/kubermatic/kubermatic/pull/15145))

### Updates

- Update KubeVirt application catalog entry to v1.7.1, fixing VM creation failures introduced in v1.1.0 ([#15561](https://github.com/kubermatic/kubermatic/pull/15561))
- Update controller-runtime to 0.23.1 and controller-tools to 0.20.1 ([#15490](https://github.com/kubermatic/kubermatic/pull/15490))
- Update Velero to version 1.17.1 ([#15457](https://github.com/kubermatic/kubermatic/pull/15457))
- Update KubeLB to v1.3.1 ([#15477](https://github.com/kubermatic/kubermatic/pull/15477))
- Update Konnectivity proxy versioning to align with Kubernetes minor versions. Kubernetes 1.34+ clusters will receive v0.34.0 until a corresponding upstream version exists ([#15408](https://github.com/kubermatic/kubermatic/pull/15408))
- Update Helm to v3.19.0 ([#15203](https://github.com/kubermatic/kubermatic/pull/15203))
- Update nginx-ingress-controller version to 1.14.3 ([#15364](https://github.com/kubermatic/kubermatic/pull/15364))
- Update `kubermatic-installer local kind` to deploy Gateway API instead of NGINX Ingress ([#15348](https://github.com/kubermatic/kubermatic/pull/15348))
- Update Envoy to version 1.37.0 ([#15351](https://github.com/kubermatic/kubermatic/pull/15351))
- Update metering to v1.3.1 ([#15353](https://github.com/kubermatic/kubermatic/pull/15353))
- Update Go version to 1.25.6 ([#15326](https://github.com/kubermatic/kubermatic/pull/15326))
- Update base image for kubermatic container image to alpine:3.23 ([#15312](https://github.com/kubermatic/kubermatic/pull/15312))
- Update Canal to v3.31 ([#15304](https://github.com/kubermatic/kubermatic/pull/15304))
- Update azuredisk-csi-driver to 1.32.11 for Kubernetes 1.32 and to 1.31.12 for Kubernetes 1.31 ([#15147](https://github.com/kubermatic/kubermatic/pull/15147))
- Add Cilium 1.18.6 and 1.17.12 ([#15305](https://github.com/kubermatic/kubermatic/pull/15305))


### Cleanups

- Migrate from the deprecated Kubernetes Endpoints API to EndpointSlices in preparation for Endpoints removal in future Kubernetes versions ([#15344](https://github.com/kubermatic/kubermatic/pull/15344))
- Remove unused `external-admin-user` ServiceAccount and ClusterRoleBinding from user clusters ([#15280](https://github.com/kubermatic/kubermatic/pull/15280))
- Remove cluster-autoscaler addon ([#15311](https://github.com/kubermatic/kubermatic/pull/15311))

### Dashboard and API

#### Cloud Providers

##### KubeVirt

- Add a new machine deployment label option to KubeVirt machine deployments to manage labels on KubeVirt VMs ([#7866](https://github.com/kubermatic/dashboard/pull/7866))
- Add advanced machine type selector for KubeVirt with GPU/CPU categorization and search capabilities ([#7797](https://github.com/kubermatic/dashboard/pull/7797))
- Filter KubeVirt OSP list based on the selected OS image version ([#7747](https://github.com/kubermatic/dashboard/pull/7747))
- Add support for listing namespaced KubeVirt VirtualMachineInstancetype objects in the KubeVirt instance type list endpoint ([#7900](https://github.com/kubermatic/dashboard/pull/7900))

##### AWS

- Add advanced machine type selector for AWS with GPU filtering support ([#7771](https://github.com/kubermatic/dashboard/pull/7771))

##### Azure

- Introduce a tabbed, searchable table for Azure machines with categorized GPU support and optimized SKU-based data retrieval ([#7768](https://github.com/kubermatic/dashboard/pull/7768))

##### GCP

- Add advanced machine type selector for GCP with GPU (Accelerator) filtering support ([#7783](https://github.com/kubermatic/dashboard/pull/7783))

##### Hetzner

- Add advanced machine type selector for Hetzner with searchable table view and categorized instance types ([#7803](https://github.com/kubermatic/dashboard/pull/7803))

##### OpenStack

- Add advanced machine type selector with search and tabular display for OpenStack flavor selection ([#7802](https://github.com/kubermatic/dashboard/pull/7802))

##### vSphere

- Show a list of predefined category tags for vSphere when adding cluster tags ([#7721](https://github.com/kubermatic/dashboard/pull/7721))

##### VMware Cloud Director

- You can now define multiple Networks for nodes of your VMware Cloud Director machine deployment ([#7452](https://github.com/kubermatic/dashboard/pull/7452))
- Fix VMware Cloud Director cluster summary not displaying the Network and Additional Networks fields ([#7887](https://github.com/kubermatic/dashboard/pull/7887))

#### New Features

- Add `nftables` to the available kube-proxy mode options ([#7861](https://github.com/kubermatic/dashboard/pull/7861))
- Add branding and whitelabeling configuration support for the Kubermatic Dashboard ([#7848](https://github.com/kubermatic/dashboard/pull/7848))
- Support all four event rate limit types ([#7796](https://github.com/kubermatic/dashboard/pull/7796))
- Add a Cluster ID tooltip to cluster and project overview pages ([#7693](https://github.com/kubermatic/dashboard/pull/7693))
- Add cluster name to snapshot list and details ([#7695](https://github.com/kubermatic/dashboard/pull/7695))

#### Bugfixes

- Set nftables as the default kube-proxy mode for Kubernetes 1.35+ clusters when using non-Cilium CNI plugins ([#7906](https://github.com/kubermatic/dashboard/pull/7906))
- Fix number-stepper input to hide native spinners in Firefox for a consistent UI across all browsers ([#7905](https://github.com/kubermatic/dashboard/pull/7905))
- REST API tokens can now be properly authenticated through the Swagger UI without being overwritten by session tokens ([#7875](https://github.com/kubermatic/dashboard/pull/7875))
- Fix styling issues with cards, tables, and button toggles in light and dark themes ([#7774](https://github.com/kubermatic/dashboard/pull/7774))
- Fix kubeconfig download with non-NGINX ingresses ([#7800](https://github.com/kubermatic/dashboard/pull/7800))
- Fix issue where OIDC kubeconfig downloads would fail with RBAC "Forbidden" errors when the identity provider returns uppercase email addresses ([#7740](https://github.com/kubermatic/dashboard/pull/7740))
- Fix encryption at rest feature failing in environments with separate master and seed clusters ([#7718](https://github.com/kubermatic/dashboard/pull/7718))
- Respect datacenter selectors for default and enforced apps; prevent duplicate app additions when switching datacenters; fix loading of enforced apps in the edit/customize cluster template dialog ([#7679](https://github.com/kubermatic/dashboard/pull/7679))
- Fix a bug where user cluster logging/monitoring checkboxes were shown even though user cluster MLA was disabled in the seed settings ([#7681](https://github.com/kubermatic/dashboard/pull/7681))
- Fix cluster summary not displaying provider details when using presets ([#7673](https://github.com/kubermatic/dashboard/pull/7673))
- Fix a regression bug which introduced errors when a user tried to log in with an email address containing uppercase letters when the lowercase version was already stored ([#7671](https://github.com/kubermatic/dashboard/pull/7671))

#### Updates

- Update Go version to 1.25.7 ([#7886](https://github.com/kubermatic/dashboard/pull/7886))
- Update frontend dependencies to patch known security vulnerabilities ([#7809](https://github.com/kubermatic/dashboard/pull/7809))
- Update resource quota calculations to use binary units (base-1024) ([#7729](https://github.com/kubermatic/dashboard/pull/7729))
- Update base image for dashboard container image to alpine:3.23 ([#7777](https://github.com/kubermatic/dashboard/pull/7777))
- Update web-terminal image to v0.12.0 ([#7777](https://github.com/kubermatic/dashboard/pull/7777))
- Update to Angular 20 with Material Design CSS prefix migration from `--mdc-*` to `--mat-*` ([#7601](https://github.com/kubermatic/dashboard/pull/7601))
- Update to controller-runtime v0.22.0 and client libraries to v0.34.0 ([#7692](https://github.com/kubermatic/dashboard/pull/7692))

#### Cleanups

- Migrate Angular templates to new control flow syntax and self-closing tags for code consistency and compliance with modern Angular standards ([#7675](https://github.com/kubermatic/dashboard/pull/7675))
- Remove beta labels from Kyverno Policies UI elements ([#7746](https://github.com/kubermatic/dashboard/pull/7746))
- The Anexia provider is now deprecated. A warning is shown in the application to inform users ([#7767](https://github.com/kubermatic/dashboard/pull/7767))
- Add deprecation warnings for the Kubernetes Dashboard feature, as the upstream project is no longer actively maintained ([#7810](https://github.com/kubermatic/dashboard/pull/7810))
