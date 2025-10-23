# Kubermatic 2.29

- [v2.29.0](#v2290)

## v2.29.0

**GitHub release: [v2.29.0](https://github.com/kubermatic/kubermatic/releases/tag/v2.29.0)**

### Breaking Changes

- Bump cert-manager to v1.17.4 ([#14853](https://github.com/kubermatic/kubermatic/pull/14853))
	-  Cert-manager now hashes large RSA keys (3072 & 4096bit) with SHA-384 or SHA-512 respectively. If you are using these key sizes in your certificates, make sure your environment can handle the aforementioned hashing algorithms
	- Log messages that were not structured have now been replaced with structured logs. If you were matching on specific log strings, this could break your setup.

- Update default Cilium version to 1.18.2 ([#15095](https://github.com/kubermatic/kubermatic/pull/15095))
	- Cilium 1.18 will fail to start on Ubuntu 22.04 nodes using kernel 5.15.0-47-generic due to missing BPF verifier fixes. 
Upgrading to a newer kernel (either enabling "Upgrade system on first boot" from KKP UI, or using a newer kernel like 5.15.0-160), or using Ubuntu 24.04 will resolve the issue.

### Supported Kubernetes Version

- Add support for Kubernetes version 1.34 ([#14940](https://github.com/kubermatic/kubermatic/pull/14940))
- Remove support for Kubernetes version 1.30 ([#14828](https://github.com/kubermatic/kubermatic/pull/14828))
- Add support for k8s patch releases 1.33.5/1.33.4/1.33.3/1.33.2/1.32.9/1.32.8/1.32.7/1.32.6/1.31.13/1.31.12/1.31.11/1.31.10 ([#14998](https://github.com/kubermatic/kubermatic/pull/14998), [#14910](https://github.com/kubermatic/kubermatic/pull/14910), [#14830](https://github.com/kubermatic/kubermatic/pull/14830), [#14783](https://github.com/kubermatic/kubermatic/pull/14783))

#### Supported Versions

- 1.34.1
- 1.33.5
- 1.33.3
- 1.33.2
- 1.32.9
- 1.32.7
- 1.32.6
- 1.31.13
- 1.31.11
- 1.31.10 

### Cloud Providers

#### KubeVirt

- A bug was fixed where evicted KubeVirt VMs configured with evictionStrategy `LiveMigrate` were treated like VMs with `External` evictionStrategy by deleting the related machine object ([#14736](https://github.com/kubermatic/kubermatic/pull/14736))
- A bug regarding network policy cleanup up in KubeVirt infra clusters when the removal of the finalizer failed after deleting the network policy was fixed ([#14802](https://github.com/kubermatic/kubermatic/pull/14802))
- Support KubeVirt vCPUs validation in the resource quota controller ([#14728](https://github.com/kubermatic/kubermatic/pull/14728))

#### OpenStack 

- Add Load Balancer Class support for OpenStack cloud provider on cluster level ([#15046](https://github.com/kubermatic/kubermatic/pull/15046))
- Support IPv4 and IPv6 custom subnet for Openstack provider ([#15080](https://github.com/kubermatic/kubermatic/pull/15080))
- Add the ability to skip router reconciliation in the OpenStack provider ([#14771](https://github.com/kubermatic/kubermatic/pull/14771))
- Fix adding router-link OpenStack finalizer in the wrong place ([#15086](https://github.com/kubermatic/kubermatic/pull/15086))

#### GCP
- Fix Load Balancer assignment in Kubernetes 1.33 and 1.34 GCP clusters.([#15123](https://github.com/kubermatic/kubermatic/pull/15123))

### New Features
- Improve `PolicyBinding` resources cleanup([#15110](https://github.com/kubermatic/kubermatic/pull/15110))
- The newly introduced external application catalog manager was added to `kubermatic-installer mirror-images` command to be respected in offline environments and fetching catalog apps from an OCI image when the external manager is enabled was introduced for that purpose. ([#14995](https://github.com/kubermatic/kubermatic/pull/14995))
- Add [Kueue](https://github.com/kubernetes-sigs/kueue) to the default applications catalog ([#15004](https://github.com/kubermatic/kubermatic/pull/15004))
- Non root device usage on worker nodes can now be enabled for containerd runtime by setting seed datacenter value `spec.datacenter.node.enableNonRootDeviceOwnership` to `true` ([#14891](https://github.com/kubermatic/kubermatic/pull/14891))
- The KubeLB tenant spec can now be defaulted at project level under `.spec.defaultTenantSpec` for KKP user cluster. For further details regarding this configuration, please take a look at [KubeLB tenant docs](https://docs.kubermatic.com/kubelb/v1.1/references/ee/#tenantspec) ([#14819](https://github.com/kubermatic/kubermatic/pull/14819))
- Add the ability to configure `kube-state-metrics` in the KKP user clusters ([#14829](https://github.com/kubermatic/kubermatic/pull/14829))
- Promtail is replaced by Grafana Alloy as the log shipper in the KKP seed clusters ([#14767](https://github.com/kubermatic/kubermatic/pull/14767))
- Add an option to restrict project modification to the admins ([#14843](https://github.com/kubermatic/kubermatic/pull/14843))
- Overwrite system application images when `overwriteRegistry` is defined ([#14773](https://github.com/kubermatic/kubermatic/pull/14773))
- KubeLB: KKP defaulting will now enable KubeLB for a cluster if it's enforced at the datacenter level ([#14732](https://github.com/kubermatic/kubermatic/pull/14732))
- Allow setting registry settings of container-runtime deployed user cluster through Cluster CR ([#14745](https://github.com/kubermatic/kubermatic/pull/14745))
- Enable DynamicResourceAllocation (DRA) for user clusters ([#14872](https://github.com/kubermatic/kubermatic/pull/14872))
- You can now use annotations and labels on user clusters to enable templating during application installations. This allows for dynamic configuration using expressions like {{- if eq (index .Cluster.Annotations "env") "dev" }}custom1{{ else }}custom2{{ end }}. This feature is useful for more flexible multi-environment setups, for example ([#14877](https://github.com/kubermatic/kubermatic/pull/14877))

### Bugfixes

- Fix invalid `PolicyTemplate` resources that set both `spec.enforced` and `spec.namespacedPolicy` ([#15110](https://github.com/kubermatic/kubermatic/pull/15110))
- Fix the default policy catalog `--deploy-default-policy-template-catalog` flag timing out in the kubermatic-installer ([#15099](https://github.com/kubermatic/kubermatic/pull/15099))
- [User Cluster MLA] Minor upgrade of Cortex to fix repeating errors in the logs ([#14944](https://github.com/kubermatic/kubermatic/pull/14944))
- The daemonset "node-local-dns" in the KKP user clusters now correctly defines port 9253 as the metrics port ([#14926](https://github.com/kubermatic/kubermatic/pull/14926))
- A caching functionality for used http.Transports when initializing MinIO clients in the seed-controller-manager is added to avoid TCP connection leaks ([#14927](https://github.com/kubermatic/kubermatic/pull/14927), [#14848](https://github.com/kubermatic/kubermatic/pull/14848))
- Fix issue with CBSL credentials and status not syncing to seed clusters ([#14703](https://github.com/kubermatic/kubermatic/pull/14703))
- Add RBAC rules for Velero Backup resources to allow get, list, and watch operations ([#14822](https://github.com/kubermatic/kubermatic/pull/14822))
- Fix log spam on deleted ResourceQuota objects ([#14714](https://github.com/kubermatic/kubermatic/pull/14714))
- Fix a regression bug regarding node-exporter pod labeling which didn't exclude node-exporter pods from pod discovery ([#14740](https://github.com/kubermatic/kubermatic/pull/14740))
- Add Velero post-backup hook to clean up /backup/* files after Prometheus backup completion to prevent disk space accumulation on the node where Prometheus is running ([#14708](https://github.com/kubermatic/kubermatic/pull/14708))
- A bug which lead to missing kube state metrics scraping was fixed ([#14759](https://github.com/kubermatic/kubermatic/pull/14759))
- Add the ETCDCTL_ENDPOINTS environment variable with name-based endpoints in all etcd pods. This enables successful execution of the `etcdctl endpoint health` command without the need for the `--cluster` flag which pulls IP based endpoints from the etcd ring ([#14724](https://github.com/kubermatic/kubermatic/pull/14724))
- Mirror the WebTerminal image ([#15108](https://github.com/kubermatic/kubermatic/pull/15108))

### Updates
- Update OpenStack CSI version to 1.34.0 ([#15115](https://github.com/kubermatic/kubermatic/pull/15115))
- Bump KubeVirt CSI Driver Operator to v0.4.5 ([#15096](https://github.com/kubermatic/kubermatic/pull/15096))
- Add Cilium 1.17.7 and 1.18.2 as supported CNI version, deprecate cilium version 1.14.16 as it's impacted by CVEs ([#15095](https://github.com/kubermatic/kubermatic/pull/15095),  [#15065](https://github.com/kubermatic/kubermatic/pull/15065), [#15048](https://github.com/kubermatic/kubermatic/pull/15048))
- Update default Canal version to v3.30.3 and deprecate v3.27 ([#15078](https://github.com/kubermatic/kubermatic/pull/15078))
- Update machine-controller version to [v1.63.1](https://github.com/kubermatic/machine-controller/releases/tag/v1.63.1) ([#15047](https://github.com/kubermatic/kubermatic/pull/15047))
- Update operating-system-manager version to [v1.7.6](https://github.com/kubermatic/operating-system-manager/releases/tag/v1.7.6) ([#15047](https://github.com/kubermatic/kubermatic/pull/15047))
- Update nginx-ingress-controller version to 1.13.2 ([#15036](https://github.com/kubermatic/kubermatic/pull/15036))
- Update Dex chart to appversion 2.44.0 ([#15041](https://github.com/kubermatic/kubermatic/pull/15041))
- KubeLB CCM has been upgraded to v1.2.0 ([#14961](https://github.com/kubermatic/kubermatic/pull/14961))
- Update Prometheus federation configuration to include machine deployment metrics from user clusters in the seed MLA Prometheus ([#14817](https://github.com/kubermatic/kubermatic/pull/14817))
- Update helm to v3.17.4 ([#14831](https://github.com/kubermatic/kubermatic/pull/14831))
- Update the user cluster and metering Prometheus instances in the KKP Seed cluster to scrape `kubelet_volume_stats_capacity_bytes` and `kubelet_volume_stats_used_bytes` metrics from the KKP user clusters ([#14769](https://github.com/kubermatic/kubermatic/pull/14769))
- Update `kubermatic-installer local kind` Dex static client configurations ([#14735](https://github.com/kubermatic/kubermatic/pull/14735))
- Update Go version to 1.25.1 ([#14940](https://github.com/kubermatic/kubermatic/pull/14940))
- Replace Bitnami charts and images with kubermatic-mirror charts and images to address issues identified in bitnami/containers#83267 ([#14873](https://github.com/kubermatic/kubermatic/pull/14873))

### Cleanups

- Gateway API CRDs installation and management have been delegated to KubeLB, that natively manages these CRDs using "-install-gateway-api-crds" and "-gateway-api-crds-channel" flags ([#14919](https://github.com/kubermatic/kubermatic/pull/14919))
- Remove support for Equinix Metal (Packet) provider ([#14827](https://github.com/kubermatic/kubermatic/pull/14827))
- By default the oauth2-proxy disables Dex's approval screen now. To return to the old behaviour, set `approval_prompt = "force"` for each IAP deployment in your Helm values.yaml ([#14751](https://github.com/kubermatic/kubermatic/pull/14751))
- Early deprecation of unsupported Falco versions 0.35.1 and 0.37.0 from the default application catalog, since they are not compatible with modern Linux Kernel versions present in machine templates ([#14861](https://github.com/kubermatic/kubermatic/pull/14861))
- The deprecated field `defaultComponentSettings` in the Seed Resource has been removed ([#15102](https://github.com/kubermatic/kubermatic/pull/15102))

### Dashboard and API

#### Cloud Providers

##### GCP

- Fix disk types and machine types values are not loaded in cluster template for Google Cloud Provider ([#7639](https://github.com/kubermatic/dashboard/pull/7639))

##### OpenStack

- Add OpenStack LoadBalancer Class configuration support at the Cluster level ([#7646](https://github.com/kubermatic/dashboard/pull/7646))
- Add a new option to enable the config drive on the OpenStack provider for machine deployments, along with a datacenter level option to enforce it for all machine deployments ([#7516](https://github.com/kubermatic/dashboard/pull/7516))
- Add new option in the user-cluster to skip router reconciliation option for OpenStack provider ([#7483](https://github.com/kubermatic/dashboard/pull/7483))
- Fix network selection to display network ID when name is missing in OpenStack ([#7513](https://github.com/kubermatic/dashboard/pull/7513))

##### KubeVirt

- Skip setting custom CPUs field in machine deployment for KubeVirt user clusters ([#7493](https://github.com/kubermatic/dashboard/pull/7493))

#### New Features

- Add strict validation for cluster encryption keys requiring proper 32-byte base64 format ([#7653](https://github.com/kubermatic/dashboard/pull/7653))
- Add the overwrite-registry flag in the api server ([#7651](https://github.com/kubermatic/dashboard/pull/7651))
- Add support to toggle Encryption at Rest for Edit Cluster dialog ([#7599](https://github.com/kubermatic/dashboard/pull/7599))
- Enables Encryption at rest feature for secrets ([#7525](https://github.com/kubermatic/dashboard/pull/7525))
- Display node labels in the nodes list table ([#7588](https://github.com/kubermatic/dashboard/pull/7588))
- Add dialog to choose authentication type (KKP API or OIDC-kubelogin) when downloading or sharing kubeconfig via OIDC ([#7522](https://github.com/kubermatic/dashboard/pull/7522))
- Use cabundle as key for caching http.Transport ([#7591](https://github.com/kubermatic/dashboard/pull/7591))
- Make NVIDIA GPU Operator labels accessible through Dashboard API ([#7520](https://github.com/kubermatic/dashboard/pull/7520))
- Web-terminal: k9s, krew, and krew plugins ns, ctx, and oidc-login are available to use in the web-terminal image ([#7509](https://github.com/kubermatic/dashboard/pull/7509))
- Add a new button in the initial node step of the user cluster wizard to configure the Cluster Autoscaler application ([#7500](https://github.com/kubermatic/dashboard/pull/7500))
- Add support to delete presets from the admin interface with detailed linkage information ([#7479](https://github.com/kubermatic/dashboard/pull/7479))
- Add an option to restrict project modification to admins ([#7504](https://github.com/kubermatic/dashboard/pull/7504))
- Move web-terminal cleanup job to seed to fix cleanup not working when the token is expired ([#7451](https://github.com/kubermatic/dashboard/pull/7451))

#### Bugfixes
- Kyverno policy bindings disappear when the template selector no longer matches the cluster. Enforcing Kyverno Policy disables the Namespaced option.
 ([#7654](https://github.com/kubermatic/dashboard/pull/7654))
- Fix the Kyverno Policybinding in a multi-seed setup ([#7631](https://github.com/kubermatic/dashboard/pull/7631))
- Fix encryption configuration handling during cluster editing ([#7620](https://github.com/kubermatic/dashboard/pull/7620))
- Fix cluster template editing when autoscaler application is not present ([#7619](https://github.com/kubermatic/dashboard/pull/7619))
- Fix a possible null pointer exception for `isGlobalViewer` ([#7610](https://github.com/kubermatic/dashboard/pull/7610))
- Disable min/max options if cluster autoscaler is not available ([#7559](https://github.com/kubermatic/dashboard/pull/7559))
- Fix web terminal token expiration by refreshing expired tokens automatically ([#7508](https://github.com/kubermatic/dashboard/pull/7508))
- Project viewers can now only view cluster templates. Create, update, and delete actions are restricted except deletion by the owner ([#7446](https://github.com/kubermatic/dashboard/pull/7446))
- Fix validation error when switching expose strategy from Tunneling to LoadBalancer by clearing tunnelingAgentIP automatically ([#7422](https://github.com/kubermatic/dashboard/pull/7422))
- Fix KubeLB checkbox state management and UI flickering issues in cluster creation wizard/edit cluster dialog ([#7458](https://github.com/kubermatic/dashboard/pull/7458))
- KubeLB: Fix a bug where enforcement on a datacenter was not enabling KubeLB for the user clusters in the dashboard ([#7453](https://github.com/kubermatic/dashboard/pull/7453))

#### Updates

- Update KKP SDK to include `subnetAllocationPool` and `subnetCIDR` ([#7626](https://github.com/kubermatic/dashboard/pull/7626))
- Update Go version to v1.25.1 ([#7554](https://github.com/kubermatic/dashboard/pull/7554))
- Update Node version to 22 ([#7539](https://github.com/kubermatic/dashboard/pull/7539))
- Update web-terminal image to v0.11.0 ([#7509](https://github.com/kubermatic/dashboard/pull/7509))

#### Cleanups

- Remove deprecated Azure Basic Load Balancer SKU option, defaulting to Standard SKU ([#7590](https://github.com/kubermatic/dashboard/pull/7590))
- Remove Equinix (Packet) provider support from cluster creation, KubeOne, and presets ([#7533](https://github.com/kubermatic/dashboard/pull/7533))

