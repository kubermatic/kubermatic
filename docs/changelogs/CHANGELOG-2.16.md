# Kubermatic 2.16

- [v2.16.0](#v2160)
- [v2.16.1](#v2161)
- [v2.16.2](#v2162)
- [v2.16.3](#v2163)
- [v2.16.4](#v2164)
- [v2.16.5](#v2165)
- [v2.16.6](#v2166)
- [v2.16.7](#v2167)
- [v2.16.8](#v2168)
- [v2.16.9](#v2169)
- [v2.16.10](#v21610)
- [v2.16.11](#v21611)
- [v2.16.12](#v21612)

## [v2.16.12](https://github.com/kubermatic/kubermatic/releases/tag/v2.16.12)

### Security

Two vulnerabilities were identified in Kubernetes ([CVE-2021-25741](https://github.com/kubernetes/kubernetes/issues/104980) and [CVE-2020-8561](https://github.com/kubernetes/kubernetes/issues/104720)), of which one (CVE-2021-25741) was fixed in Kubernetes 1.19.15 / 1.20.11. CVE-2020-8561 is mitigated by Kubermatic not allowing users to reconfigure the kube-apiserver.

Because of these updates, this KKP release includes automatic update rules for all 1.19/1.20 clusters older than 1.19.15 / 1.20.11. This release also removes all affected Kubernetes versions from the list of supported versions. While CVE-2020-8561 affects the controlplane, CVE-2021-25741 affects the kubelets, which means that updating the controlplane is not enough. Once the automated controlplane updates have completed, an administrator must manually patch all vulnerable `MachineDeployment`s in all affected userclusters.

To lower the resource consumption on the seed clusters during the reconciliation / node rotation, it's recommended to adjust the `spec.seedControllerManager.maximumParallelReconciles` option in the `KubermaticConfiguration` to restrict the number of parallel updates. Users of the legacy `kubermatic` Helm chart need to update `kubermatic.maxParallelReconcile` in their `values.yaml` to achieve the same effect.

The automatic update rules can, if needed, be overwritten using the `spec.versions.kubernetes.updates` field in the `KubermaticConfiguration` or updating the `updates.yaml` if using the legacy `kubermatic` Helm chart. See [#7824](https://github.com/kubermatic/kubermatic/issues/7824) for how the versions and updates are configured. It is however not recommended to deviate from the default and leave userclusters vulnerable.

### Misc

- Add support of Kubernetes 1.20 in cluster-autoscaler addon ([#7521](https://github.com/kubermatic/kubermatic/issues/7521))
- Remove Gatekeeper from default accessible addon list ([#7532](https://github.com/kubermatic/kubermatic/issues/7532))
- Fix dashboard source in the Prometheus Exporter dashboard ([#7640](https://github.com/kubermatic/kubermatic/issues/7640))


## [v2.16.11](https://github.com/kubermatic/kubermatic/releases/tag/v2.16.11)

### Security

- Upgrade machine-controller to v1.24.5 to address [runC vulnerability CVE-2021-30465](https://github.com/opencontainers/runc/security/advisories/GHSA-c3xm-pvg7-gh7r) ([#7165](https://github.com/kubermatic/kubermatic/issues/7165))

### Bugfixes

- Fix a bug that always applies default values to container resources ([#7302](https://github.com/kubermatic/kubermatic/issues/7302))
- Add `ClusterFeatureCCMClusterName` feature for OpenStack clusters. This feature adds the `--cluster-name` flag to the OpenStack external CCM deployment. The feature gate is enabled by default for newly created clusters. Enabling this feature gate for existing clusters will cause the external CCM to lose the track of the existing cloud resources (such as Load Balancers), so it's up to the users to manually clean up any leftover resources. ([#7330](https://github.com/kubermatic/kubermatic/issues/7330))

### Misc

- Add support for `use-octavia` setting in Openstack provider specs. It defaults to `true` but leaves the possibility to set it to `false` if your provider doesn't support Octavia yet but Neutron LBaaSv2 ([#6529](https://github.com/kubermatic/kubermatic/issues/6529))


## [v2.16.10](https://github.com/kubermatic/kubermatic/releases/tag/v2.16.10)

### Misc

- Fix CA certificates not being mounted on Seed clusters using CentOS/Fedora ([#6680](https://github.com/kubermatic/kubermatic/issues/6680))
- Use the systemd cgroup driver for newly-created Kubernetes 1.19+ clusters using the kubeadm provider. Since the kubelet-configmap addon is not reconciled, this change will not affect existing clusters, only newly-created clusters. ([#7065](https://github.com/kubermatic/kubermatic/issues/7065))
- Re-enable NodeLocal DNS Cache in user clusters ([#7075](https://github.com/kubermatic/kubermatic/issues/7075))


## [v2.16.9](https://github.com/kubermatic/kubermatic/releases/tag/v2.16.9)

### Misc

- Adds `FELIX_IGNORELOOSERPF=true` to `calico-node` container env to allow running on nodes with `net.ipv4.conf.*.rp_filter = 2` set ([#6855](https://github.com/kubermatic/kubermatic/issues/6855))
- Fix default version configuration to have automatic upgrade from Kubernetes 1.16 to 1.17 ([#6899](https://github.com/kubermatic/kubermatic/issues/6899))
- Fix OpenStack crashing with Kubernetes 1.20 and 1.21 ([#6924](https://github.com/kubermatic/kubermatic/issues/6924))
- Update machine-controller to 1.24.4. Fixed double instance creation in us-east1 AWS ([#6962](https://github.com/kubermatic/kubermatic/issues/6962))


## [v2.16.8](https://github.com/kubermatic/kubermatic/releases/tag/v2.16.8)

### Misc

- Installer does not listen on port 8080 anymore ([#6788](https://github.com/kubermatic/kubermatic/issues/6788))
- Node-local-dns is now using UDP for external queries ([#6796](https://github.com/kubermatic/kubermatic/issues/6796))


## [v2.16.7](https://github.com/kubermatic/kubermatic/releases/tag/v2.16.7)

### Bugfixes

- Fix deployment of Openstack CCM ([#6750](https://github.com/kubermatic/kubermatic/issues/6750))
- Projects are now synced from the Master cluster to all Seed clusters. Fixes issue where user clusters could not be created properly on multi seed clusters, when the seed is not also the master cluster ([#6754](https://github.com/kubermatic/kubermatic/issues/6754))
- Fix installer trying an invalid certificate to test cert-manager ([#6761](https://github.com/kubermatic/kubermatic/issues/6761))

### Misc

- Allow to disable the s3-credentials Secret in the Minio chart ([#6760](https://github.com/kubermatic/kubermatic/issues/6760))


## [v2.16.6](https://github.com/kubermatic/kubermatic/releases/tag/v2.16.6)

### Bugfixes

- Fix cert-manager validating webhook ([#6741](https://github.com/kubermatic/kubermatic/issues/6741))
- Fix the operator failing to reconcile the ValidatingWebhookConfiguration object for the cluster validation webhook ([#6639](https://github.com/kubermatic/kubermatic/issues/6639))

### Misc

- Change default gatekeeper webhook timeout to 3 sec ([#6709](https://github.com/kubermatic/kubermatic/issues/6709))
- Update Velero to 1.5.3 ([#6701](https://github.com/kubermatic/kubermatic/issues/6701))


## [v2.16.5](https://github.com/kubermatic/kubermatic/releases/tag/v2.16.5)

### Misc

- Update nginx-ingress-controller to 0.44.0 ([#6651](https://github.com/kubermatic/kubermatic/issues/6651))

### Bugfixes

- Fix CE installer binary in EE downloads ([#6673](https://github.com/kubermatic/kubermatic/issues/6673))


## [v2.16.4](https://github.com/kubermatic/kubermatic/releases/tag/v2.16.4)

### Misc

- Provide the possibility of configuring leader election parameters for user cluster components ([#6641](https://github.com/kubermatic/kubermatic/issues/6641))
- Add `registry_mirrors` to Seed node settings ([#6667](https://github.com/kubermatic/kubermatic/issues/6667))

### Bugfixes

- Fix nodeport-proxy role used with LoadBalancer expose strategy ([#6646](https://github.com/kubermatic/kubermatic/issues/6646))


## [v2.16.3](https://github.com/kubermatic/dashboard/releases/tag/v2.16.3)

This version includes significant improvements to Hetzner userclusters. Please refer to the amended [2.16 upgrade notes](https://docs.kubermatic.com/kubermatic/v2.16/upgrading/2.15_to_2.16/) for more information.

### Misc

- Add support for Hetzner CCM ([#6588](https://github.com/kubermatic/kubermatic/issues/6588))
- Update Hetzner CSI ([#6615](https://github.com/kubermatic/kubermatic/issues/6615))
- Update CSI drivers ([#6594](https://github.com/kubermatic/kubermatic/issues/6594))
- Increase default gatekeeper webhook timeout from 2 to 10 seconds, and add option in cluster settings to configure it ([#6603](https://github.com/kubermatic/kubermatic/issues/6603))
- Remove duplicate Kubeadm hints from cluster page ([#3114](https://github.com/kubermatic/dashboard/issues/3114))
- Change vSphere's diskSizeGB option from optional to required ([#3121](https://github.com/kubermatic/dashboard/issues/3121))

### Bugfixes

- Fix a bug in OPA integration where deleting a Constraint Template in the seed cluster, when the user cluster Constraint Template is already deleted, caused the deletion to get stuck. ([#6582](https://github.com/kubermatic/kubermatic/issues/6582))
- Fix a bug in OPA integration where creating a cluster with OPA integration enabled didn't trigger the Constraint Template reconcile loop ([#6582](https://github.com/kubermatic/kubermatic/issues/6582))
- Fix a bug with Kubermatic constraints delete getting stuck when corresponding user cluster constraint is missing ([#6598](https://github.com/kubermatic/kubermatic/issues/6598))


## [v2.16.2](https://github.com/kubermatic/kubermatic/releases/tag/v2.16.2)

### Bugfixes

- Fix KAS service port in Tunneling agent configuration ([#6569](https://github.com/kubermatic/kubermatic/issues/6569))


## [v2.16.1](https://github.com/kubermatic/kubermatic/releases/tag/v2.16.1)

**Note:** No Docker images have been published for this release. Please use 2.16.2 instead.

### Bugfixes

- Fix issue with gatekeeper not recognizing the AdmissionReview v1 version by changing the webhook to use v1beta1 ([#6550](https://github.com/kubermatic/kubermatic/issues/6550))


## [v2.16.0](https://github.com/kubermatic/kubermatic/releases/tag/v2.16.0)

Before upgrading, make sure to read the [general upgrade guidelines](https://docs.kubermatic.com/kubermatic/v2.16/upgrading/guidelines/)
as well as the [2.16 upgrade notes](https://docs.kubermatic.com/kubermatic/v2.16/upgrading/2.15_to_2.16/).

### Supported Kubernetes Versions

* 1.17.9
* 1.17.11
* 1.17.12
* 1.17.13
* 1.17.16
* 1.18.6
* 1.18.8
* 1.18.10
* 1.18.14
* 1.19.0
* 1.19.2
* 1.19.3
* 1.20.2

### Highlights

- Add Kubernetes 1.20, remove Kubernetes 1.16 ([#6032](https://github.com/kubermatic/kubermatic/issues/6032), [#6122](https://github.com/kubermatic/kubermatic/issues/6122))
- Add Tunneling expose strategy (tech preview) ([#6445](https://github.com/kubermatic/kubermatic/issues/6445))
- First parts of the revamped V2 API are available as a preview (see API section below)
- cert-manager is not a hard dependency for KKP anymore; certificates are acquired using Ingress annotations instead ([#5962](https://github.com/kubermatic/kubermatic/issues/5962), [#5969](https://github.com/kubermatic/kubermatic/issues/5969), [#6069](https://github.com/kubermatic/kubermatic/issues/6069), [#6282](https://github.com/kubermatic/kubermatic/issues/6282), [#6119](https://github.com/kubermatic/kubermatic/issues/6119))

### Breaking Changes

- This is the last release for which the legacy Helm chart is available. Users are encouraged to migrate to the KKP Operator.

### Cloud Provider

- Add Anexia provider ([#6101](https://github.com/kubermatic/kubermatic/issues/6101), [#6128](https://github.com/kubermatic/kubermatic/issues/6128))
- Make `cluster` field in vSphere datacenter spec optional ([#5886](https://github.com/kubermatic/kubermatic/issues/5886))
- Fix creation of RHEL8 machines ([#5950](https://github.com/kubermatic/kubermatic/issues/5950))
- Default to the latest available OpenStack CCM version. This fixes a bug where newly-created OpenStack clusters running Kubernetes 1.19 were using the in-tree cloud provider instead of the external CCM. Those clusters will remain to use the in-tree cloud provider until the CCM migration mechanism is not implemented. The OpenStack clusters running Kubernetes 1.19+ and created with KKP 2.15.6+ will use the external CCM ([#6272](https://github.com/kubermatic/kubermatic/issues/6272))
- Use CoreOS-Cloud-Config for Flatcar machines on AWS ([#6405](https://github.com/kubermatic/kubermatic/issues/6405))

### Misc

- Add Kubernetes 1.17.12, 1.19.2 ([#5927](https://github.com/kubermatic/kubermatic/issues/5927))
- Add Kubernetes 1.17.13, 1.18.10, 1.19.3 ([#6032](https://github.com/kubermatic/kubermatic/issues/6032))
- Add `DNSDomain` variable to addon TemplateData ([#6160](https://github.com/kubermatic/kubermatic/issues/6160))
- Add `MaximumParallelReconciles` option to KubermaticConfiguration ([#6002](https://github.com/kubermatic/kubermatic/issues/6002))
- Add `operator.kubermatic.io/skip-reconciling` annotation to Seeds to allow step-by-step seed cluster upgrades ([#5883](https://github.com/kubermatic/kubermatic/issues/5883))
- Add a controller which syncs Constraints from the seed cluster user cluster namespace to the corresponding user clusters when OPA integration is enabled ([#6224](https://github.com/kubermatic/kubermatic/issues/6224))
- Add a new feature gate to the seed-controller to enable etcd-launcher for all user clusters ([#5997](https://github.com/kubermatic/kubermatic/issues/5997), [#5973](https://github.com/kubermatic/kubermatic/issues/5973))
- Add admission control configuration for the user cluster API deployment ([#6308](https://github.com/kubermatic/kubermatic/issues/6308))
- Add new cluster-autoscaler addon ([#5869](https://github.com/kubermatic/kubermatic/issues/5869))
- Add service account token volume projection options for user clusters ([#6382](https://github.com/kubermatic/kubermatic/issues/6382))
- Add support for KubermaticConfiguration in `image-loader` utility ([#6063](https://github.com/kubermatic/kubermatic/issues/6063))
- Add support for `InstanceReadyCheckPeriod` and `InstanceReadyCheckTimeout` to Openstack provider ([#6139](https://github.com/kubermatic/kubermatic/issues/6139))
- Allow controlling external cluster functionality with global settings ([#5912](https://github.com/kubermatic/kubermatic/issues/5912))
- Allow to customize the Docker image tag for Cluster Addons ([#6102](https://github.com/kubermatic/kubermatic/issues/6102))
- Always mount CABundle for Dex into the kubermatic-api Pod, even when `OIDCKubeCfgEndpoint` is disabled ([#5968](https://github.com/kubermatic/kubermatic/issues/5968))
- Avoid forcing cleanup of failed backup job pods, so that cluster administrators can still look at the pod's logs ([#5913](https://github.com/kubermatic/kubermatic/issues/5913))
- Create an RBAC role to allow kubeadm to get nodes. This fixes nodes failing to join kubeadm clusters running Kubernetes 1.18+ ([#6241](https://github.com/kubermatic/kubermatic/issues/6241))
- Dex configuration does not support `staticPasswordLogins` anymore, use `staticPasswords` instead ([#6289](https://github.com/kubermatic/kubermatic/issues/6289))
- Expose `ServiceAccountSettings` in cluster API object ([#6423](https://github.com/kubermatic/kubermatic/issues/6423))
- Extend Cluster CRD with `PodNodeSelectorAdmissionPluginConfig` ([#6305](https://github.com/kubermatic/kubermatic/issues/6305))
- Extend global settings for resource quota ([#6448](https://github.com/kubermatic/kubermatic/issues/6448))
- Fix Kubermatic Operator getting stuck in Kubernetes 1.18 clusters when reconciling Ingresses ([#5915](https://github.com/kubermatic/kubermatic/issues/5915))
- Fix Prometheus alerts misfiring about absent KKP components ([#6167](https://github.com/kubermatic/kubermatic/issues/6167))
- Fix Prometheus `cluster_info` metric having the wrong `type` label ([#6138](https://github.com/kubermatic/kubermatic/issues/6138))
- Fix cert-manager webhook Service naming ([#6150](https://github.com/kubermatic/kubermatic/issues/6150))
- Fix installer not being able to probe for Certificate support ([#6135](https://github.com/kubermatic/kubermatic/issues/6135))
- Fix master-controller-manager being too verbose ([#5889](https://github.com/kubermatic/kubermatic/issues/5889))
- Fix missing logo in Dex login screens ([#6019](https://github.com/kubermatic/kubermatic/issues/6019))
- Fix orphaned apiserver-is-running initContainers in usercluster controlplane. This can cause a short reconciliation burst to bring older usercluster resources in all Seed clusters up to date. Tune the maxReconcileLimit if needed ([#6329](https://github.com/kubermatic/kubermatic/issues/6329))
- Fix overflowing `kubermatic.io/cleaned-up-loadbalancers` annotation on Cluster objects ([#6229](https://github.com/kubermatic/kubermatic/issues/6229))
- Fix user-cluster Grafana dashboard showing inflated numbers under certain circumstances ([#6026](https://github.com/kubermatic/kubermatic/issues/6026))
- Gatekeeper is now deployed automatically for the user clusters as part of Kubermatic OPA integration ([#5814](https://github.com/kubermatic/kubermatic/issues/5814))
- Improve Helm error handling in KKP Installer ([#6076](https://github.com/kubermatic/kubermatic/issues/6076))
- Improved initial node deployment creation process. Right now cluster annotation is used to save the node deployment object before it is created to improve stability. ([#6064](https://github.com/kubermatic/kubermatic/issues/6064))
- Make etcd-launcher repository configurable in `KubermaticConfiguration` CRD ([#5880](https://github.com/kubermatic/kubermatic/issues/5880))
- Make imagePullSecret optional for Kubermatic Operator ([#5874](https://github.com/kubermatic/kubermatic/issues/5874))
- Makefile: better support for compiling with debug symbols ([#5933](https://github.com/kubermatic/kubermatic/issues/5933))
- Move to k8s.gcr.io Docker registry for CoreDNS, metrics-server, and NodeLocalDNSCache ([#5963](https://github.com/kubermatic/kubermatic/issues/5963))
- Raise cert-manager resource limits to prevent OOMKills ([#6216](https://github.com/kubermatic/kubermatic/issues/6216))
- Remove Helm charts for deprecated ELK (Elasticsearch, Fluentbit, Kibana) stack ([#6149](https://github.com/kubermatic/kubermatic/issues/6149))
- Remove kubernetes-dashboard Helm chart ([#6108](https://github.com/kubermatic/kubermatic/issues/6108))
- Ship image-loader as part of GitHub releases ([#6092](https://github.com/kubermatic/kubermatic/issues/6092))
- Start as a fresh etcd member if data directory is empty ([#6221](https://github.com/kubermatic/kubermatic/issues/6221))
- The User SSH Key Agent can now be disabled per cluster in order to manage SSH keys manually ([#6443](https://github.com/kubermatic/kubermatic/issues/6443), [#6426](https://github.com/kubermatic/kubermatic/issues/6426), [#6444](https://github.com/kubermatic/kubermatic/issues/6444))
- Update to networking.k8s.io/v1beta1 for managing Ingresses ([#6292](https://github.com/kubermatic/kubermatic/issues/6292))

### UI

- Add Datastore/Datastore Cluster support to the VSphere provider in the wizard ([#2762](https://github.com/kubermatic/dashboard/issues/2762))
- Add Preset management UI to the admin settings ([#2880](https://github.com/kubermatic/dashboard/issues/2880))
- Add flag `continuouslyReconcile` to addons ([#2618](https://github.com/kubermatic/dashboard/issues/2618))
- Add option to enable/disable external cluster import feature from admin settings ([#2644](https://github.com/kubermatic/dashboard/issues/2644))
- Add option to filter clusters ([#2695](https://github.com/kubermatic/dashboard/issues/2695))
- Add option to specify Pod Node Selector Configuration ([#2929](https://github.com/kubermatic/dashboard/issues/2929))
- Add support for Anexia cloud provider ([#2693](https://github.com/kubermatic/dashboard/issues/2693))
- Add support for `instanceReadyCheckPeriod` and `instanceReadyCheckTimeout` to the Openstack provider ([#2781](https://github.com/kubermatic/dashboard/issues/2781))
- Add the option to specify OS/data disk size for Azure clusters and support selection of multiple zones ([#2547](https://github.com/kubermatic/dashboard/issues/2547))
- Allow adding help text for addon forms ([#2770](https://github.com/kubermatic/dashboard/issues/2770))
- Allow specifying help text for addon form controls ([#6117](https://github.com/kubermatic/kubermatic/issues/6117))
- Azure resource groups, security groups and route tables will be now loaded from the API to provide autocompletion ([#2936](https://github.com/kubermatic/dashboard/issues/2936))
- Cluster related resources will be now displayed in tabs ([#2876](https://github.com/kubermatic/dashboard/issues/2876))
- Display deletion state of accessible addons ([#2674](https://github.com/kubermatic/dashboard/issues/2674))
- Distributions in the wizard are now correctly shown based on admin settings ([#2839](https://github.com/kubermatic/dashboard/issues/2839))
- Fix addon variables edit ([#2731](https://github.com/kubermatic/dashboard/issues/2731))
- Fix end of life chip/badge styling ([#2841](https://github.com/kubermatic/dashboard/issues/2841))
- Fix issue with listing all projects if one of them had no owner set ([#2848](https://github.com/kubermatic/dashboard/issues/2848))
- Fix loading of the access rights in the SSH keys view ([#2645](https://github.com/kubermatic/dashboard/issues/2645))
- Fix missing group name on Service Account list ([#2851](https://github.com/kubermatic/dashboard/issues/2851))
- Fix project selector auto-scroll to selected value on refresh ([#2638](https://github.com/kubermatic/dashboard/issues/2638))
- Fix styling of 'Add RBAC Binding' dialog ([#2850](https://github.com/kubermatic/dashboard/issues/2850))
- Fix the bug with labels that were removed from form after pressing enter key ([#2903](https://github.com/kubermatic/dashboard/issues/2903))
- Fix wizard rendering in Safari ([#2661](https://github.com/kubermatic/dashboard/issues/2661))
- Improve browser support ([#2668](https://github.com/kubermatic/dashboard/issues/2668))
- Non-existing default projects will be now unchecked in the settings ([#2630](https://github.com/kubermatic/dashboard/issues/2630))
- Openstack: Fill the form with defaults for username and domain name ([#2928](https://github.com/kubermatic/dashboard/issues/2928))
- Remove `mat-icon` dependency and replace them by own icons ([#2883](https://github.com/kubermatic/dashboard/issues/2883))
- Restore list of cluster nodes ([#2773](https://github.com/kubermatic/dashboard/issues/2773))
- Support User SSH Keys in Kubeadm cloud provider ([#2747](https://github.com/kubermatic/dashboard/issues/2747))
- Switch to new cluster endpoints ([#2641](https://github.com/kubermatic/dashboard/issues/2641))
- The seed datacenter param was removed from the path. It is no longer required thanks to the switch to the new version of API ([#2815](https://github.com/kubermatic/dashboard/issues/2815))
- Update cluster resource loading states ([#2690](https://github.com/kubermatic/dashboard/issues/2690))
- Update login page background ([#2849](https://github.com/kubermatic/dashboard/issues/2849))
- Use an endpoint to get AWS Security Group IDs in the wizard for the cluster creation ([#2909](https://github.com/kubermatic/dashboard/issues/2909))
- Use endpoint to load Azure subnets autocompletions ([#2988](https://github.com/kubermatic/dashboard/issues/2988))

### API

- Add endpoint to list AWS Security Groups: `GET /api/v1/providers/aws/{dc}/securitygroups` ([#6331](https://github.com/kubermatic/kubermatic/issues/6331))
- Add new endpoints to list/create/update presets ([#6208](https://github.com/kubermatic/kubermatic/issues/6208))
- remove deprecated `v1/nodes` endpoints ([#6031](https://github.com/kubermatic/kubermatic/issues/6031))
- change endpoint name from DeleteMachineNode to DeleteMachineDeploymentNode ([#6115](https://github.com/kubermatic/kubermatic/issues/6115))
- first parts of the revamped V2 API are available:
  - manage `ConstraintTemplate`s ([#5917](https://github.com/kubermatic/kubermatic/issues/5917), [#5966](https://github.com/kubermatic/kubermatic/issues/5966), [#5885](https://github.com/kubermatic/kubermatic/issues/5885), [#5959](https://github.com/kubermatic/kubermatic/issues/5959))
  - manage `Constraint`s ([#6034](https://github.com/kubermatic/kubermatic/issues/6034), [#6127](https://github.com/kubermatic/kubermatic/issues/6127), [#6116](https://github.com/kubermatic/kubermatic/issues/6116), [#6141](https://github.com/kubermatic/kubermatic/issues/6141))
  - list Azure Subnets, VNets etc. ([#6395](https://github.com/kubermatic/kubermatic/issues/6395), [#6363](https://github.com/kubermatic/kubermatic/issues/6363), [#6340](https://github.com/kubermatic/kubermatic/issues/6340))
  - list provider related resources ([#6228](https://github.com/kubermatic/kubermatic/issues/6228), [#6262](https://github.com/kubermatic/kubermatic/issues/6262), [#6264](https://github.com/kubermatic/kubermatic/issues/6264), [#6287](https://github.com/kubermatic/kubermatic/issues/6287), [#6223](https://github.com/kubermatic/kubermatic/issues/6223), [#6275](https://github.com/kubermatic/kubermatic/issues/6275))
  - manage MachineDeployments ([#6109](https://github.com/kubermatic/kubermatic/issues/6109), [#6111](https://github.com/kubermatic/kubermatic/issues/6111), [#6156](https://github.com/kubermatic/kubermatic/issues/6156), [#6068](https://github.com/kubermatic/kubermatic/issues/6068), [#6157](https://github.com/kubermatic/kubermatic/issues/6157), [#6074](https://github.com/kubermatic/kubermatic/issues/6074), [#6132](https://github.com/kubermatic/kubermatic/issues/6132), [#6107](https://github.com/kubermatic/kubermatic/issues/6107), [#6136](https://github.com/kubermatic/kubermatic/issues/6136))
  - manage cluster addons ([#6215](https://github.com/kubermatic/kubermatic/issues/6215))
  - manage RBAC in clusters ([#6196](https://github.com/kubermatic/kubermatic/issues/6196), [#6187](https://github.com/kubermatic/kubermatic/issues/6187), [#6177](https://github.com/kubermatic/kubermatic/issues/6177), [#6162](https://github.com/kubermatic/kubermatic/issues/6162))
  - manage Gatekeeper (Open Policy Agent, OPA) ([#6306](https://github.com/kubermatic/kubermatic/issues/6306), [#6286](https://github.com/kubermatic/kubermatic/issues/6286))
  - manage cluster nodes ([#6030](https://github.com/kubermatic/kubermatic/issues/6030), [#6130](https://github.com/kubermatic/kubermatic/issues/6130))
  - manage cluster SSH keys ([#6005](https://github.com/kubermatic/kubermatic/issues/6005))
  - list cluster namespaces ([#6004](https://github.com/kubermatic/kubermatic/issues/6004))
  - access dashboard proxy ([#6299](https://github.com/kubermatic/kubermatic/issues/6299))
  - access cluster health and metrics ([#5872](https://github.com/kubermatic/kubermatic/issues/5872), [#5908](https://github.com/kubermatic/kubermatic/issues/5908))
  - manage kubeconfig and tokens ([#5881](https://github.com/kubermatic/kubermatic/issues/5881), [#6238](https://github.com/kubermatic/kubermatic/issues/6238))
  - list cluster updates ([#6021](https://github.com/kubermatic/kubermatic/issues/6021))

### Updates

- Prometheus 2.23.0 ([#6290](https://github.com/kubermatic/kubermatic/issues/6290))
- Thanos 0.17.2 ([#6290](https://github.com/kubermatic/kubermatic/issues/6290))
- Velero 1.5.2 ([#6145](https://github.com/kubermatic/kubermatic/issues/6145))
- machine-controller v1.23.1 ([#6387](https://github.com/kubermatic/kubermatic/issues/6387))


### Changes since v2.16.0-beta.1

- Add option to disable User SSH Key Agent from the cluster wizard ([#3025](https://github.com/kubermatic/dashboard/issues/3025))
