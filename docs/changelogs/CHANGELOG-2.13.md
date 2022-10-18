# Kubermatic 2.13

- [v2.13.0](#v2130)
- [v2.13.1](#v2131)
- [v2.13.2](#v2132)
- [v2.13.3](#v2133)
- [v2.13.4](#v2134)
- [v2.13.5](#v2135)
- [v2.13.6](#v2136)
- [v2.13.7](#v2137)
- [v2.13.8](#v2138)
- [v2.13.9](#v2139)
- [v2.13.10](#v21310)

## [v2.13.10](https://github.com/kubermatic/kubermatic/releases/tag/v2.13.10)

### Misc

- Improve image-loader usability, add support for Helm charts ([#6090](https://github.com/kubermatic/kubermatic/issues/6090))

## [v2.13.9](https://github.com/kubermatic/kubermatic/releases/tag/v2.13.9)

### Misc

- Add a configuration flag for seed-controller-manager to enforce default addons on userclusters. Disabled by default ([#5987](https://github.com/kubermatic/kubermatic/issues/5987))


## [v2.13.8](https://github.com/kubermatic/kubermatic/releases/tag/v2.13.8)

### Bugfixes

- Fix `componentsOverride` of a cluster affecting other clusters ([#5702](https://github.com/kubermatic/kubermatic/issues/5702))


## [v2.13.7](https://github.com/kubermatic/kubermatic/releases/tag/v2.13.7)

- Added Kubernetes v1.16.13, and removed v1.16.2-7 in default version configuration ([#5661](https://github.com/kubermatic/kubermatic/issues/5661))
- Added Kubernetes v1.17.9, and removed v1.17.0-3 in default version configuration ([#5667](https://github.com/kubermatic/kubermatic/issues/5667))


## [v2.13.6](https://github.com/kubermatic/kubermatic/releases/tag/v2.13.6)

- Fixed a bug preventing editing of existing cluster credential secrets ([#5569](https://github.com/kubermatic/kubermatic/issues/5569))


## [v2.13.5](https://github.com/kubermatic/kubermatic/releases/tag/v2.13.5)

- Updated machine-controller to v1.10.4 to address issue in CNI plugins ([#5443](https://github.com/kubermatic/kubermatic/issues/5443))


## [v2.13.4](https://github.com/kubermatic/kubermatic/releases/tag/v2.13.4)

- **ACTION REQUIRED:** The most recent backup for user clusters is kept when the cluster is deleted. Adjust the cleanup-container to get the old behaviour (delete all backups) back. ([#5262](https://github.com/kubermatic/kubermatic/issues/5262))
- Updated machine-controller to v1.10.3 to fix the Docker daemon/CLI version incompatibility ([#5427](https://github.com/kubermatic/kubermatic/issues/5427))


## [v2.13.3](https://github.com/kubermatic/kubermatic/releases/tag/v2.13.3)

This release contains only improvements to the image build process.


## [v2.13.2](https://github.com/kubermatic/kubermatic/releases/tag/v2.13.2)

- Openstack: include distributed routers in existing router search ([#5334](https://github.com/kubermatic/kubermatic/issues/5334))


## [v2.13.1](https://github.com/kubermatic/kubermatic/releases/tag/v2.13.1)

- Fixed swagger and API client for ssh key creation. ([#5069](https://github.com/kubermatic/kubermatic/issues/5069))
- Added Kubernetes v1.15.10, v1.16.7, v1.17.3 ([#5102](https://github.com/kubermatic/kubermatic/issues/5102))
- AddonConfig's shortDescription field is now used in the accessible addons overview. ([#2050](https://github.com/kubermatic/dashboard/issues/2050))


## [v2.13.0](https://github.com/kubermatic/kubermatic/releases/tag/v2.13.0)

### Supported Kubernetes versions

- `1.15.5`
- `1.15.6`
- `1.15.7`
- `1.15.9`
- `1.16.2`
- `1.16.3`
- `1.16.4`
- `1.16.6`
- `1.17.0`
- `1.17.2`
- Openshift `v4.1.18`

### Major changes

- End-of-Life Kubernetes v1.14 is no longer supported. ([#4987](https://github.com/kubermatic/kubermatic/issues/4987))
- The `authorized_keys` files on nodes are now updated whenever the SSH keys for a cluster are changed ([#4531](https://github.com/kubermatic/kubermatic/issues/4531))
- Added support for custom CA for OpenID provider in Kubermatic API. ([#4994](https://github.com/kubermatic/kubermatic/issues/4994))
- Added user settings panel. ([#1738](https://github.com/kubermatic/dashboard/issues/1738))
- Added cluster addon UI
- MachineDeployments can now be configured to enable dynamic kubelet config ([#4946](https://github.com/kubermatic/kubermatic/issues/4946))
- Added RBAC management functionality to UI ([#1815](https://github.com/kubermatic/dashboard/issues/1815))
- Added RedHat Enterprise Linux as an OS option ([#669](https://github.com/kubermatic/machine-controller/issues/669))
- Added SUSE Linux Enterprise Server as an OS option ([#659](https://github.com/kubermatic/machine-controller/issues/659))

### Cloud providers

- Openstack: A bug that caused cluster reconciliation to fail if the controller crashed at the wrong time was fixed ([#4754](https://github.com/kubermatic/kubermatic/issues/4754)
- Openstack: New Kubernetes 1.16+ clusters use the external Cloud Controller Manager and CSI by default ([#4756](https://github.com/kubermatic/kubermatic/issues/4756))
- vSphere: Fixed a bug that resulted in a faulty cloud config when using a non-default port ([#4562](https://github.com/kubermatic/kubermatic/issues/4562))
- vSphere: Fixed a bug which cased custom VM folder paths not to be put in cloud-configs ([#4737](https://github.com/kubermatic/kubermatic/issues/4737))
- vSphere: The robustness of machine reconciliation has been improved. ([#4651](https://github.com/kubermatic/kubermatic/issues/4651))
- vSphere: Added support for datastore clusters (#671)
- Azure: Node sizes are displayed in size dropdown when creating/updating a node deployment ([#1908](https://github.com/kubermatic/dashboard/issues/1908))
- GCP: Networks are fetched from API now ([#1913](https://github.com/kubermatic/dashboard/issues/1913))

### Bugfixes

- Fixed parsing Kibana's logs in Fluent-Bit ([#4544](https://github.com/kubermatic/kubermatic/issues/4544))
- Fixed master-controller failing to create project-label-synchronizer controllers. ([#4577](https://github.com/kubermatic/kubermatic/issues/4577))
- Fixed broken NodePort-Proxy for user clusters with LoadBalancer expose strategy. ([#4590](https://github.com/kubermatic/kubermatic/issues/4590))
- Fixed cluster namespaces being stuck in Terminating state when deleting a cluster. ([#4619](https://github.com/kubermatic/kubermatic/issues/4619))
- Fixed Seed Validation Webhook rejecting new Seeds in certain situations ([#4662](https://github.com/kubermatic/kubermatic/issues/4662))
- A panic that could occur on clusters that lack both credentials and a credentialsSecret was fixed. ([#4742](https://github.com/kubermatic/kubermatic/issues/4742))
- A bug that occasionally resulted in a `Error: no matches for kind "MachineDeployment" in version "cluster.k8s.io/v1alpha1"` visible in the UI was fixed. ([#4870](https://github.com/kubermatic/kubermatic/issues/4870))
- A memory leak in the port-forwarding of the Kubernetes dashboard and Openshift console endpoints was fixed ([#4879](https://github.com/kubermatic/kubermatic/issues/4879))
- Fixed a bug that could result in 403 errors during cluster creation when using the BringYourOwn provider ([#4892](https://github.com/kubermatic/kubermatic/issues/4892))
- Fixed a bug that prevented clusters in working seeds from being listed in the dashboard if any other seed was unreachable. ([#4961](https://github.com/kubermatic/kubermatic/issues/4961))
- Prevented removing system labels during cluster edit ([#4986](https://github.com/kubermatic/kubermatic/issues/4986))
- Fixed FluentbitManyRetries Prometheus alert being too sensitive to harmless backpressure. ([#5011](https://github.com/kubermatic/kubermatic/issues/5011))
- Fixed deleting user-selectable addons from clusters. ([#5022](https://github.com/kubermatic/kubermatic/issues/5022))
- Fixed node name validation while creating clusters and node deployments ([#1783](https://github.com/kubermatic/dashboard/issues/1783))

### UI

- **ACTION REQUIRED:** Added logos and descriptions for the addons. In order to see the logos and descriptions addons have to be configured with AddonConfig CRDs with the same names as addons. ([#1824](https://github.com/kubermatic/dashboard/issues/1824))
- **ACTION REQUIRED:** Added application settings view. Some of the settings were moved from config map to the `KubermaticSettings` CRD. In order to use them in the UI it is required to manually update the CRD or do it from newly added UI. ([#1772](https://github.com/kubermatic/dashboard/issues/1772))
- Fixed label form validator. ([#1710](https://github.com/kubermatic/dashboard/issues/1710))
- Removed `Edit Settings` option from cluster detail view and instead combine everything under `Edit Cluster`. ([#1718](https://github.com/kubermatic/dashboard/issues/1718))
- Enabled edit options for kubeAdm ([#1735](https://github.com/kubermatic/dashboard/issues/1735))
- Switched flag proportions to 4:3. ([#1742](https://github.com/kubermatic/dashboard/issues/1742))
- Added new project view ([#1766](https://github.com/kubermatic/dashboard/issues/1766))
- Added custom links to admin settings. ([#1800](https://github.com/kubermatic/dashboard/issues/1800))
- Blocked option to edit cluster labels inherited from the project. ([#1801](https://github.com/kubermatic/dashboard/issues/1801))
- Moved pod security policy configuration to the edit cluster dialog. ([#1837](https://github.com/kubermatic/dashboard/issues/1837))
- Restyled some elements in the admin panel. ([#1850](https://github.com/kubermatic/dashboard/issues/1850))
- Added separate save indicators for custom links in the admin panel. ([#1862](https://github.com/kubermatic/dashboard/issues/1862))

### Addons

- The dashboard addon was removed as it's now deployed in the seed and can be used via its proxy endpoint ([#4567](https://github.com/kubermatic/kubermatic/issues/4567))
- Added default namespace/cluster roles for addons ([#4695](https://github.com/kubermatic/kubermatic/issues/4695))
- Introduced addon configurations. ([#4702](https://github.com/kubermatic/kubermatic/issues/4702))
- Fixed addon config get and list endpoints. ([#4734](https://github.com/kubermatic/kubermatic/issues/4734))
- Added forms for addon variables. ([#1846](https://github.com/kubermatic/dashboard/issues/1846))

### Misc

- **ACTION REQUIRED:** Updated cert-manager to 0.12.0. This requires a full reinstall of the chart. See https://cert-manager.io/docs/installation/upgrading/upgrading-0.10-0.11/ ([#4857](https://github.com/kubermatic/kubermatic/issues/4857))
- Updated Alertmanager to 0.20.0 ([#4864](https://github.com/kubermatic/kubermatic/issues/4864))
- Update Kubernetes Dashboard to v2.0.0-rc3 ([#5015](https://github.com/kubermatic/kubermatic/issues/5015))
- Updated Dex to v2.12.0 ([#4869](https://github.com/kubermatic/kubermatic/issues/4869))
- The envoy version used by the nodeport-proxy was updated to v1.12.2 ([#4865](https://github.com/kubermatic/kubermatic/issues/4865))
- Etcd was upgraded to 3.4 for 1.17+ clusters ([#4856](https://github.com/kubermatic/kubermatic/issues/4856))
- Updated Grafana to 6.5.2 ([#4858](https://github.com/kubermatic/kubermatic/issues/4858))
- Updated karma to 0.52 ([#4859](https://github.com/kubermatic/kubermatic/issues/4859))
- Updated kube-state-metrics to 1.8.0 ([#4860](https://github.com/kubermatic/kubermatic/issues/4860))
- Updated machine-controller to v1.10.0 ([#5070](https://github.com/kubermatic/kubermatic/issues/5070))
  - Added support for EBS volume encryption ([#663](https://github.com/kubermatic/machine-controller/issues/663))
  - kubelet sets initial machine taints via --register-with-taints ([#664](https://github.com/kubermatic/machine-controller/issues/664))
  - Moved deprecated kubelet flags into config file ([#667](https://github.com/kubermatic/machine-controller/issues/667))
  - Enabled swap accounting for Ubuntu deployments ([#666](https://github.com/kubermatic/machine-controller/issues/666))
- Updated nginx-ingress-controller to v0.28.0 ([#4999](https://github.com/kubermatic/kubermatic/issues/4999))
- Updated Minio to RELEASE.2019-10-12T01-39-57Z ([#4868](https://github.com/kubermatic/kubermatic/issues/4868))
- Updated Prometheus to 2.14 in Seed and User clusters ([#4684](https://github.com/kubermatic/kubermatic/issues/4684))
- Updated Thanos to 0.8.1 ([#4549](https://github.com/kubermatic/kubermatic/issues/4549))
- An email-restricted Datacenter can now have multiple email domains specified. ([#4643](https://github.com/kubermatic/kubermatic/issues/4643))
- Add fluent-bit Grafana dashboard ([#4545](https://github.com/kubermatic/kubermatic/issues/4545))
- Updated Dex page styling. ([#4632](https://github.com/kubermatic/kubermatic/issues/4632))
- Openshift: added metrics-server ([#4671](https://github.com/kubermatic/kubermatic/issues/4671))
- For new clusters, the Kubelet port 12050 is not exposed publicly anymore ([#4703](https://github.com/kubermatic/kubermatic/issues/4703))
- The cert-manager Helm chart now creates global ClusterIssuers for Let's Encrypt. ([#4732](https://github.com/kubermatic/kubermatic/issues/4732))
- Added migration for cluster user labels ([#4744](https://github.com/kubermatic/kubermatic/issues/4744))
- Fixed seed-proxy controller not working in namespaces other than `kubermatic`. ([#4775](https://github.com/kubermatic/kubermatic/issues/4775))
- The docker logs on the nodes now get rotated via the new `logrotate` addon ([#4813](https://github.com/kubermatic/kubermatic/issues/4813))
- Made node-exporter an optional addon. ([#4832](https://github.com/kubermatic/kubermatic/issues/4832))
- Added parent cluster readable name to default worker names. ([#4839](https://github.com/kubermatic/kubermatic/issues/4839))
- The QPS settings of Kubeletes can now be configured per-cluster using addon Variables ([#4854](https://github.com/kubermatic/kubermatic/issues/4854))
- Access to Kubernetes Dashboard can be now enabled/disabled by the global settings. ([#4889](https://github.com/kubermatic/kubermatic/issues/4889))
- Added support for dynamic presets ([#4903](https://github.com/kubermatic/kubermatic/issues/4903))
- Presets can now be filtered by datacenter ([#4991](https://github.com/kubermatic/kubermatic/issues/4991))
- Revoking the viewer token is possible via UI now. ([#1708](https://github.com/kubermatic/dashboard/issues/1708))
