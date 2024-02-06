# Kubermatic 2.25

- [v2.25.0-beta](#v2250-beta)

## [2.25.0-beta](https://github.com/kubermatic/kubermatic/releases/tag/2.25.0-beta)

### API Changes

- Add the edge cloud provider ([#13018](https://github.com/kubermatic/kubermatic/pull/13018))
- EtcdStatefulSetSettings: Add nodeSelector option to let etcd pods only run on specific nodes ([#12838](https://github.com/kubermatic/kubermatic/pull/12838))

### Breaking Changes

- * VMware Cloud Director: Support for attaching multiple networks to a vApp* [Action Required] The field `ovdcNetwork` in `cluster` and `preset` CRDs is considered deprecated for VMware Cloud Director and `ovdcNetworks` should be used instead ([#12996](https://github.com/kubermatic/kubermatic/pull/12996))

### Bugfixes

- Action required: if you use `velero.restic.deploy: true`, you will see new daemonset `node-agent` running in `velero` namespace. You might need to remove existing daemonset named `restic` manually ([#12998](https://github.com/kubermatic/kubermatic/pull/12998))
- Add a special `CiliumNetworkPolicy` feature gate to `KubermaticConfiguration` that has to be set it Cilium is used as CNI for Seeds ([#12886](https://github.com/kubermatic/kubermatic/pull/12886))
- Applied a fix to VPA caused by [upstream release issue](https://github.com/kubernetes/autoscaler/issues/5982) which caused insufficient RBAC permission for VPA recommender pod ([#12872](https://github.com/kubermatic/kubermatic/pull/12872))
- Cert-manager values block fixed. So cert-manager deployment will get updated as part of upgrade ([#12854](https://github.com/kubermatic/kubermatic/pull/12854))
- Fix `mirror-images` command in installer not being able to extract the addons ([#12868](https://github.com/kubermatic/kubermatic/pull/12868))
- Fix a bug where resources deployed in the user cluster namespace on seed, for CSI drivers, were not being removed when the CSI driver was disabled ([#13045](https://github.com/kubermatic/kubermatic/pull/13045))
- Fix cases where, when using dedicated infra- and ccm-credentials, infra-credentials were always overwritten by ccm-credentials ([#12421](https://github.com/kubermatic/kubermatic/pull/12421))
- Fix panic if no KubeVirt DNS config was set in the datacenter ([#12933](https://github.com/kubermatic/kubermatic/pull/12933))
- Fix the issue with blocked cluster provisioning, when selected initial applications that conflicted with Cilium system application and user-cluster-controller-manager was stuck ([#12997](https://github.com/kubermatic/kubermatic/pull/12997))
- No longer fail constructing vSphere endpoint when a `/` suffix is present in the datacenter configuration ([#12861](https://github.com/kubermatic/kubermatic/pull/12861))
- Remove "Node Resource Usage" dashboard and associated Prometheus recording rules from Master/Seed MLA as data shown was incorrect ([#12950](https://github.com/kubermatic/kubermatic/pull/12950))
- Stop constantly re-deploying operating-system-manager when registry mirrors are configured ([#12972](https://github.com/kubermatic/kubermatic/pull/12972))
- The Kubermatic installer will now detect DNS settings based on the Ingress instead of the nginx-ingress LoadBalancer, allowing for other ingress solutions to be properly detected ([#12934](https://github.com/kubermatic/kubermatic/pull/12934))
- Update Anexia CCM (cloud-controller-manager) to version 1.5.5- Fixes leaking LoadBalancer reconciliation metric- Updates various dependencies ([#12909](https://github.com/kubermatic/kubermatic/pull/12909))

### Chore

- Action required: [User-mla] If you had copied `values.yaml` of loki-distributed chart to further customize it, then please cleanup your copy of `values.yaml` for user-mla to retain your customization only ([#12967](https://github.com/kubermatic/kubermatic/pull/12967))      
- Add support for GCP/GCE cloud-controller-manager (CCM). Existing user clusters can be migrated to the external CCM by setting the `externalCloudProvider` feature gate or using the KKP Dashboard ([#12955](https://github.com/kubermatic/kubermatic/pull/12955))
- KKP is now built with Go 1.21.5 ([#12897](https://github.com/kubermatic/kubermatic/pull/12897))
- Kubermatic-installer: update local KubeVirt CDI chart to v1.58.0 ([#12850](https://github.com/kubermatic/kubermatic/pull/12850))        
- Kubermatic-installer: update local KubeVirt chart to v1.1.0 ([#12847](https://github.com/kubermatic/kubermatic/pull/12847))
- Remove 1.25 from list of supported versions on AKS (EOL on January 14th) ([#12962](https://github.com/kubermatic/kubermatic/pull/12962))
- REVERTED 12886 ([#12892](https://github.com/kubermatic/kubermatic/pull/12892))
- Some of high cardinality metrics were dropped from User-Cluster MLA prometheus. If your KKP installation was using some of those metrics for custom Grafana dashboards for user-clusters, your dashboards might stop showing some of the charts ([#12756](https://github.com/kubermatic/kubermatic/pull/12756))
- Update KKP images to Alpine 3.18; auxiliary single-binary images (alertmanager-authorization-server, network-interface-manager, s3-exporter and user-ssh-keys-agent) have been changed to use `gcr.io/distroless/static-debian12` as the base image ([#12870](https://github.com/kubermatic/kubermatic/pull/12870))
- Update metering to v1.1.2, fixing an error when a custom CA bundle is used ([#13013](https://github.com/kubermatic/kubermatic/pull/13013))
- Update to Go 1.21.4 ([#12857](https://github.com/kubermatic/kubermatic/pull/12857))
- Update to Go 1.21.6 ([#12968](https://github.com/kubermatic/kubermatic/pull/12968))
- Update Vertical Pod Autoscaler to 1.0 ([#12863](https://github.com/kubermatic/kubermatic/pull/12863))

### Cleanup

- Add support for Kubernetes v1.26.13, v1.27.10, v1.28.6, v1.29.1 ([#12981](https://github.com/kubermatic/kubermatic/pull/12981))
- Remove `CloudControllerReconcilledSuccessfully` (double L) Cluster condition, which was deprecated in KKP 2.21 and has since been replaced with `CloudControllerReconciledSuccessfully` (single L) ([#12867](https://github.com/kubermatic/kubermatic/pull/12867))
- Remove CriticalAddonsOnly toleration from node-local-dns DaemonSet as it has more general tolerations configured ([#12957](https://github.com/kubermatic/kubermatic/pull/12957))
- Remove support for Kubernetes 1.26 ([#13032](https://github.com/kubermatic/kubermatic/pull/13032))

### Documentation

- Examples now include command to generate secrets that works on vanilla macOS ([#12974](https://github.com/kubermatic/kubermatic/pull/12974))

### Miscellaneous

- - Add a new field `backupConfig` to the Cluster Spec.- Add a new API type `ClusterBackupStorageLocation` for cluster backup integration ([#12929](https://github.com/kubermatic/kubermatic/pull/12929))
- Add Support for Kubernetes 1.29 ([#12936](https://github.com/kubermatic/kubermatic/pull/12936))
- Addon manifests are now loaded once upon startup of the seed-controller-manager instead of during every reconciliation. Invalid addons will now send the seed-controller-manager into a crash loop ([#12684](https://github.com/kubermatic/kubermatic/pull/12684))
- Deprecate v1.11 and v1.12 Cilium and Hubble KKP Addons, as Cilium CNI is managed by Applications from version 1.13 ([#12848](https://github.com/kubermatic/kubermatic/pull/12848))
- If the seed cluster is using Cilium as CNI, create CiliumClusterwideNetworkPolicy for api-server connectivity ([#12924](https://github.com/kubermatic/kubermatic/pull/12924))
- Increase the default resources for VPA components to prevent OOMs ([#12887](https://github.com/kubermatic/kubermatic/pull/12887))       
- Kube state metrics can be configured to get metrics for custom kubernetes resources ([#13027](https://github.com/kubermatic/kubermatic/pull/13027))
- Openstack: allow configuring Cinder CSI topology support either on `Cluster` or `Seed` resource field `cinderTopologyEnabled` ([#12878](https://github.com/kubermatic/kubermatic/pull/12878))

### New Feature

- - Deploy all Velero components on user cluster when backup is enabled ([#13010](https://github.com/kubermatic/kubermatic/pull/13010))   
- Action required: User-mla Cortex chart upgraded to resolve issues for cortex-compactor and improve stability of user-cluster MLA feature. Few actions are required to be taken to use new upgraded charts:* Refer to [Upstream helm chart values](https://github.com/cortexproject/cortex-helm-chart/blob/v2.1.0/values.yaml) to see the latest default values.* Some of the values from earlier `values.yaml` are now incompatible with latest version. They are removed in the `values.yaml` in the current chart. But if you had copied the original values.yaml to customize it further, you may see that `kubermatic-installer` will detect such incompatible options and churn out errors and explain that action that needs to be taken.* The memcached-* charts are now subcharts of cortex chart so if you provided configuration for `memcached-*` blocks in your `values.yaml` for user-mla, you must move them under `cortex:` block ([#12935](https://github.com/kubermatic/kubermatic/pull/12935))
- Add `Seed.spec.metering.retentionDays` to configure the Prometheus retention; fix missing defaulting for `Seed.spec.metering.storageSize` ([#12843](https://github.com/kubermatic/kubermatic/pull/12843))
- Add k8sgpt operator to the Default Application Catalogue ([#13025](https://github.com/kubermatic/kubermatic/pull/13025))
- Add new admin option to enable/disable user cluster backups ([#12888](https://github.com/kubermatic/kubermatic/pull/12888))
- Add support for configuring allowed IP allocation modes for VMware Cloud Director in KubermaticSettings ([#13002](https://github.com/kubermatic/kubermatic/pull/13002))
- Application-catalog: add KubeVirt ([#12851](https://github.com/kubermatic/kubermatic/pull/12851))
- Charts/kubermatic-operator: ability to configure environment variables for the kubermatic-operator pod ([#12973](https://github.com/kubermatic/kubermatic/pull/12973))
- Update KubeLB CCM image to v0.5.0 ([#13023](https://github.com/kubermatic/kubermatic/pull/13023))
- Upstream Documentation and SourceURLs can be added to ApplicationDefinitions ([#13019](https://github.com/kubermatic/kubermatic/pull/13019))
- VMware Cloud Director: Move CSI controller to seed cluster ([#13020](https://github.com/kubermatic/kubermatic/pull/13020))
