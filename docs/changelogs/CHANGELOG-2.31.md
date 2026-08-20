# Kubermatic 2.31

- [v2.31.0](#v2310)

## v2.31.0

**GitHub release: [v2.31.0](https://github.com/kubermatic/kubermatic/releases/tag/v2.31.0)**

### Breaking Changes

This release contains changes that require additional attention, please read the following items carefully.

- Envoy Gateway has been updated to v1.8.3 and the bundled Gateway API CRDs to v1.5.1 (adding TLSRoute and ListenerSet); the Envoy data plane now runs distroless-v1.38.3. action required: users with their own EnvoyProxy or SecurityPolicy resources should review the Envoy Gateway 1.7/1.8 breaking changes — `samplingFraction` now samples 100x more frequently, a `0s` SecurityPolicy timeout now means infinite instead of immediate, and OIDC is translated into a single native oauth2 filter ([#16214](https://github.com/kubermatic/kubermatic/pull/16214))
- Gateway API is now the enforced default for KKP ingress, the nginx-ingress-controller path has been removed. The `--migrate-gateway-api` and `--migrate-upstream-nginx-ingress`are now deprecated no-ops, users still relying on nginx-ingress would be migrated to Gateway API while upgrading. To remove the legacy LoadBalancer, re-run the installer with `--clean-nginx-lb` ([#15946](https://github.com/kubermatic/kubermatic/pull/15946))
- Deprecate Cilium versions are no longer offered as supported CNI versions. Existing clusters continue to work, but users should upgrade to Cilium 1.17.16 or 1.18.10. Very important: Clusters on Cilium 1.15.x must upgrade one minor at a time, for example 1.15.16 -> 1.16.9 -> 1.17.16 ([#15971](https://github.com/kubermatic/kubermatic/pull/15971))
- Along with #15861, this PR reduces amount of memory utilization by the prometheus component of each user-cluster. But if you had bumped up the prometheus memory requests via componentsOverride, you would not get benefit of memory optimizations. So please review actual memory usage of user-xxxx/prometheus pods and adjust componentsOverride, as needed. action required: If you had written custom alert rules / grafana dashboards on seed which uses metrics from job="nodes", you would need to update job name to kubelet instead. If you had written alerts / dashboards which use "instance" attribute, you will need to rewire your alerts / dahboards with pod attribute for some of the metrics ([#16141](https://github.com/kubermatic/kubermatic/pull/16141))
- This PR reduces amount of memory utilization by the prometheus component of each user-cluster. But if you had bumped up the prometheus memory requests via componentsOverride, you would not get benefit of memory optimizations. So please review actual memory usage of user-xxxx/prometheus pods and adjust componentsOverride, as needed. action required: If you had written custom alert rules / grafana dashboards on seed which uses metrics from job="nodes", you would need to update job name to kubelet instead ([#15861](https://github.com/kubermatic/kubermatic/pull/15861))
- Update Grafana Alloy from v1.9.2 to v1.17.0 (Helm chart 1.1.2 → 1.10.0). The current chart configuration is unaffected. If extending the helm chart with additional components, please review the https://github.com/grafana/alloy/blob/main/operations/helm/charts/alloy/CHANGELOG.md and https://grafana.com/docs/alloy/latest/release-notes/ for any upstream breaking changes ([#16027](https://github.com/kubermatic/kubermatic/pull/16027))
- KKP dashboard login now uses the OAuth authorization code flow with PKCE instead of the implicit flow, and all three OIDC flows (KKP dashboard login, Kubernetes dashboard login, and OIDC kubeconfig) are served by a single Dex client: "kubermaticIssuer". The separate "kubermatic" client ID, which existed only for the dashboard implicit flow, has been removed, and `KubermaticConfiguration.spec.auth.clientID` now defaults to "kubermaticIssuer". action required: if your identity provider (e.g. Dex) still references the "kubermatic" client ID, update it to "kubermaticIssuer" and make sure the client is configured for the authorization code flow with the correct redirect URIs before upgrading. See the [upgrade guide](https://docs.kubermatic.com/kubermatic/v2.31/installation/upgrading/upgrading-from-2.30-to-2.31/) for details ([#15860](https://github.com/kubermatic/kubermatic/pull/15860), [kubermatic/dashboard#8053](https://github.com/kubermatic/dashboard/pull/8053))

### New Features

- Add alpha project scoped KubeVirt accelerator quota accounting and enforcement. For activated projects, KKP aggregates immutable Machine footprints into ResourceQuota usage and readiness status, then enforces configured accelerator limits once accounting is ready and fresh. Existing unactivated projects and CPU, memory, and storage quota behavior remain unchanged ([#16265](https://github.com/kubermatic/kubermatic/pull/16265))
- Add per-datacenter default CPU/memory/disk size for KubeVirt worker nodes via nodeDefaults in the Seed's KubeVirt datacenter spec ([#16164](https://github.com/kubermatic/kubermatic/pull/16164))
- The MLA minio chart is updated to 5.4.0 ([#16125](https://github.com/kubermatic/kubermatic/pull/16125))
- By using the flag `kube-ovn-enabled` when running the command `kubermatic-installer local kind` kube-ovn will be deployed as the cni instead of kindnet. Also the kkp preset will be configured with the default vpc ([#14405](https://github.com/kubermatic/kubermatic/pull/14405))
- Add cis-bench run workflow to generate CIS benchmark report ([#16061](https://github.com/kubermatic/kubermatic/pull/16061))
- When you enable usercluster monitoring, automatically, kube-state-metrics and node-exporter Applications get installed in the cluster. If you disable the usercluster monitoring, same get removed as well. If the your existing user-cluster's use kube-state-metrics and node-exporter as legacy addons, nothing gets changed. If you delete old kube-state-metrics or node-exporter Addon, then they will get replaced with newer ApplicationInstaller as long user-cluster monitoring was enabled ([#15900](https://github.com/kubermatic/kubermatic/pull/15900))
- Add a workflow to publish cis benchmark report based on kkp and k8s version mentioned in the issue with label `cis-bench-request` ([#16043](https://github.com/kubermatic/kubermatic/pull/16043))
- Add `disabledAuditWebhookBackendDCs` setting to disable the Audit Webhook Backend option in the dashboard for specific datacenters ([#15804](https://github.com/kubermatic/kubermatic/pull/15804))
- The iap, minio, monitoring/grafana, telemetry, and kubermatic-operator Helm charts now support an `existingSecret` parameter to reference a pre-created Kubernetes Secret instead of generating one from values, enabling GitOps secret management without storing credentials in values.yaml ([#15979](https://github.com/kubermatic/kubermatic/pull/15979))
- Add allowVolumeExpansion and reclaimPolicy fields to the KubeVirtInfraStorageClass struct in the API, enabling explicit configuration of volume expansion and reclaim policy ([#15912](https://github.com/kubermatic/kubermatic/pull/15912))
- Makes host/zone anti-affinity configurable also for apiserver, controller-manager, scheduler, prometheus, usercluster-controller, operating-system-manager, CoreDNS, kube-state-metrics, machine-controller, konnectivity ([#15810](https://github.com/kubermatic/kubermatic/pull/15810))
- Add support for k8s' AuthenticationConfiguration as an alternative to configuring a single OIDC provider ([#15740](https://github.com/kubermatic/kubermatic/pull/15740))
- Reduces the Deployment/StatefulSet/DaemonSet revisionHistoryLimit of user cluster components to 2 to save etcd resources ([#15823](https://github.com/kubermatic/kubermatic/pull/15823))
- Add cluster-level resource configuration for Kyverno. Users can now configure resource requests and limits for the Kyverno admission, background, cleanup, and reports controllers ([#15736](https://github.com/kubermatic/kubermatic/pull/15736))
- Add Cilium 1.18.8 and 1.17.14 ([#15720](https://github.com/kubermatic/kubermatic/pull/15720))
- Set Canal default version to v3.31 ([#15721](https://github.com/kubermatic/kubermatic/pull/15721))
- Add new alerts providing insights into health of cortex used by user-cluster MLA ([#15630](https://github.com/kubermatic/kubermatic/pull/15630))
- Dex HTTPRoute path and pathType are now configurable via `httpRoute.path` and `httpRoute.pathType` values, allowing users to deploy Dex on a separate subdomain with root path instead of being limited to path-based routing ([#15627](https://github.com/kubermatic/kubermatic/pull/15627))
- Seed Grafana now has 12 new grafana dashboards under MLA Stack folder ([#15603](https://github.com/kubermatic/kubermatic/pull/15603))
- Envoy-gateway-controller: The envoyProxy image configuration now supports separate repository and tag fields for easier image mirroring. The legacy single-string format continues to work for backward compatibility ([#15595](https://github.com/kubermatic/kubermatic/pull/15595))

### API Changes

- Envoy Gateway has been updated to v1.8.3 and the bundled Gateway API CRDs to v1.5.1 (adding TLSRoute and ListenerSet); the Envoy data plane now runs distroless-v1.38.3. action required: users with their own EnvoyProxy or SecurityPolicy resources should review the Envoy Gateway 1.7/1.8 breaking changes — `samplingFraction` now samples 100x more frequently, a `0s` SecurityPolicy timeout now means infinite instead of immediate, and OIDC is translated into a single native oauth2 filter ([#16214](https://github.com/kubermatic/kubermatic/pull/16214))
- Add adminGroups in KubermaticSettings to automatically grant or revoke KKP admin privileges based on users base on OIDC groups ([#16165](https://github.com/kubermatic/kubermatic/pull/16165))
- Add Bring Your Own Gateway support for Gateway API mode. Operators can configure `spec.ingress.gateway.externalGateway` in the KubermaticConfiguration to make KKP attach its managed HTTPRoutes to an externally managed Gateway instead of creating and managing the default Gateway itself ([#15862](https://github.com/kubermatic/kubermatic/pull/15862))
- KKP core components (API, dashboard, webhook, master-controller-manager, seed-controller-manager) now support `nodeSelector`, `affinity`, `tolerations`, `topologySpreadConstraints`, and `priorityClassName` fields in `KubermaticConfiguration` to control pod scheduling ([#15828](https://github.com/kubermatic/kubermatic/pull/15828))
- Add support for configuring an existing TLS Secret for the operator-managed default Gateway via `spec.ingress.gateway.tls.secretRef`. Hostname-based Gateway listeners synced from watched `HTTPRoute`s now also work in manual TLS mode by reusing the configured certificate references. When using manual TLS, the provided certificate must cover all served hostnames, including MLA/IAP hostnames ([#15732](https://github.com/kubermatic/kubermatic/pull/15732))
- Add `spec.ingress.gateway.infrastructureAnnotations` to `KubermaticConfiguration` to configure `Gateway.spec.infrastructure.annotations` on the operator managed Gateway ([#15725](https://github.com/kubermatic/kubermatic/pull/15725))

### Bugfixes

- The Cluster Autoscaler application now grants the read access to resource.k8s.io that cluster-autoscaler 1.35 and newer require ([#16251](https://github.com/kubermatic/kubermatic/pull/16251))
- Kubermatic-installer now passes --force-conflicts to Helm 4 only for releases that Helm applies server-side, fixing deploy failures on releases that were originally installed with Helm 3 ([#16210](https://github.com/kubermatic/kubermatic/pull/16210))
- Fix a bug where multiple GroupProjectBindings with the same group and project could be created. The admission webhook now rejects duplicate bindings at creation time and prevents an existing binding from being updated into a conflicting group/project pair ([#16162](https://github.com/kubermatic/kubermatic/pull/16162))
- Fix UserProjectBindings being deleted before their User logs in for the first time ([#16131](https://github.com/kubermatic/kubermatic/pull/16131))
- Kubermatic-installer now enables server-side apply with conflict forcing when running with Helm 4, fixing `deploy` failures caused by field-ownership conflicts (for example on the dockercfg Secret). With Helm 4 the deprecated `--atomic` flag is replaced by `--rollback-on-failure` ([#16138](https://github.com/kubermatic/kubermatic/pull/16138))
- Fix nodeport-proxy-envoy Prometheus annotations to include the standard `prometheus.io/path` metrics path annotation ([#16092](https://github.com/kubermatic/kubermatic/pull/16092))
- Fix the kubevirt-network-controller emitting spurious "invalid NetworkPolicy" warning events and potentially panicking when reconciling cluster-isolation NetworkPolicies in default-deny mode before the cluster's apiserver address or DNS configuration were available ([#16074](https://github.com/kubermatic/kubermatic/pull/16074))
- Fix ee resource quota validation for KubeVirt MachineDeployments using user-deployed namespaced VirtualMachineInstancetype resources (e.g. custom GPU instancetypes). Previously, machine creation was incorrectly rejected with "instancetype not found" when resource quotas were configured ([#15958](https://github.com/kubermatic/kubermatic/pull/15958))
- Fix seed Prometheus scraping envoy-agent directly via worker private IPs for tunneling user clusters ([#16024](https://github.com/kubermatic/kubermatic/pull/16024))
- Fix apiserver OIDC issuer NetworkPolicies for OIDC issuers exposed through selector backed seed side LoadBalancer Services, like KubeLB, when the CNI enforces egress against translated backend pod identities ([#16014](https://github.com/kubermatic/kubermatic/pull/16014))
- KKP now configures Cilium to exclude the reserved (KKP) NodeLocalDNS address from local address detection when NodeLocalDNS is enabled. This fixes DNS access to NodeLocalDNS for Cilium clusters with restrictive egress NetworkPolicies, for example Web Terminal sessions with internet access disabled. Existing clusters require a restart of the Cilium DaemonSet for the new startup configuration to take effect if needed. Admins can either restart it manually or set Cilium's `rollOutCiliumPods=true` Helm value, this will roll the agents automatically on configmap changes ([#15996](https://github.com/kubermatic/kubermatic/pull/15996))
- Fix MLA cleanup silently leaving Grafana users, organizations, datasources, and dashboards behind when Grafana returned an error during deletion; cleanup now verifies the deletion and retries on failure ([#15980](https://github.com/kubermatic/kubermatic/pull/15980))
- Mirror-images now ignores configured dockerTagSuffix values together with repository overrides when --ignore-repository-overrides is set, so reused offline configurations resolve upstream images instead of failing with MANIFEST_UNKNOWN ([#15967](https://github.com/kubermatic/kubermatic/pull/15967))
- Fix a kkp-master-operator issue where the /nvidia-gpu-operator ApplicationDefinition would fail with a context deadline exceeded error during KKP upgrades ([#15960](https://github.com/kubermatic/kubermatic/pull/15960))
- Fix recovery for Helm-based ApplicationInstallations whose Helm release is stuck in a pending state or whose retry state no longer matches the deployed Helm release ([#15892](https://github.com/kubermatic/kubermatic/pull/15892))
- BYO Gateway migrations now wait for the external Gateway and KKP-managed HTTPRoutes to be accepted before completing Gateway cleanup ([#15896](https://github.com/kubermatic/kubermatic/pull/15896))
- SSH keys from machine deployment providerSpec are no longer removed from worker nodes by the user-ssh-key-agent ([#15863](https://github.com/kubermatic/kubermatic/pull/15863))
- Remove the creation of cluster scope resources from KubeVirt provider and offload that functionality to the platform admin. Needed permissions to be created in the cluster: PersistentVolumes: "get", "list", "watch" ([#15830](https://github.com/kubermatic/kubermatic/pull/15830))
- System ApplicationDefinitions now receive upstream `defaultValuesBlock` changes during KKP upgrades. Admin customizations are preserved via hash-based detection ([#15691](https://github.com/kubermatic/kubermatic/pull/15691))
- Fix KubeVirt CSI RBAC permissions by adding the missing patch and update verbs for persistentvolumeclaims, and introducing a ClusterRole and ClusterRoleBinding for persistentvolumes with get, list, and watch permissions ([#15602](https://github.com/kubermatic/kubermatic/pull/15602))
- Fix seed-controller-manager cache sync timeout issue on large kkp instance clusters ([#15722](https://github.com/kubermatic/kubermatic/pull/15722))
- Gateway and HTTPRoute resources are now properly owned by KubermaticConfiguration and will be garbage collected on deletion. User-added labels and annotations on these resources are no longer overwritten during reconciliation ([#15642](https://github.com/kubermatic/kubermatic/pull/15642))
- Fix Gateway API listener churn where kubermatic-operator would cyclically remove and re-add dynamic listeners during reconciliation. Dynamic listeners added by httproute-gateway-sync controller are now preserved ([#15628](https://github.com/kubermatic/kubermatic/pull/15628))
- Fix the datasource error on the `Kubermatic/Controller Manager` and `Kubermatic/Controller-Runtime Metrics` Grafana dashboards ([#14857](https://github.com/kubermatic/kubermatic/pull/14857))
- Add missing condition to skip MLA Secrets deployment ([#15659](https://github.com/kubermatic/kubermatic/pull/15659))
- Mirror the missing cluster-autoscaler images ([#15651](https://github.com/kubermatic/kubermatic/pull/15651))
- Fix ineffective anti-affinity for the seed nodeport-proxy-envoy Deployment by aligning its anti-affinity selector with the pod labels actually used by the Deployment ([#15601](https://github.com/kubermatic/kubermatic/pull/15601))
- Fix kubermatic-installer attempting to mirror a non-existent kubectl image from docker.io/bitnamilegacy for v1.35 ([#15576](https://github.com/kubermatic/kubermatic/pull/15576))

### Cleanups

- Gateway API is now the enforced default for KKP ingress, the nginx-ingress-controller path has been removed. The `--migrate-gateway-api` and `--migrate-upstream-nginx-ingress`are now deprecated no-ops, users still relying on nginx-ingress would be migrated to Gateway API while upgrading. To remove the legacy LoadBalancer, re-run the installer with `--clean-nginx-lb` ([#15946](https://github.com/kubermatic/kubermatic/pull/15946))
- Deprecate Cilium versions are no longer offered as supported CNI versions. Existing clusters continue to work, but users should upgrade to Cilium 1.17.16 or 1.18.10. Very important: Clusters on Cilium 1.15.x must upgrade one minor at a time, for example 1.15.16 -> 1.16.9 -> 1.17.16 ([#15971](https://github.com/kubermatic/kubermatic/pull/15971))
- Drop support for k8s v1.32 ([#15832](https://github.com/kubermatic/kubermatic/pull/15832))
- The `Project.spec.defaultTenantSpec` field is now schemaless and preserves unknown fields. Existing values are forward-compatible. Refer to the KubeLB `TenantSpec` reference (https://docs.kubermatic.com/kubelb/latest/references/ee/#tenantspec) for details ([#15848](https://github.com/kubermatic/kubermatic/pull/15848))

### Design


### Miscellaneous

- The metering version is upgraded to v1.4.1, adding persistent storage usage to the JSON cluster report ([#16268](https://github.com/kubermatic/kubermatic/pull/16268))
- Add a new `subnets` field (array) to the KubeVirt preset, allowing a preset to offer multiple subnet choices. The existing `subnetName` field is deprecated in favor of `subnets` ([#16223](https://github.com/kubermatic/kubermatic/pull/16223))
- Add a accelerator quota field to the ResourceQuota API ([#16211](https://github.com/kubermatic/kubermatic/pull/16211))
- Add support for Kubernetes v1.36 ([#15986](https://github.com/kubermatic/kubermatic/pull/15986))
- Add improvements for the handling of Kyverno PolicyBindings and generated Kyverno resources when PolicyTemplates are deleted, Kyverno is disabled, or clusters are deleted ([#16034](https://github.com/kubermatic/kubermatic/pull/16034))
- Add enableImageDiscovery option to OpenStack settings for listing project-scoped images in the dashboard ([#16082](https://github.com/kubermatic/kubermatic/pull/16082))
- Add support for k8s patch releases 1.35.6/1.34.9/1.33.13 ([#15992](https://github.com/kubermatic/kubermatic/pull/15992))
- Kubermatic-operator now reconciles Gateway API resources before Deployments, preventing missing ConfigMaps from blocking Gateway creation ([#15712](https://github.com/kubermatic/kubermatic/pull/15712))
- By default the metallb app uses frr 10.4.1 now which fixes a bug that affected installations on AWS ([#15615](https://github.com/kubermatic/kubermatic/pull/15615))
- The label key used for network policies for kubevirt virtual machines changed from `cluster.x-k8s.io/cluster-name` to `kubermatic.k8c.io/cluster-id` ([#15606](https://github.com/kubermatic/kubermatic/pull/15606))
- Canal v3.30 and v3.31 now pull Calico images from quay.io instead of docker.io to avoid Docker Hub rate limits that could block cluster bootstrap ([#15620](https://github.com/kubermatic/kubermatic/pull/15620))
- Introduced a new `--separate-seed` flag to the `deploy seed` command, enabling the deployment of either the Ingress Controller or the Envoy Gateway when operating with a separate seed setup ([#15578](https://github.com/kubermatic/kubermatic/pull/15578))

### Chores

- Add NVIDIA GPU Operator v26.3.3 to the application catalog ([#16281](https://github.com/kubermatic/kubermatic/pull/16281))
- Add support for k8s v1.36.3 and updated the default Kubernetes version to v1.35.7 ([#16237](https://github.com/kubermatic/kubermatic/pull/16237))
- Along with #15861, this PR reduces amount of memory utilization by the prometheus component of each user-cluster. But if you had bumped up the prometheus memory requests via componentsOverride, you would not get benefit of memory optimizations. So please review actual memory usage of user-xxxx/prometheus pods and adjust componentsOverride, as needed. action required: If you had written custom alert rules / grafana dashboards on seed which uses metrics from job="nodes", you would need to update job name to kubelet instead. If you had written alerts / dashboards which use "instance" attribute, you will need to rewire your alerts / dahboards with pod attribute for some of the metrics ([#16141](https://github.com/kubermatic/kubermatic/pull/16141))
- The metering version is upgraded to v1.4.0 ([#16199](https://github.com/kubermatic/kubermatic/pull/16199))
- Add support for k8s patch releases v1.35.7/v1.34.10 ([#16166](https://github.com/kubermatic/kubermatic/pull/16166))
- This PR reduces amount of memory utilization by the prometheus component of each user-cluster. But if you had bumped up the prometheus memory requests via componentsOverride, you would not get benefit of memory optimizations. So please review actual memory usage of user-xxxx/prometheus pods and adjust componentsOverride, as needed. action required: If you had written custom alert rules / grafana dashboards on seed which uses metrics from job="nodes", you would need to update job name to kubelet instead ([#15861](https://github.com/kubermatic/kubermatic/pull/15861))
- The cert-manager chart shipped with KKP is bumped to v1.20.3 ([#16124](https://github.com/kubermatic/kubermatic/pull/16124))
- Remove opened from issue type for cis bench workflow ([#16080](https://github.com/kubermatic/kubermatic/pull/16080))
- Add support for Cilium 1.19.4 and made it the default Cilium version for new user clusters ([#15976](https://github.com/kubermatic/kubermatic/pull/15976))
- Skip deployment for github actions workflow ([#16077](https://github.com/kubermatic/kubermatic/pull/16077))
- Add issue link in the comment in case of CIS failures ([#16072](https://github.com/kubermatic/kubermatic/pull/16072))
- Add issue template for cis benchmark request ([#16067](https://github.com/kubermatic/kubermatic/pull/16067))
- Add ApplicationInstallation Prometheus metrics ([#15937](https://github.com/kubermatic/kubermatic/pull/15937))
- The Seed MLA Loki chart has been upgraded to version v7.0.0 ([#15879](https://github.com/kubermatic/kubermatic/pull/15879))
- User-cluster MLA grafana is upgraded to latest available version (v13.0.1) ([#15906](https://github.com/kubermatic/kubermatic/pull/15906))
- The kubermatic-installer supports Helm 4 now in addition to Helm 3 ([#15902](https://github.com/kubermatic/kubermatic/pull/15902))
- Add support of k8s patch releases v1.35.5/v1.34.8/v1.33.12 ([#15869](https://github.com/kubermatic/kubermatic/pull/15869))
- Add support for k8s patch releases v1.35.4/v1.34.7/v1.33.11 ([#15748](https://github.com/kubermatic/kubermatic/pull/15748))
- Add support for k8s patch releases v1.35.3/v1.34.6/v1.33.10 ([#15679](https://github.com/kubermatic/kubermatic/pull/15679))

### Updates

- Update machine-controller to v1.66.2 ([#16279](https://github.com/kubermatic/kubermatic/pull/16279))
- Update operating-system-manager to v1.11.3 ([#16282](https://github.com/kubermatic/kubermatic/pull/16282))
- Update the Cluster Autoscaler Helm chart to 9.59.0 ([#16253](https://github.com/kubermatic/kubermatic/pull/16253))
- Update util image version to 2.9.0 ([#16258](https://github.com/kubermatic/kubermatic/pull/16258))
- Update OSM version to [v1.11.1](https://github.com/kubermatic/operating-system-manager/releases/tag/v1.11.1) ([#16245](https://github.com/kubermatic/kubermatic/pull/16245))
- Update operating-system-manager to v1.11.0 ([#16237](https://github.com/kubermatic/kubermatic/pull/16237))
- Update Go version to v1.26.5 to include upstream security fixes ([#16185](https://github.com/kubermatic/kubermatic/pull/16185))
- Update the Azure cloud-controller-manager, cloud-node-manager, and the Azure Disk and Azure File CSI drivers to their latest upstream versions per supported Kubernetes minor. The controller-manager and node-manager images now come from the maintained mcr.microsoft.com/oss/v2 registry path (the previous /oss path was frozen at v1.34.3), clearing the base-image CVEs those stale images carried. Kubernetes 1.35 clusters now receive matching 1.35 controller-manager and node-manager images instead of 1.34 ones ([#16195](https://github.com/kubermatic/kubermatic/pull/16195))
- Update the d3fk/s3cmd image to a current digest built on Alpine 3.24, resolving 62 CVEs inherited from the previously used end-of-life Alpine 3.17 base ([#16194](https://github.com/kubermatic/kubermatic/pull/16194))
- Update machine-controller to [v1.66.1](https://github.com/kubermatic/machine-controller/releases/tag/v1.66.1) ([#16179](https://github.com/kubermatic/kubermatic/pull/16179))
- Update web-terminal image version to0.13.0 ([#16029](https://github.com/kubermatic/kubermatic/pull/16029))
- Update machine controller to v1.66.0 ([#16145](https://github.com/kubermatic/kubermatic/pull/16145))
- Update KubeLB CCM to v1.4.3 ([#16132](https://github.com/kubermatic/kubermatic/pull/16132))
- Update MLA Gateway nginx image to v1.31.2-alpine ([#16084](https://github.com/kubermatic/kubermatic/pull/16084))
- Update AIKit and MetalLB application catalog entries with their current documentation and source URLs ([#15668](https://github.com/kubermatic/kubermatic/pull/15668))
- Update Grafana Alloy from v1.9.2 to v1.17.0 (Helm chart 1.1.2 → 1.10.0). The current chart configuration is unaffected. If extending the helm chart with additional components, please review the https://github.com/grafana/alloy/blob/main/operations/helm/charts/alloy/CHANGELOG.md and https://grafana.com/docs/alloy/latest/release-notes/ for any upstream breaking changes ([#16027](https://github.com/kubermatic/kubermatic/pull/16027))
- Update the KubeVirt CSI driver operator to v0.5.3 to fix StorageClasses getting an empty reclaimPolicy when no reclaimPolicy is configured ([#16032](https://github.com/kubermatic/kubermatic/pull/16032))
- Update machine-controller to v1.65.3 ([#16022](https://github.com/kubermatic/kubermatic/pull/16022))
- Update OSM to v1.10.7 ([#16020](https://github.com/kubermatic/kubermatic/pull/16020))
- Update Go version to 1.26.4 ([#15981](https://github.com/kubermatic/kubermatic/pull/15981))
- Update the default Cilium CNI version to 1.18.10 and added Cilium 1.17.16 and 1.18.10 as supported CNI versions ([#15961](https://github.com/kubermatic/kubermatic/pull/15961))
- Update Machine Controller to v1.65.2 ([#15926](https://github.com/kubermatic/kubermatic/pull/15926))
- Update Operating System Manager to v1.10.6 ([#15919](https://github.com/kubermatic/kubermatic/pull/15919))
- Update Operating System Manager to v1.10.5 ([#15847](https://github.com/kubermatic/kubermatic/pull/15847))
- Update KubeLB to v1.4.1 ([#15849](https://github.com/kubermatic/kubermatic/pull/15849))
- Update KubeLB CCM version to 1.3.10 ([#15806](https://github.com/kubermatic/kubermatic/pull/15806))
- Update vSphere CSI driver to v3.6.0 to pick up upstream session and ListView handling improvements that address vSphere volume attach failures after vCenter session expiry ([#15766](https://github.com/kubermatic/kubermatic/pull/15766))
- Update OSM version to [v1.10.4](https://github.com/kubermatic/operating-system-manager/releases/tag/v1.10.4) ([#15767](https://github.com/kubermatic/kubermatic/pull/15767))
- Update KubeLB version to 1.3.9 ([#15762](https://github.com/kubermatic/kubermatic/pull/15762))
- Update containerd version to v2.0.3 from v2.0.2 ([#15747](https://github.com/kubermatic/kubermatic/pull/15747))
- Update gpu-operator application to v26.3.0 ([#15739](https://github.com/kubermatic/kubermatic/pull/15739))
- Update OSM version to [v1.10.3](https://github.com/kubermatic/operating-system-manager/releases/tag/v1.10.3) ([#15678](https://github.com/kubermatic/kubermatic/pull/15678))
- Update the kubectl image tag to `1.33.4` to fix container startup failures referenced in Velero charts ([#15643](https://github.com/kubermatic/kubermatic/pull/15643))
- Update KubeLB to v1.3.7 ([#15664](https://github.com/kubermatic/kubermatic/pull/15664))
- Update OSM to v1.10.2 ([#15656](https://github.com/kubermatic/kubermatic/pull/15656))
- Update OSM to v1.10.1 ([#15597](https://github.com/kubermatic/kubermatic/pull/15597))
- Update application-catalog-manager ([#15593](https://github.com/kubermatic/kubermatic/pull/15593))
- Update cert-manager to v1.19.4 ([#15580](https://github.com/kubermatic/kubermatic/pull/15580))
- Update to KubeLB v1.3.5 ([#15588](https://github.com/kubermatic/kubermatic/pull/15588))

