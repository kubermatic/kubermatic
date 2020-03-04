### [v2.13.1]()


- Fixed swagger and API client for ssh key creation. [#5069](https://github.com/kubermatic/kubermatic/issues/5069) ([kdomanski](https://github.com/kdomanski))
- Added Kubernetes v1.15.10, v1.16.7, v1.17.3 [#5102](https://github.com/kubermatic/kubermatic/issues/5102) ([kdomanski](https://github.com/kdomanski))
- AddonConfig's shortDescription field is now used in the accessible addons overview. [#2050](https://github.com/kubermatic/dashboard-v2/issues/2050) ([maciaszczykm](https://github.com/maciaszczykm))




### [v2.13.0]()


Supported Kubernetes versions:
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

**Major changes:**
- End-of-Life Kubernetes v1.14 is no longer supported. [#4987](https://github.com/kubermatic/kubermatic/issues/4987) ([kdomanski](https://github.com/kdomanski))
- The `authorized_keys` files on nodes are now updated whenever the SSH keys for a cluster are changed [#4531](https://github.com/kubermatic/kubermatic/issues/4531) ([moadqassem](https://github.com/moadqassem))
- Added support for custom CA for OpenID provider in Kubermatic API. [#4994](https://github.com/kubermatic/kubermatic/issues/4994) ([xrstf](https://github.com/xrstf))
- Added user settings panel. [#1738](https://github.com/kubermatic/dashboard-v2/issues/1738) ([maciaszczykm](https://github.com/maciaszczykm))
- Added cluster addon UI
- MachineDeployments can now be configured to enable dynamic kubelet config [#4946](https://github.com/kubermatic/kubermatic/issues/4946) ([kdomanski](https://github.com/kdomanski))
- Added RBAC management functionality to UI [#1815](https://github.com/kubermatic/dashboard-v2/issues/1815) ([kgroschoff](https://github.com/kgroschoff))
- Added RedHat Enterprise Linux as an OS option (#669)
- Added SUSE Linux Enterprise Server as an OS option (#659)

**Cloud providers:**
- Openstack: A bug that caused cluster reconciliation to fail if the controller crashed at the wrong time was fixed [#4754](https://github.com/kubermatic/kubermatic/issues/4754) ([alvaroaleman](https://github.com/alvaroaleman))
- Openstack: New Kubernetes 1.16&#43; clusters use the external Cloud Controller Manager and CSI by default [#4756](https://github.com/kubermatic/kubermatic/issues/4756) ([alvaroaleman](https://github.com/alvaroaleman))
- vSphere: Fixed a bug that resulted in a faulty cloud config when using a non-default port [#4562](https://github.com/kubermatic/kubermatic/issues/4562) ([alvaroaleman](https://github.com/alvaroaleman))
- vSphere: Fixed a bug which cased custom VM folder paths not to be put in cloud-configs [#4737](https://github.com/kubermatic/kubermatic/issues/4737) ([kdomanski](https://github.com/kdomanski))
- vSphere: The robustness of machine reconciliation has been improved. [#4651](https://github.com/kubermatic/kubermatic/issues/4651) ([alvaroaleman](https://github.com/alvaroaleman))
- vSphere: Added support for datastore clusters (#671)
- Azure: Node sizes are displayed in size dropdown when creating/updating a node deployment [#1908](https://github.com/kubermatic/dashboard-v2/issues/1908) ([bashofmann](https://github.com/bashofmann))
- GCP: Networks are fetched from API now [#1913](https://github.com/kubermatic/dashboard-v2/issues/1913) ([kgroschoff](https://github.com/kgroschoff))


**Bugfixes:**
- Fixed parsing Kibana's logs in Fluent-Bit [#4544](https://github.com/kubermatic/kubermatic/issues/4544) ([xrstf](https://github.com/xrstf))
- Fixed master-controller failing to create project-label-synchronizer controllers. [#4577](https://github.com/kubermatic/kubermatic/issues/4577) ([xrstf](https://github.com/xrstf))
- Fixed broken NodePort-Proxy for user clusters with LoadBalancer expose strategy. [#4590](https://github.com/kubermatic/kubermatic/issues/4590) ([xrstf](https://github.com/xrstf))
- Fixed cluster namespaces being stuck in Terminating state when deleting a cluster. [#4619](https://github.com/kubermatic/kubermatic/issues/4619) ([xrstf](https://github.com/xrstf))
- Fixed Seed Validation Webhook rejecting new Seeds in certain situations [#4662](https://github.com/kubermatic/kubermatic/issues/4662) ([xrstf](https://github.com/xrstf))
- A panic that could occur on clusters that lack both credentials and a credentialsSecret was fixed. [#4742](https://github.com/kubermatic/kubermatic/issues/4742) ([alvaroaleman](https://github.com/alvaroaleman))
- A bug that occasionally resulted in a `Error: no matches for kind "MachineDeployment" in version "cluster.k8s.io/v1alpha1"` visible in the UI was fixed. [#4870](https://github.com/kubermatic/kubermatic/issues/4870) ([alvaroaleman](https://github.com/alvaroaleman))
- A memory leak in the port-forwarding of the Kubernetes dashboard and Openshift console endpoints was fixed [#4879](https://github.com/kubermatic/kubermatic/issues/4879) ([alvaroaleman](https://github.com/alvaroaleman))
- Fixed a bug that could result in 403 errors during cluster creation when using the BringYourOwn provider [#4892](https://github.com/kubermatic/kubermatic/issues/4892) ([alvaroaleman](https://github.com/alvaroaleman))
- Fixed a bug that prevented clusters in working seeds from being listed in the dashboard if any other seed was unreachable. [#4961](https://github.com/kubermatic/kubermatic/issues/4961) ([xrstf](https://github.com/xrstf))
- Prevented removing system labels during cluster edit [#4986](https://github.com/kubermatic/kubermatic/issues/4986) ([zreigz](https://github.com/zreigz))
- Fixed FluentbitManyRetries Prometheus alert being too sensitive to harmless backpressure. [#5011](https://github.com/kubermatic/kubermatic/issues/5011) ([xrstf](https://github.com/xrstf))
- Fixed deleting user-selectable addons from clusters. [#5022](https://github.com/kubermatic/kubermatic/issues/5022) ([xrstf](https://github.com/xrstf))
- Fixed node name validation while creating clusters and node deployments [#1783](https://github.com/kubermatic/dashboard-v2/issues/1783) ([chrkl](https://github.com/chrkl))

**UI:**
- ACTION REQUIRED: Added logos and descriptions for the addons. In order to see the logos and descriptions addons have to be configured with AddonConfig CRDs with the same names as addons. [#1824](https://github.com/kubermatic/dashboard-v2/issues/1824) ([maciaszczykm](https://github.com/maciaszczykm))
- ACTION REQUIRED: Added application settings view. Some of the settings were moved from config map to the `KubermaticSettings` CRD. In order to use them in the UI it is required to manually update the CRD or do it from newly added UI. [#1772](https://github.com/kubermatic/dashboard-v2/issues/1772) ([maciaszczykm](https://github.com/maciaszczykm))
- Fixed label form validator. [#1710](https://github.com/kubermatic/dashboard-v2/issues/1710) ([maciaszczykm](https://github.com/maciaszczykm))
- Removed `Edit Settings` option from cluster detail view and instead combine everything under `Edit Cluster`. [#1718](https://github.com/kubermatic/dashboard-v2/issues/1718) ([kgroschoff](https://github.com/kgroschoff))
- Enabled edit options for kubeAdm [#1735](https://github.com/kubermatic/dashboard-v2/issues/1735) ([kgroschoff](https://github.com/kgroschoff))
- Switched flag proportions to 4:3. [#1742](https://github.com/kubermatic/dashboard-v2/issues/1742) ([maciaszczykm](https://github.com/maciaszczykm))
- Added new project view [#1766](https://github.com/kubermatic/dashboard-v2/issues/1766) ([kgroschoff](https://github.com/kgroschoff))
- Added custom links to admin settings. [#1800](https://github.com/kubermatic/dashboard-v2/issues/1800) ([maciaszczykm](https://github.com/maciaszczykm))
- Blocked option to edit cluster labels inherited from the project. [#1801](https://github.com/kubermatic/dashboard-v2/issues/1801) ([floreks](https://github.com/floreks))
- Moved pod security policy configuration to the edit cluster dialog. [#1837](https://github.com/kubermatic/dashboard-v2/issues/1837) ([maciaszczykm](https://github.com/maciaszczykm))
- Restyled some elements in the admin panel. [#1850](https://github.com/kubermatic/dashboard-v2/issues/1850) ([maciaszczykm](https://github.com/maciaszczykm))
- Added separate save indicators for custom links in the admin panel. [#1862](https://github.com/kubermatic/dashboard-v2/issues/1862) ([maciaszczykm](https://github.com/maciaszczykm))

**Addons:**
- The dashboard addon was removed as it's now deployed in the seed and can be used via its proxy endpoint [#4567](https://github.com/kubermatic/kubermatic/issues/4567) ([alvaroaleman](https://github.com/alvaroaleman))
- Added default namespace/cluster roles for addons [#4695](https://github.com/kubermatic/kubermatic/issues/4695) ([zreigz](https://github.com/zreigz))
- Introduced addon configurations. [#4702](https://github.com/kubermatic/kubermatic/issues/4702) ([maciaszczykm](https://github.com/maciaszczykm))
- Fixed addon config get and list endpoints. [#4734](https://github.com/kubermatic/kubermatic/issues/4734) ([maciaszczykm](https://github.com/maciaszczykm))
- Added forms for addon variables. [#1846](https://github.com/kubermatic/dashboard-v2/issues/1846) ([maciaszczykm](https://github.com/maciaszczykm))

**Misc:**
- ACTION REQUIRED: Updated cert-manager to 0.12.0. This requires a full reinstall of the chart. See https://cert-manager.io/docs/installation/upgrading/upgrading-0.10-0.11/ [#4857](https://github.com/kubermatic/kubermatic/issues/4857) ([xrstf](https://github.com/xrstf))
- Updated Alertmanager to 0.20.0 [#4864](https://github.com/kubermatic/kubermatic/issues/4864) ([xrstf](https://github.com/xrstf))
- Update Kubernetes Dashboard to v2.0.0-rc3 [#5015](https://github.com/kubermatic/kubermatic/issues/5015) ([floreks](https://github.com/floreks))
- Updated Dex to v2.12.0 [#4869](https://github.com/kubermatic/kubermatic/issues/4869) ([xrstf](https://github.com/xrstf))
- The envoy version used by the nodeport-proxy was updated to v1.12.2 [#4865](https://github.com/kubermatic/kubermatic/issues/4865) ([alvaroaleman](https://github.com/alvaroaleman))
- Etcd was upgraded to 3.4 for 1.17&#43; clusters [#4856](https://github.com/kubermatic/kubermatic/issues/4856) ([alvaroaleman](https://github.com/alvaroaleman))
- Updated Grafana to 6.5.2 [#4858](https://github.com/kubermatic/kubermatic/issues/4858) ([xrstf](https://github.com/xrstf))
- Updated karma to 0.52 [#4859](https://github.com/kubermatic/kubermatic/issues/4859) ([xrstf](https://github.com/xrstf))
- Updated kube-state-metrics to 1.8.0 [#4860](https://github.com/kubermatic/kubermatic/issues/4860) ([xrstf](https://github.com/xrstf))
- Updated machine-controller to v1.10.0 [#5070](https://github.com/kubermatic/kubermatic/issues/5070) ([kdomanski](https://github.com/kdomanski))
  - Added support for EBS volume encryption (#663)
  - kubelet sets intial machine taints via --register-with-taints (#664)
  - Moved deprecated kubelet flags into config file (#667)
  - Enabled swap accounting for Ubuntu deployments (#666)
- Updated nginx-ingress-controller to v0.28.0 [#4999](https://github.com/kubermatic/kubermatic/issues/4999) ([kdomanski](https://github.com/kdomanski))
- Updated Minio to RELEASE.2019-10-12T01-39-57Z [#4868](https://github.com/kubermatic/kubermatic/issues/4868) ([xrstf](https://github.com/xrstf))
- Updated Prometheus to 2.14 in Seed and User clusters [#4684](https://github.com/kubermatic/kubermatic/issues/4684) ([xrstf](https://github.com/xrstf))
- Updated Thanos to 0.8.1 [#4549](https://github.com/kubermatic/kubermatic/issues/4549) ([xrstf](https://github.com/xrstf))
- An email-restricted Datacenter can now have multiple email domains specified. [#4643](https://github.com/kubermatic/kubermatic/issues/4643) ([kdomanski](https://github.com/kdomanski))
- Add fluent-bit Grafana dashboard [#4545](https://github.com/kubermatic/kubermatic/issues/4545) ([xrstf](https://github.com/xrstf))                    
- Updated Dex page styling. [#4632](https://github.com/kubermatic/kubermatic/issues/4632) ([maciaszczykm](https://github.com/maciaszczykm))
- Openshift: added metrics-server [#4671](https://github.com/kubermatic/kubermatic/issues/4671) ([kron4eg](https://github.com/kron4eg))
- For new clusters, the Kubelet port 12050 is not exposed publicly anymore [#4703](https://github.com/kubermatic/kubermatic/issues/4703) ([bashofmann](https://github.com/bashofmann))
- The cert-manager Helm chart now creates global ClusterIssuers for Let'&#39;'s Encrypt. [#4732](https://github.com/kubermatic/kubermatic/issues/4732) ([xrstf](https://github.com/xrstf))
- Added migration for cluster user labels [#4744](https://github.com/kubermatic/kubermatic/issues/4744) ([zreigz](https://github.com/zreigz))
- Fixed seed-proxy controller not working in namespaces other than `kubermatic`. [#4775](https://github.com/kubermatic/kubermatic/issues/4775) ([xrstf](https://github.com/xrstf))
- The docker logs on the nodes now get rotated via the new `logrotate` addon [#4813](https://github.com/kubermatic/kubermatic/issues/4813) ([moadqassem](https://github.com/moadqassem))
- Made node-exporter an optional addon. [#4832](https://github.com/kubermatic/kubermatic/issues/4832) ([maciaszczykm](https://github.com/maciaszczykm))
- Added parent cluster readable name to default worker names. [#4839](https://github.com/kubermatic/kubermatic/issues/4839) ([maciaszczykm](https://github.com/maciaszczykm))
- The QPS settings of Kubeletes can now be configured per-cluster using addon Variables [#4854](https://github.com/kubermatic/kubermatic/issues/4854) ([kdomanski](https://github.com/kdomanski))
- Access to Kubernetes Dashboard can be now enabled/disabled by the global settings. [#4889](https://github.com/kubermatic/kubermatic/issues/4889) ([floreks](https://github.com/floreks))
- Added support for dynamic presets [#4903](https://github.com/kubermatic/kubermatic/issues/4903) ([zreigz](https://github.com/zreigz))
- Presets can now be filtered by datacenter [#4991](https://github.com/kubermatic/kubermatic/issues/4991) ([zreigz](https://github.com/zreigz))
- Revoking the viewer token is possible via UI now. [#1708](https://github.com/kubermatic/dashboard-v2/issues/1708) ([kgroschoff](https://github.com/kgroschoff))




### [v2.12.6]()


**Misc:**

- System labels can no longer be removed by the user. [#4983](https://github.com/kubermatic/kubermatic/issues/4983) ([zreigz](https://github.com/zreigz))
- End-of-Life Kubernetes v1.14 is no longer supported. [#4988](https://github.com/kubermatic/kubermatic/issues/4988) ([kdomanski](https://github.com/kdomanski))
- Added Kubernetes v1.15.7, v1.15.9, v1.16.4, v1.16.6 [#4995](https://github.com/kubermatic/kubermatic/issues/4995) ([kdomanski](https://github.com/kdomanski))




### [v2.12.5]()


- A bug that occasionally resulted in a `Error: no matches for kind "MachineDeployment" in version "cluster.k8s.io/v1alpha1"` visible in the UI was fixed. [#4870](https://github.com/kubermatic/kubermatic/issues/4870) ([alvaroaleman](https://github.com/alvaroaleman))
- A memory leak in the port-forwarding of the Kubernetes dashboard and Openshift console endpoints was fixed [#4879](https://github.com/kubermatic/kubermatic/issues/4879) ([alvaroaleman](https://github.com/alvaroaleman))
- Enabled edit options for kubeAdm [#1873](https://github.com/kubermatic/dashboard-v2/issues/1873) ([kgroschoff](https://github.com/kgroschoff))




### [v2.12.4]()


- Fixed an issue with adding new node deployments on Openstack [#1836](https://github.com/kubermatic/dashboard-v2/issues/1836) ([floreks](https://github.com/floreks))
- Added migration for cluster user labels [#4744](https://github.com/kubermatic/kubermatic/issues/4744) ([zreigz](https://github.com/zreigz))
- Added Kubernetes `v1.14.9`, `v1.15.6` and `v1.16.3` [#4752](https://github.com/kubermatic/kubermatic/issues/4752) ([kdomanski](https://github.com/kdomanski))
- Openstack: A bug that caused cluster reconciliation to fail if the controller crashed at the wrong time was fixed [#4754](https://github.com/kubermatic/kubermatic/issues/4754) ([alvaroaleman](https://github.com/alvaroaleman))




### [v2.12.3]()


- Fixed extended cluster options not being properly applied [#1812](https://github.com/kubermatic/dashboard-v2/issues/1812) ([kgroschoff](https://github.com/kgroschoff))
- A panic that could occur on clusters that lack both credentials and a credentialsSecret was fixed. [#4742](https://github.com/kubermatic/kubermatic/issues/4742) ([alvaroaleman](https://github.com/alvaroaleman))


### [v2.12.2]()


- The robustness of vSphere machine reconciliation has been improved. [#4651](https://github.com/kubermatic/kubermatic/issues/4651) ([alvaroaleman](https://github.com/alvaroaleman))
- Fixe Seed Validation Webhook rejecting new Seeds in certain situations [#4662](https://github.com/kubermatic/kubermatic/issues/4662) ([xrstf](https://github.com/xrstf))
- Rolled nginx-ingress-controller back to 0.25.1 to fix SSL redirect issues. [#4693](https://github.com/kubermatic/kubermatic/issues/4693) ([xrstf](https://github.com/xrstf))




### [v2.12.1]()


- VSphere: Fixed a bug that resulted in a faulty cloud config when using a non-default port [#4562](https://github.com/kubermatic/kubermatic/issues/4562) ([alvaroaleman](https://github.com/alvaroaleman))
- Fixed master-controller failing to create project-label-synchronizer controllers. [#4577](https://github.com/kubermatic/kubermatic/issues/4577) ([xrstf](https://github.com/xrstf))
- Fixed broken NodePort-Proxy for user clusters with LoadBalancer expose strategy. [#4590](https://github.com/kubermatic/kubermatic/issues/4590) ([xrstf](https://github.com/xrstf))




### [v2.12.0]()


Supported Kubernetes versions:
- `1.14.8`
- `1.15.5`
- `1.16.2`
- Openshift `v4.1.18` preview


**Major new features:**
- Kubernetes 1.16 support was added [#4313](https://github.com/kubermatic/kubermatic/issues/4313) ([alvaroaleman](https://github.com/alvaroaleman))
- It is now possible to also configure automatic node updates by setting `automaticNodeUpdate: true` in the `updates.yaml`. This option implies `automatic: true` as node versions must not be newer than the version of the corresponding controlplane. [#4258](https://github.com/kubermatic/kubermatic/issues/4258) ([alvaroaleman](https://github.com/alvaroaleman))
- Cloud credentials can now be configured as presets [#3723](https://github.com/kubermatic/kubermatic/issues/3723) ([zreigz](https://github.com/zreigz))
- Access to datacenters can now be restricted based on the user&#39;s email domain. [#4470](https://github.com/kubermatic/kubermatic/issues/4470) ([kdomanski](https://github.com/kdomanski))
- It is now possible to open the Kubernetes Dashboard from the Kubermatic UI. [#4460](https://github.com/kubermatic/kubermatic/issues/4460) ([floreks](https://github.com/floreks))
- An option to use AWS Route53 DNS validation was added to the `certs` chart. [#4397](https://github.com/kubermatic/kubermatic/issues/4397) ([alvaroaleman](https://github.com/alvaroaleman))
- Added possibility to add labels to projects and clusters and have these labels inherited by node objects.
- Added support for Kubernetes audit logging [#4151](https://github.com/kubermatic/kubermatic/issues/4151) ([eqrx](https://github.com/eqrx))
- Connect button on cluster details will now open Kubernetes Dashboard/Openshift Console [#1667](https://github.com/kubermatic/dashboard-v2/issues/1667) ([floreks](https://github.com/floreks))
- Pod Security Policies can now be enabled [#4062](https://github.com/kubermatic/kubermatic/issues/4062) ([bashofmann](https://github.com/bashofmann))
- Added support for optional cluster addons [#1683](https://github.com/kubermatic/dashboard-v2/issues/1683) ([maciaszczykm](https://github.com/maciaszczykm)) 

**Installation and updating:**
- ACTION REQUIRED: the `zone_character` field must be removed from all AWS datacenters in `datacenters.yaml` [#3986](https://github.com/kubermatic/kubermatic/issues/3986) ([kdomanski](https://github.com/kdomanski))
- ACTION REQUIRED: The default number of apiserver replicas was increased to 2. You can revert to the old behavior by setting `.Kubermatic.apiserverDefaultReplicas` in the `values.yaml` [#3885](https://github.com/kubermatic/kubermatic/issues/3885) ([alvaroaleman](https://github.com/alvaroaleman))
- ACTION REQUIRED: The literal credentials on the `Cluster` object are being deprecated in favor of storing them in a secret. If you have addons that use credentials, replace `.Cluster.Spec.Cloud` with `.Credentials`. [#4463](https://github.com/kubermatic/kubermatic/issues/4463) ([alvaroaleman](https://github.com/alvaroaleman))
- ACTION REQUIRED: Kubermatic now doesn&#39;t accept unknown keys in its config files anymore and will crash if an unknown key is present
- ACTION REQUIRED: BYO datacenters now need to be specific in the `datacenters.yaml` with a value of `{}`, e.G `bringyourown: {}` [#3794](https://github.com/kubermatic/kubermatic/issues/3794) ([alvaroaleman](https://github.com/alvaroaleman))
- ACTION REQUIRED: Velero does not backup Prometheus, Elasticsearch and Minio by default anymore. [#4482](https://github.com/kubermatic/kubermatic/issues/4482) ([xrstf](https://github.com/xrstf))
- ACTION REQUIRED: On AWS, the nodeport-proxy will be recreated as NLB. DNS entries must be updated to point to the new LB. [#3840](https://github.com/kubermatic/kubermatic/pull/3840) ([mrIncompetent](https://github.com/mrIncompetent))
- The deprecated nodePortPoxy key for Helm values has been removed. [#3830](https://github.com/kubermatic/kubermatic/issues/3830) ([xrstf](https://github.com/xrstf))
- Support setting oidc authentication settings on cluster [#3751](https://github.com/kubermatic/kubermatic/issues/3751) ([bashofmann](https://github.com/bashofmann))
- The worker-count of controller-manager and master-controller are now configurable [#3918](https://github.com/kubermatic/kubermatic/issues/3918) ([bashofmann](https://github.com/bashofmann))
- master-controller-manager can now be deployed with multiple replicas [#4307](https://github.com/kubermatic/kubermatic/issues/4307) ([xrstf](https://github.com/xrstf))
- It is now possible to configure an http proxy on a Seed. This will result in the proxy being used for all control plane pods in that seed that talk to a cloudprovider and for all machines in that Seed, unless its overriden on Datacenter level. [#4459](https://github.com/kubermatic/kubermatic/issues/4459) ([alvaroaleman](https://github.com/alvaroaleman))
- The cert-manager Helm chart now allows configuring extra values for its controllers args and env vars. [#4398](https://github.com/kubermatic/kubermatic/issues/4398) ([alvaroaleman](https://github.com/alvaroaleman))
- A fix for CVE-2019-11253 for clusters that were created with a Kubernetes version &lt; 1.14 was deployed [#4520](https://github.com/kubermatic/kubermatic/issues/4520) ([alvaroaleman](https://github.com/alvaroaleman))

**Monitoring and logging:**
- Alertmanager&#39;s inhibition feature is now used to hide consequential alerts. [#3833](https://github.com/kubermatic/kubermatic/issues/3833) ([xrstf](https://github.com/xrstf))
- Removed cluster owner name and email labels from kubermatic_cluster_info metric to prevent leaking PII [#3854](https://github.com/kubermatic/kubermatic/issues/3854) ([xrstf](https://github.com/xrstf))
- New Prometheus metrics kubermatic_addon_created kubermatic_addon_deleted
- New alert KubermaticAddonDeletionTakesTooLong [#3941](https://github.com/kubermatic/kubermatic/issues/3941) ([bashofmann](https://github.com/bashofmann))
- FluentBit will now collect the journald logs [#4001](https://github.com/kubermatic/kubermatic/issues/4001) ([mrIncompetent](https://github.com/mrIncompetent))
- FluentBit can now collect the kernel messages [#4007](https://github.com/kubermatic/kubermatic/issues/4007) ([mrIncompetent](https://github.com/mrIncompetent))
- FluentBit now always sets the node name in logs [#4010](https://github.com/kubermatic/kubermatic/issues/4010) ([mrIncompetent](https://github.com/mrIncompetent))
- Added new KubermaticClusterPaused alert with &#34;none&#34; severity for inhibiting alerts from paused clusters [#3846](https://github.com/kubermatic/kubermatic/issues/3846) ([xrstf](https://github.com/xrstf))
- Removed Helm-based templating in Grafana dashboards [#4475](https://github.com/kubermatic/kubermatic/issues/4475) ([xrstf](https://github.com/xrstf))
- Added type label (kubernetes/openshift) to kubermatic_cluster_info metric. [#4452](https://github.com/kubermatic/kubermatic/issues/4452) ([xrstf](https://github.com/xrstf))
- Added metrics endpoint for cluster control plane:GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/metrics [#4208](https://github.com/kubermatic/kubermatic/issues/4208) ([zreigz](https://github.com/zreigz))
- Added a new endpoint for node deployment metrics:GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodedeployments/{nodedeployment_id}/metrics [#4176](https://github.com/kubermatic/kubermatic/issues/4176) ([zreigz](https://github.com/zreigz))

**Cloud providers:**
- Openstack: A bug that could result in many securtiy groups being created when the creation of security group rules failed was fixed [#3848](https://github.com/kubermatic/kubermatic/issues/3848) ([alvaroaleman](https://github.com/alvaroaleman))
- Openstack: Fixed a bug preventing an interrupted cluster creation from being resumed. [#4476](https://github.com/kubermatic/kubermatic/issues/4476) ([kdomanski](https://github.com/kdomanski))
- Openstack: Disk size of nodes is now configurable [#4153](https://github.com/kubermatic/kubermatic/issues/4153) ([bashofmann](https://github.com/bashofmann))
- Openstack: Added a security group API compatibility workaround for very old versions of Openstack. [#4479](https://github.com/kubermatic/kubermatic/issues/4479) ([kdomanski](https://github.com/kdomanski))
- Openstack: Fixed fetching the list of tenants on some OpenStack configurations with one region [#4182](https://github.com/kubermatic/kubermatic/issues/4182) ([zreigz](https://github.com/zreigz))
- Openstack: Added support for Project ID to the wizard [#1386](https://github.com/kubermatic/dashboard-v2/issues/1386) ([floreks](https://github.com/floreks))
- Openstack: The project name can now be provided manually [#1423](https://github.com/kubermatic/dashboard-v2/issues/1423) ([floreks](https://github.com/floreks))
- Openstack: Fixed API usage for datacenters with only one region [#4538](https://github.com/kubermatic/kubermatic/issues/4538) ([zreigz](https://github.com/zreigz))
- Openstack: Fixed a bug that resulted in the router not being attached to the subnet when the subnet was manually created [#4521](https://github.com/kubermatic/kubermatic/issues/4521) ([alvaroaleman](https://github.com/alvaroaleman))
- AWS: MachineDeployments can now be created in any availability zone of the cluster&#39;s region [#3870](https://github.com/kubermatic/kubermatic/issues/3870) ([kdomanski](https://github.com/kdomanski))
- AWS: Reduced the role permissions for the control-plane &amp; worker role to the minimum [#3995](https://github.com/kubermatic/kubermatic/issues/3995) ([mrIncompetent](https://github.com/mrIncompetent))
- AWS: The subnet can now be selected [#1499](https://github.com/kubermatic/dashboard-v2/issues/1499) ([kgroschoff](https://github.com/kgroschoff))
- AWS: Setting `Control plane role (ARN)` now is possible [#1512](https://github.com/kubermatic/dashboard-v2/issues/1512) ([kgroschoff](https://github.com/kgroschoff))
- AWS: VM sizes are fetched from the API now. [#1513](https://github.com/kubermatic/dashboard-v2/issues/1513) ([maciaszczykm](https://github.com/maciaszczykm))
- AWS: Worker nodes can now be provisioned without a public IP. [#1591](https://github.com/kubermatic/dashboard-v2/issues/1591) ([maciaszczykm](https://github.com/maciaszczykm))
- GCP: machine and disk types are now fetched from GCP. [#1363](https://github.com/kubermatic/dashboard-v2/issues/1363) ([maciaszczykm](https://github.com/maciaszczykm))
- vSphere: the VM folder can now be configured
- Added support for KubeVirt provider. [#1608](https://github.com/kubermatic/dashboard-v2/issues/1608) ([maciaszczykm](https://github.com/maciaszczykm))

**Bugfixes:**
- A bug that sometimes resulted in the creation of the initial NodeDeployment failing was fixed [#3894](https://github.com/kubermatic/kubermatic/issues/3894) ([alvaroaleman](https://github.com/alvaroaleman))
- `kubeadm join` has been fixed for v1.15 clusters [#4161](https://github.com/kubermatic/kubermatic/issues/4161) ([kdomanski](https://github.com/kdomanski))
- Fixed a bug that could cause intermittent delays when using kubectl logs/exec with `exposeStrategy: LoadBalancer` [#4278](https://github.com/kubermatic/kubermatic/issues/4278) ([alvaroaleman](https://github.com/alvaroaleman))
- A bug that prevented node Labels, Taints and Annotations from getting applied correctly was fixed. [#4368](https://github.com/kubermatic/kubermatic/issues/4368) ([alvaroaleman](https://github.com/alvaroaleman))
- Fixed worker nodes provisioning for instances with a Kernel &gt;= 4.19 [#4178](https://github.com/kubermatic/kubermatic/issues/4178) ([alvaroaleman](https://github.com/alvaroaleman))
- Fixed an issue that kept clusters stuck if their creation didn&#39;t succeed and they got deleted with LB and/or PV cleanup enabled [#3973](https://github.com/kubermatic/kubermatic/issues/3973) ([alvaroaleman](https://github.com/alvaroaleman))
- Fixed an issue where deleted project owners would come back after a while [#4025](https://github.com/kubermatic/kubermatic/issues/4025) ([zreigz](https://github.com/zreigz))
- Enabling the OIDC feature flag in clusters has been fixed. [#4127](https://github.com/kubermatic/kubermatic/issues/4127) ([zreigz](https://github.com/zreigz))

**Misc:**
- The share cluster feature now allows to use groups, if passed by the IDP. All groups are prefixed with `oidc:` [#4244](https://github.com/kubermatic/kubermatic/issues/4244) ([alvaroaleman](https://github.com/alvaroaleman))
- The kube-proxy mode (ipvs/iptables) can now be configured. If not specified, it defaults to ipvs. [#4247](https://github.com/kubermatic/kubermatic/issues/4247) ([nikhita](https://github.com/nikhita))
- Addons can now read the AWS region  from the `kubermatic.io/aws-region` annotation on the cluster [#4434](https://github.com/kubermatic/kubermatic/issues/4434) ([alvaroaleman](https://github.com/alvaroaleman))
- Allow disabling of apiserver endpoint reconciling. [#4396](https://github.com/kubermatic/kubermatic/issues/4396) ([thz](https://github.com/thz))
- Allow cluster owner to manage RBACs from Kubermatic API [#4321](https://github.com/kubermatic/kubermatic/issues/4321) ([zreigz](https://github.com/zreigz))
- The default service CIDR for new clusters was increased and changed from 10.10.10.0/24 to 10.240.16.0/20 [#4227](https://github.com/kubermatic/kubermatic/issues/4227) ([alvaroaleman](https://github.com/alvaroaleman))
- Retries of the initial node deployment creation do not create an event anymore but continue to be logged at debug level. [#4226](https://github.com/kubermatic/kubermatic/issues/4226) ([alvaroaleman](https://github.com/alvaroaleman))
- Added option to enforce cluster cleanup in UI [#3966](https://github.com/kubermatic/kubermatic/issues/3966) ([kgroschoff](https://github.com/kgroschoff))
- Support PodSecurityPolicies in addons [#4174](https://github.com/kubermatic/kubermatic/issues/4174) ([bashofmann](https://github.com/bashofmann))
- Kubernetes versions affected by CVE-2019-9512 and CVE-2019-9514 have been dropped [#4113](https://github.com/kubermatic/kubermatic/issues/4113) ([kdomanski](https://github.com/kdomanski))
- Kubernetes versions affected by CVE-2019-11247 and CVE-2019-11249 have been dropped [#4066](https://github.com/kubermatic/kubermatic/issues/4066) ([kdomanski](https://github.com/kdomanski))
- Kubernetes 1.13 which is end-of-life has been removed. [#4327](https://github.com/kubermatic/kubermatic/issues/4327) ([kdomanski](https://github.com/kdomanski))
- Updated Alertmanager to 0.19 [#4340](https://github.com/kubermatic/kubermatic/issues/4340) ([xrstf](https://github.com/xrstf))
- Updated blackbox-exporter to 0.15.1 [#4341](https://github.com/kubermatic/kubermatic/issues/4341) ([xrstf](https://github.com/xrstf))
- Updated Canal to v3.8 [#3791](https://github.com/kubermatic/kubermatic/issues/3791) ([mrIncompetent](https://github.com/mrIncompetent))
- Updated cert-manager to 0.10.1 [#4407](https://github.com/kubermatic/kubermatic/issues/4407) ([xrstf](https://github.com/xrstf))
- Updated Dex to 2.19 [#4343](https://github.com/kubermatic/kubermatic/issues/4343) ([xrstf](https://github.com/xrstf))
- Updated Envoy to 1.11.1 [#4075](https://github.com/kubermatic/kubermatic/issues/4075) ([xrstf](https://github.com/xrstf))
- Updated etcd to 3.3.15 [#4199](https://github.com/kubermatic/kubermatic/issues/4199) ([bashofmann](https://github.com/bashofmann))
- Updated FluentBit to v1.2.2 [#4022](https://github.com/kubermatic/kubermatic/issues/4022) ([mrIncompetent](https://github.com/mrIncompetent))
- Updated Grafana to 6.3.5 [#4342](https://github.com/kubermatic/kubermatic/issues/4342) ([xrstf](https://github.com/xrstf))
- Updated helm-exporter to 0.4.2 [#4124](https://github.com/kubermatic/kubermatic/issues/4124) ([xrstf](https://github.com/xrstf))
- Updated kube-state-metrics to 1.7.2 [#4129](https://github.com/kubermatic/kubermatic/issues/4129) ([xrstf](https://github.com/xrstf))
- Updated Minio to 2019-09-18T21-55-05Z [#4339](https://github.com/kubermatic/kubermatic/issues/4339) ([xrstf](https://github.com/xrstf))
- Updated machine-controller to v1.5.6 [#4310](https://github.com/kubermatic/kubermatic/issues/4310) ([kdomanski](https://github.com/kdomanski))
- Updated nginx-ingress-controller to 0.26.1 [#4400](https://github.com/kubermatic/kubermatic/issues/4400) ([xrstf](https://github.com/xrstf))
- Updated Prometheus to 2.12.0 [#4131](https://github.com/kubermatic/kubermatic/issues/4131) ([xrstf](https://github.com/xrstf))
- Updated Velero to v1.1.0 [#4468](https://github.com/kubermatic/kubermatic/issues/4468) ([kron4eg](https://github.com/kron4eg))


**Dashboard:**
- Added Swagger UI for Kubermatic API [#1418](https://github.com/kubermatic/dashboard-v2/issues/1418) ([bashofmann](https://github.com/bashofmann))
- Redesign dialog to manage SSH keys on cluster [#1353](https://github.com/kubermatic/dashboard-v2/issues/1353) ([kgroschoff](https://github.com/kgroschoff))
- GCP zones are now fetched from API. [#1379](https://github.com/kubermatic/dashboard-v2/issues/1379) ([maciaszczykm](https://github.com/maciaszczykm))
- Redesign Wizard: Summary [#1409](https://github.com/kubermatic/dashboard-v2/issues/1409) ([kgroschoff](https://github.com/kgroschoff))
- Cluster type toggle in wizard is now hidden if only one cluster type is active [#1425](https://github.com/kubermatic/dashboard-v2/issues/1425) ([bashofmann](https://github.com/bashofmann))
- Disabled the possibility of adding new node deployments until the cluster is fully ready. [#1439](https://github.com/kubermatic/dashboard-v2/issues/1439) ([maciaszczykm](https://github.com/maciaszczykm))
- The cluster name is now editable from the dashboard [#1455](https://github.com/kubermatic/dashboard-v2/issues/1455) ([bashofmann](https://github.com/bashofmann))
- Added warning about node deployment changes that will recreate all nodes. [#1479](https://github.com/kubermatic/dashboard-v2/issues/1479) ([maciaszczykm](https://github.com/maciaszczykm))
- OIDC client id is now configurable [#1505](https://github.com/kubermatic/dashboard-v2/issues/1505) ([bashofmann](https://github.com/bashofmann))
- Replaced particles with a static background. [#1578](https://github.com/kubermatic/dashboard-v2/issues/1578) ([maciaszczykm](https://github.com/maciaszczykm))
- Pod Security Policy can now be activated from the wizard. [#1647](https://github.com/kubermatic/dashboard-v2/issues/1647) ([maciaszczykm](https://github.com/maciaszczykm))
- Redesigned extended options in wizard [#1609](https://github.com/kubermatic/dashboard-v2/issues/1609) ([kgroschoff](https://github.com/kgroschoff))
- Various security improvements in authentication
- Various other visual improvements




### [v2.11.8]()


- End-of-Life Kubernetes v1.14 is no longer supported. [#4989](https://github.com/kubermatic/kubermatic/issues/4989) ([kdomanski](https://github.com/kdomanski))
- Added Kubernetes v1.15.7, v1.15.9 [#4995](https://github.com/kubermatic/kubermatic/issues/4995) ([kdomanski](https://github.com/kdomanski))




### [v2.11.7]()


- Kubernetes 1.13 which is end-of-life has been removed. [#4327](https://github.com/kubermatic/kubermatic/issues/4327) ([kdomanski](https://github.com/kdomanski))
- Added Kubernetes v1.15.4 [#4329](https://github.com/kubermatic/kubermatic/issues/4329) ([kdomanski](https://github.com/kdomanski))
- Added Kubernetes v1.14.7 [#4330](https://github.com/kubermatic/kubermatic/issues/4330) ([kdomanski](https://github.com/kdomanski))
- A bug that prevented node Labels, Taints and Annotations from getting applied correctly was fixed. [#4368](https://github.com/kubermatic/kubermatic/issues/4368) ([alvaroaleman](https://github.com/alvaroaleman))
- Removed K8S releases affected by CVE-2019-11253 [#4515](https://github.com/kubermatic/kubermatic/issues/4515) ([kdomanski](https://github.com/kdomanski))
- A fix for CVE-2019-11253 for clusters that were created with a Kubernetes version &lt; 1.14 was deployed [#4520](https://github.com/kubermatic/kubermatic/issues/4520) ([alvaroaleman](https://github.com/alvaroaleman))
- Openstack: fixed API usage for datacenters with only one region [#4536](https://github.com/kubermatic/kubermatic/issues/4536) ([zreigz](https://github.com/zreigz))




### [v2.11.6]()


- Fixed a bug that could cause intermittent delays when using kubectl logs/exec with `exposeStrategy: LoadBalancer` [#4279](https://github.com/kubermatic/kubermatic/issues/4279) ([kubermatic-bot](https://github.com/kubermatic-bot))




### [v2.11.5]()


- Fix a bug that caused setup on nodes with a Kernel &gt; 4.18 to fail [#4180](https://github.com/kubermatic/kubermatic/issues/4180) ([alvaroaleman](https://github.com/alvaroaleman))
- Fixed fetching the list of tenants on some OpenStack configurations with one region [#4185](https://github.com/kubermatic/kubermatic/issues/4185) ([kubermatic-bot](https://github.com/kubermatic-bot))
- Fixed a bug that could result in the clusterdeletion sometimes getting stuck [#4202](https://github.com/kubermatic/kubermatic/issues/4202) ([alvaroaleman](https://github.com/alvaroaleman))




### [v2.11.4]()


- `kubeadm join` has been fixed for v1.15 clusters [#4162](https://github.com/kubermatic/kubermatic/issues/4162) ([kubermatic-bot](https://github.com/kubermatic-bot))




### [v2.11.3]()


- Kubermatic Swagger API Spec is now exposed over its API server [#3890](https://github.com/kubermatic/kubermatic/issues/3890) ([bashofmann](https://github.com/bashofmann))
- updated Envoy to 1.11.1 [#4075](https://github.com/kubermatic/kubermatic/issues/4075) ([xrstf](https://github.com/xrstf))
- Kubernetes versions affected by CVE-2019-9512 and CVE-2019-9514 have been dropped [#4118](https://github.com/kubermatic/kubermatic/issues/4118) ([kubermatic-bot](https://github.com/kubermatic-bot))
- Enabling the OIDC feature flag in clusters has been fixed. [#4136](https://github.com/kubermatic/kubermatic/issues/4136) ([kubermatic-bot](https://github.com/kubermatic-bot))




### [v2.10.3]()


- Kubernetes 1.11 which is end-of-life has been removed. [#4031](https://github.com/kubermatic/kubermatic/issues/4031) ([kubermatic-bot](https://github.com/kubermatic-bot))
- Kubernetes 1.12 which is end-of-life has been removed. [#4065](https://github.com/kubermatic/kubermatic/issues/4065) ([kdomanski](https://github.com/kdomanski))
- Kubernetes versions affected by CVE-2019-11247 and CVE-2019-11249 have been dropped [#4066](https://github.com/kubermatic/kubermatic/issues/4066) ([kdomanski](https://github.com/kdomanski))
- Kubernetes versions affected by CVE-2019-9512 and CVE-2019-9514 have been dropped [#4113](https://github.com/kubermatic/kubermatic/issues/4113) ([kdomanski](https://github.com/kdomanski))
- updated Envoy to 1.11.1 [#4075](https://github.com/kubermatic/kubermatic/issues/4075) ([xrstf](https://github.com/xrstf))




### [v2.11.2]()


- Fixed an issue where deleted project owners would come back after a while [#4020](https://github.com/kubermatic/kubermatic/issues/4020) ([mrIncompetent](https://github.com/mrIncompetent))
- Kubernetes versions affected by CVE-2019-11247 and CVE-2019-11249 have been dropped [#4066](https://github.com/kubermatic/kubermatic/issues/4066) ([kdomanski](https://github.com/kdomanski))
- Kubernetes 1.11 which is end-of-life has been removed. [#4030](https://github.com/kubermatic/kubermatic/issues/4030) ([kubermatic-bot](https://github.com/kubermatic-bot))
- Kubernetes 1.12 which is end-of-life has been removed. [#4067](https://github.com/kubermatic/kubermatic/issues/4067) ([kubermatic-bot](https://github.com/kubermatic-bot))




### [v2.11.1]()


**Misc:**

- Openstack: A bug that could result in many securtiy groups being created when the creation of security group rules failed was fixed [#3848](https://github.com/kubermatic/kubermatic/issues/3848) ([alvaroaleman](https://github.com/alvaroaleman))
- Added Kubernetes `v1.15.1` [#3859](https://github.com/kubermatic/kubermatic/issues/3859) ([kubermatic-bot](https://github.com/kubermatic-bot))
- Updated machine controller to `v1.5.1` [#3883](https://github.com/kubermatic/kubermatic/issues/3883) ([kdomanski](https://github.com/kdomanski))
- A bug that sometimes resulted in the creation of the initial NodeDeployment failing was fixed [#3894](https://github.com/kubermatic/kubermatic/issues/3894) ([alvaroaleman](https://github.com/alvaroaleman))
- Fixed an issue that kept clusters stuck if their creation didn't succeed and they got deleted with LB and/or PV cleanup enabled [#3973](https://github.com/kubermatic/kubermatic/issues/3973) ([alvaroaleman](https://github.com/alvaroaleman))
- Fixed joining nodes to Bring Your Own clusters running Kubernetes 1.14 [#3976](https://github.com/kubermatic/kubermatic/issues/3976) ([kubermatic-bot](https://github.com/kubermatic-bot))


**Dashboard:**

- Fixed an issue with handling resources refresh on error conditions [#1452](https://github.com/kubermatic/dashboard-v2/issues/1452) ([floreks](https://github.com/floreks))
- Openstack: the project name can now be provided manually [#1426](https://github.com/kubermatic/dashboard-v2/issues/1426) ([floreks](https://github.com/floreks))
- JS dependencies have been updated to address potential vulnerabilities in some of them. [#1388](https://github.com/kubermatic/dashboard-v2/issues/1388) ([kgroschoff](https://github.com/kgroschoff)




### [v2.11.0]()


Supported Kubernetes versions:
- `1.11.5-10`
- `1.12.3-10`
- `1.13.0-5`
- `1.13.7`
- `1.14.0-1`
- `1.14.3-4`
- `1.15.0`


**Cloud providers:**
- It is now possible to create Kubermatic-managed clusters on Packet. [#3419](https://github.com/kubermatic/kubermatic/issues/3419) ([nikhita](https://github.com/nikhita))
- It is now possible to create Kubermatic-managed clusters on GCP. [#3350](https://github.com/kubermatic/kubermatic/issues/3350) ([nikhita](https://github.com/nikhita))
- the API stops creating an initial node deployment for new cluster for KubeAdm providers. [#3346](https://github.com/kubermatic/kubermatic/issues/3346) ([p0lyn0mial](https://github.com/p0lyn0mial))
- Openstack: datacenter can be configured with minimum required CPU and memory for nodes [#3487](https://github.com/kubermatic/kubermatic/issues/3487) ([bashofmann](https://github.com/bashofmann))
- vsphere: root disk size is now configurable [#3629](https://github.com/kubermatic/kubermatic/issues/3629) ([kgroschoff](https://github.com/kgroschoff))
- Azure: fixed failure to provision on new regions due to lower number of fault domains [#3584](https://github.com/kubermatic/kubermatic/issues/3584) ([kdomanski](https://github.com/kdomanski))


**Monitoring:**
- [ACTION REQUIRED] refactored Alertmanager Helm chart for master-cluster monitoring, see documentation for migration notes  [#3448](https://github.com/kubermatic/kubermatic/issues/3448) ([xrstf](https://github.com/xrstf))
- cAdvisor metrics are now being scraped for user clusters [#3390](https://github.com/kubermatic/kubermatic/issues/3390) ([mrIncompetent](https://github.com/mrIncompetent))
- fixed kube-state-metrics in user-clusters not being scraped [#3427](https://github.com/kubermatic/kubermatic/issues/3427) ([xrstf](https://github.com/xrstf))
- Improved debugging of resource leftovers through new etcd Object Count dashboard [#3508](https://github.com/kubermatic/kubermatic/issues/3508) ([xrstf](https://github.com/xrstf))
- New Grafana dashboards for monitoring Elasticsearch [#3516](https://github.com/kubermatic/kubermatic/issues/3516) ([xrstf](https://github.com/xrstf))
- Added optional Thanos integration to Prometheus for better long-term metrics storage [#3531](https://github.com/kubermatic/kubermatic/issues/3531) ([xrstf](https://github.com/xrstf))


**Misc:**
- [ACTION REQUIRED] nodePortPoxy Helm values has been renamed to nodePortProxy, old root key is now deprecated; please update your Helm values [#3418](https://github.com/kubermatic/kubermatic/issues/3418) ([xrstf](https://github.com/xrstf))
- Service accounts have been implemented.
- Support for Kubernetes 1.15 was added [#3579](https://github.com/kubermatic/kubermatic/issues/3579) ([alvaroaleman](https://github.com/alvaroaleman))
- More details are shown when using `kubectl get machine/machineset/machinedeployment` [#3364](https://github.com/kubermatic/kubermatic/issues/3364) ([alvaroaleman](https://github.com/alvaroaleman))
- The resiliency of in-cluster DNS was greatly improved by adding the nodelocal-dns-cache addon, which runs a DNS cache on each node, avoiding the need to use NAT for DNS queries [#3369](https://github.com/kubermatic/kubermatic/issues/3369) ([alvaroaleman](https://github.com/alvaroaleman))
- Added containerRuntimeVersion and kernelVersion to NodeInfo [#3381](https://github.com/kubermatic/kubermatic/issues/3381) ([bashofmann](https://github.com/bashofmann))
- It is now possible to configure Kubermatic to create one service of type LoadBalancer per user cluster instead of exposing all of them via the nodeport-proxy on one central LoadBalancer service [#3387](https://github.com/kubermatic/kubermatic/issues/3387) ([alvaroaleman](https://github.com/alvaroaleman))
- Pod AntiAffinity and PDBs were added to the Kubermatic control plane components,the monitoring stack and the logging stack to spread them out if possible and reduce the chance of unavailability [#3393](https://github.com/kubermatic/kubermatic/issues/3393) ([alvaroaleman](https://github.com/alvaroaleman))
- Reduced API latency for loading Nodes & NodeDeployments [#3405](https://github.com/kubermatic/kubermatic/issues/3405) ([mrIncompetent](https://github.com/mrIncompetent))
- replace gambol99/keycloak-proxy 2.3.0 with official keycloak-gatekeeper 6.0.1 [#3411](https://github.com/kubermatic/kubermatic/issues/3411) ([xrstf](https://github.com/xrstf))
- More additional printer columns for kubermatic crds [#3542](https://github.com/kubermatic/kubermatic/issues/3542) ([bashofmann](https://github.com/bashofmann))
- Insecure Kubernetes versions v1.13.6 and v1.14.2 have been disabled. [#3554](https://github.com/kubermatic/kubermatic/issues/3554) ([mrIncompetent](https://github.com/mrIncompetent))
- Kubermatic now supports running in environments where the Internet can only be accessed via a http proxy [#3615](https://github.com/kubermatic/kubermatic/issues/3615) ([mrIncompetent](https://github.com/mrIncompetent))
- ICMP traffic to clusters is now always permitted to allow MTU discovery [#3618](https://github.com/kubermatic/kubermatic/issues/3618) ([kdomanski](https://github.com/kdomanski))
- A bug that caused errors on very big addon manifests was fixed [#3366](https://github.com/kubermatic/kubermatic/issues/3366) ([alvaroaleman](https://github.com/alvaroaleman))
- Updated Prometheus to 2.10.0 [#3612](https://github.com/kubermatic/kubermatic/issues/3612) ([xrstf](https://github.com/xrstf))
- Updated cert-manager to 0.8.0 [#3525](https://github.com/kubermatic/kubermatic/issues/3525) ([xrstf](https://github.com/xrstf))
- Updated Minio to RELEASE.2019-06-11T00-44-33Z [#3614](https://github.com/kubermatic/kubermatic/issues/3614) ([xrstf](https://github.com/xrstf))
- Updated Grafana to 6.2.1 [#3528](https://github.com/kubermatic/kubermatic/issues/3528) ([xrstf](https://github.com/xrstf))
- Updated kube-state-metrics to 1.6.0 [#3420](https://github.com/kubermatic/kubermatic/issues/3420) ([xrstf](https://github.com/xrstf))
- Updated Dex to 2.16.0 [#3361](https://github.com/kubermatic/kubermatic/issues/3361) ([xrstf](https://github.com/xrstf))
- Updated Alertmanager to 0.17.0, deprecate version field in favor of image.tag in Helm values.yaml [#3410](https://github.com/kubermatic/kubermatic/issues/3410) ([xrstf](https://github.com/xrstf))
- Updated `machine-controller` to `v1.4.2`. [#3778](https://github.com/kubermatic/kubermatic/issues/3778) ([alvaroaleman](https://github.com/alvaroaleman))
- Updated node-exporter to 0.18.1 [#3613](https://github.com/kubermatic/kubermatic/issues/3613) ([xrstf](https://github.com/xrstf))
- Updated fluent-bit to 1.1.2 [#3561](https://github.com/kubermatic/kubermatic/issues/3561) ([xrstf](https://github.com/xrstf))
- Updated Velero to 1.0 [#3527](https://github.com/kubermatic/kubermatic/issues/3527) ([xrstf](https://github.com/xrstf))


**Dashboard:**
- The project menu has been redesigned. [#1195](https://github.com/kubermatic/dashboard-v2/issues/1195) ([maciaszczykm](https://github.com/maciaszczykm))
- Fixed changing default OpenStack image on operating system change [#1215](https://github.com/kubermatic/dashboard-v2/issues/1215) ([bashofmann](https://github.com/bashofmann))
- `containerRuntimeVersion` and `kernelVersion` are now displayed on NodeDeployment detail page [#1216](https://github.com/kubermatic/dashboard-v2/issues/1216) ([bashofmann](https://github.com/bashofmann))
- Custom links can now be added to the footer. [#1220](https://github.com/kubermatic/dashboard-v2/issues/1220) ([maciaszczykm](https://github.com/maciaszczykm))
- The OIDC provider URL is now configurable via &#34;oidc_provider_url&#34; variable. [#1222](https://github.com/kubermatic/dashboard-v2/issues/1222) ([maciaszczykm](https://github.com/maciaszczykm))
- The application logo has been changed. [#1232](https://github.com/kubermatic/dashboard-v2/issues/1232) ([maciaszczykm](https://github.com/maciaszczykm))
- The breadcrumbs component has been removed. The dialogs and buttons have been redesigned. [#1233](https://github.com/kubermatic/dashboard-v2/issues/1233) ([maciaszczykm](https://github.com/maciaszczykm))
- Packet cloud provider is now supported. [#1238](https://github.com/kubermatic/dashboard-v2/issues/1238) ([maciaszczykm](https://github.com/maciaszczykm))
- Tables have been redesigned. [#1240](https://github.com/kubermatic/dashboard-v2/issues/1240) ([kgroschoff](https://github.com/kgroschoff))
- Added option to specify taints when creating/updating NodeDeployments [#1244](https://github.com/kubermatic/dashboard-v2/issues/1244) ([bashofmann](https://github.com/bashofmann))
- Styling of the cluster details view has been improved. [#1270](https://github.com/kubermatic/dashboard-v2/issues/1270) ([maciaszczykm](https://github.com/maciaszczykm))
- Missing parameters for OIDC providers have been added. [#1273](https://github.com/kubermatic/dashboard-v2/issues/1273) ([maciaszczykm](https://github.com/maciaszczykm))
- Dates are now displayed using relative format, i.e. 3 days ago. [#1303](https://github.com/kubermatic/dashboard-v2/issues/1303) ([maciaszczykm](https://github.com/maciaszczykm))
- Redesigned dialogs and cluster details page. [#1305](https://github.com/kubermatic/dashboard-v2/issues/1305) ([maciaszczykm](https://github.com/maciaszczykm))
- Add provider GCP to UI [#1307](https://github.com/kubermatic/dashboard-v2/issues/1307) ([kgroschoff](https://github.com/kgroschoff))
- Redesigned notifications. [#1315](https://github.com/kubermatic/dashboard-v2/issues/1315) ([maciaszczykm](https://github.com/maciaszczykm))
- The Instance Profile Name for AWS could be specified in UI. [#1317](https://github.com/kubermatic/dashboard-v2/issues/1317) ([kgroschoff](https://github.com/kgroschoff))
- Redesigned node deployment view. [#1320](https://github.com/kubermatic/dashboard-v2/issues/1320) ([maciaszczykm](https://github.com/maciaszczykm))
- Redesigned cluster details page. [#1345](https://github.com/kubermatic/dashboard-v2/issues/1345) ([kubermatic-bot](https://github.com/kubermatic-bot))



### [v2.10.2]()


**Misc:**

- Updated Dashboard to `v1.2.2` [#3553](https://github.com/kubermatic/kubermatic/issues/3553) ([kubermatic-bot](https://github.com/kubermatic-bot))
    - Missing parameters for OIDC providers have been added. [#1273](https://github.com/kubermatic/dashboard-v2/issues/1273) ([maciaszczykm](https://github.com/maciaszczykm))
    - `containerRuntimeVersion` and `kernelVersion` are now displayed on NodeDeployment detail page [#1217](https://github.com/kubermatic/dashboard-v2/issues/1217) ([kubermatic-bot](https://github.com/kubermatic-bot))
    - Fixed changing default OpenStack image on Operating System change [#1218](https://github.com/kubermatic/dashboard-v2/issues/1218) ([kubermatic-bot](https://github.com/kubermatic-bot))
    - The OIDC provider URL is now configurable via &#34;oidc_provider_url&#34; variable. [#1224](https://github.com/kubermatic/dashboard-v2/issues/1224) ([kubermatic-bot](https://github.com/kubermatic-bot))
- Insecure Kubernetes versions v1.13.6 and v1.14.2 have been disabled. [#3554](https://github.com/kubermatic/kubermatic/issues/3554) ([mrIncompetent](https://github.com/mrIncompetent))




### [v2.10.1]()

**Bugfix:**

- A bug that caused errors on very big addon manifests was fixed [#3366](https://github.com/kubermatic/kubermatic/issues/3366) ([alvaroaleman](https://github.com/alvaroaleman))
- fixed kube-state-metrics in user-clusters not being scraped [#3431](https://github.com/kubermatic/kubermatic/issues/3431) ([kubermatic-bot](https://github.com/kubermatic-bot))
- Updated the machine-controller to fix the wrong CentOS image for AWS instances [#3432](https://github.com/kubermatic/kubermatic/issues/3432) ([mrIncompetent](https://github.com/mrIncompetent))
- vSphere VMs are cleaned up on ISO failure.  [#3474](https://github.com/kubermatic/kubermatic/issues/3474) ([nikhita](https://github.com/nikhita))


**Misc:**

- updated Prometheus to `v2.9.2` [#3348](https://github.com/kubermatic/kubermatic/issues/3348) ([kubermatic-bot](https://github.com/kubermatic-bot))
- Draining of nodes now times out after 2h [#3354](https://github.com/kubermatic/kubermatic/issues/3354) ([kubermatic-bot](https://github.com/kubermatic-bot))
- the API stops creating an initial node deployment for new cluster for KubeAdm providers. [#3373](https://github.com/kubermatic/kubermatic/issues/3373) ([kubermatic-bot](https://github.com/kubermatic-bot))
- More details are shown when using `kubectl get machine/machineset/machinedeployment` [#3377](https://github.com/kubermatic/kubermatic/issues/3377) ([kubermatic-bot](https://github.com/kubermatic-bot))
- Pod AntiAffinity and PDBs were added to the Kubermatic control plane components and the monitoring stack to spread them out if possible and reduce the chance of unavailability [#3400](https://github.com/kubermatic/kubermatic/issues/3400) ([kubermatic-bot](https://github.com/kubermatic-bot))
- Support for Kubernetes 1.11.10 was added [#3429](https://github.com/kubermatic/kubermatic/issues/3429) ([kubermatic-bot](https://github.com/kubermatic-bot))




### [v2.10.0]()

## Features

### Kubermatic core

* ACTION REQUIRED: The config option `Values.kubermatic.rbac` changed to `Values.kubermatic.masterController` [#3051](https://github.com/kubermatic/kubermatic/pull/3051) ([@zreigz](https://github.com/zreigz))
* The user cluster controller manager was added. It is deployed within the cluster namespace in the seed and takes care of reconciling all resources that are inside the user cluster
* Add feature gate to enable etcd corruption check [#2460](https://github.com/kubermatic/kubermatic/pull/2460) ([@mrIncompetent](https://github.com/mrIncompetent))
* Kubernetes 1.10 was removed as officially supported version from Kubermatic as it's EOL [#2712](https://github.com/kubermatic/kubermatic/pull/2712) ([@alvaroaleman](https://github.com/alvaroaleman))
* Add short names to the ClusterAPI CRDs to allow using `kubectl get md` for `machinedeployments`, `kubectl get ms` for `machinesets` and `kubectl get ma` to get `machines`
   [#2718](https://github.com/kubermatic/kubermatic/pull/2718) ([@toschneck](https://github.com/toschneck))
* Update canal to v2.6.12, Kubernetes Dashboard to v1.10.1 and replace kube-dns with CoreDNS 1.3.1 [#2985](https://github.com/kubermatic/kubermatic/pull/2985) ([@mrIncompetent](https://github.com/mrIncompetent))
* Update Vertical Pod Autoscaler to 0.5 [#3143](https://github.com/kubermatic/kubermatic/pull/3143) ([@xrstf](https://github.com/xrstf))
* Avoid the name "kubermatic" for cloud provider resources visible by end users [#3152](https://github.com/kubermatic/kubermatic/pull/3152) ([@mrIncompetent](https://github.com/mrIncompetent))
* In order to provide Grafana dashboards for user cluster resource usage, the node-exporter is now deployed by default as an addon into user clusters. [#3089](https://github.com/kubermatic/kubermatic/pull/3089) ([@xrstf](https://github.com/xrstf))
* Make the default AMI's for AWS instances configurable via the datacenters.yaml [#3169](https://github.com/kubermatic/kubermatic/pull/3169) ([@mrIncompetent](https://github.com/mrIncompetent))
* Vertical Pod Autoscaler is not deployed by default anymore [#2805](https://github.com/kubermatic/kubermatic/pull/2805) ([@xrstf](https://github.com/xrstf))
* Initial node deployments are now created inside the same API call as the cluster, fixing spurious issues where the creation didn't happen [#2989](https://github.com/kubermatic/kubermatic/pull/2989) ([@maciaszczykm](https://github.com/maciaszczykm))
* Errors when reconciling MachineDeployments and MachineSets will now result in an event on the object [#2923](https://github.com/kubermatic/kubermatic/pull/2923) ([@alvaroaleman](https://github.com/alvaroaleman))
* Filter out not valid VM types for azure provider [#2736](https://github.com/kubermatic/kubermatic/pull/2736) ([@zreigz](https://github.com/zreigz))
* Mark cluster upgrades as restricted if kubelet version is incompatible. [#2976](https://github.com/kubermatic/kubermatic/pull/2976) ([@maciaszczykm](https://github.com/maciaszczykm))
* Enable automatic detection of the OpenStack BlockStorage API version within the cloud config [#3112](https://github.com/kubermatic/kubermatic/pull/3112) ([@mrIncompetent](https://github.com/mrIncompetent))
* Add the ContainerLinuxUpdateOperator to all clusters that use ContainerLinux nodes [#3239](https://github.com/kubermatic/kubermatic/pull/3239) ([@mrIncompetent](https://github.com/mrIncompetent))
* The trust-device-path cloud config property of Openstack clusters can be configured via datacenters.yaml. [#3265](https://github.com/kubermatic/kubermatic/pull/3265) ([@nikhita](https://github.com/nikhita))
* Set AntiAffinity for pods to prevent situations where the API servers of all clusters got scheduled on a single node [#3269](https://github.com/kubermatic/kubermatic/pull/3269) ([@mrIncompetent](https://github.com/mrIncompetent))
* Set resource requests & limits for all addons [#3270](https://github.com/kubermatic/kubermatic/pull/3270) ([@mrIncompetent](https://github.com/mrIncompetent))
* Add Kubernetes v1.14.1 to the list of supported versions [#3273](https://github.com/kubermatic/kubermatic/pull/3273) ([@mrIncompetent](https://github.com/mrIncompetent))
* A small amount of resources gets reserved on each node for the Kubelet and system services [#3298](https://github.com/kubermatic/kubermatic/pull/3298) ([@alvaroaleman](https://github.com/alvaroaleman))
* Update etcd to v3.3.12 [#3288](https://github.com/kubermatic/kubermatic/pull/3288) ([@mrIncompetent](https://github.com/mrIncompetent))
* Update the metrics-server to v0.3.2 [#3289](https://github.com/kubermatic/kubermatic/pull/3289) ([@mrIncompetent](https://github.com/mrIncompetent))
* Update the user cluster Prometheus to v2.9.1 [#3287](https://github.com/kubermatic/kubermatic/pull/3287) ([@mrIncompetent](https://github.com/mrIncompetent))
* It is now possible to scale MachineDeployments and MachineSets via `kubectl scale` [#3277](https://github.com/kubermatic/kubermatic/pull/3277) ([@alvaroaleman](https://github.com/alvaroaleman))

## Dashboard

* The color scheme of the Dashboard was changed
* It is now possible to edit the project name in UI [#1003](https://github.com/kubermatic/dashboard-v2/pull/1003) ([@kgroschoff](https://github.com/kgroschoff))
* Made Nodes and Node Deployments statuses more accurate [#1016](https://github.com/kubermatic/dashboard-v2/pull/1016) ([@maciaszczykm](https://github.com/maciaszczykm))
* Redesign DigitalOcean sizes and OpenStack flavors option pickers [#1021](https://github.com/kubermatic/dashboard-v2/pull/1021) ([@maciaszczykm](https://github.com/maciaszczykm))
* Smoother operation on bad network connection thanks to changes in asset caching. [#1030](https://github.com/kubermatic/dashboard-v2/pull/1030) ([@kdomanski](https://github.com/kdomanski))
* Added a flag allowing to change the default number of nodes created with clusters. [#1032](https://github.com/kubermatic/dashboard-v2/pull/1032) ([@maciaszczykm](https://github.com/maciaszczykm))
* Setting openstack tags for instances is possible via UI now. [#1038](https://github.com/kubermatic/dashboard-v2/pull/1038) ([@kgroschoff](https://github.com/kgroschoff))
* Allowed Node Deployment naming. [#1039](https://github.com/kubermatic/dashboard-v2/pull/1039) ([@maciaszczykm](https://github.com/maciaszczykm))
* Adding multiple owners to a project is possible via UI now. [#1042](https://github.com/kubermatic/dashboard-v2/pull/1042) ([@kgroschoff](https://github.com/kgroschoff))
* Allowed specifying kubelet version for Node Deployments. [#1047](https://github.com/kubermatic/dashboard-v2/pull/1047) ([@maciaszczykm](https://github.com/maciaszczykm))
* Events related to the Nodes are now displayed in the Node Deployment details view. [#1054](https://github.com/kubermatic/dashboard-v2/pull/1054) ([@maciaszczykm](https://github.com/maciaszczykm))
* Fixed reload behaviour of openstack setting fields. [#1056](https://github.com/kubermatic/dashboard-v2/pull/1056) ([@kgroschoff](https://github.com/kgroschoff))
* Fixed a bug with the missing version in the footer. [#1067](https://github.com/kubermatic/dashboard-v2/pull/1067) ([@maciaszczykm](https://github.com/maciaszczykm))
* Project owners are now visible in project list view . [#1082](https://github.com/kubermatic/dashboard-v2/pull/1082) ([@kgroschoff](https://github.com/kgroschoff))
* Added possibility to assign labels to nodes. [#1101](https://github.com/kubermatic/dashboard-v2/pull/1101) ([@maciaszczykm](https://github.com/maciaszczykm))
* Updated AWS instance types. [#1122](https://github.com/kubermatic/dashboard-v2/pull/1122) ([@maciaszczykm](https://github.com/maciaszczykm))
* Fixed display number of replicas if the field is empty (0 replicas). [#1126](https://github.com/kubermatic/dashboard-v2/pull/1126) ([@maciaszczykm](https://github.com/maciaszczykm))
* Added an option to include custom links into the application. [#1131](https://github.com/kubermatic/dashboard-v2/pull/1131) ([@maciaszczykm](https://github.com/maciaszczykm))
* Remove AWS instance types t3.nano & t3.micro as they are too small to schedule any workload on them [#1138](https://github.com/kubermatic/dashboard-v2/pull/1138) ([@mrIncompetent](https://github.com/mrIncompetent))
* Redesigned the application sidebar. [#1173](https://github.com/kubermatic/dashboard-v2/pull/1173) ([@maciaszczykm](https://github.com/maciaszczykm))

### Logging & Monitoring stack

* Update fluent-bit to 1.0.6 [#3222](https://github.com/kubermatic/kubermatic/pull/3222) ([@xrstf](https://github.com/xrstf))
* Add elasticsearch-exporter to logging stack to improve monitoring [#2773](https://github.com/kubermatic/kubermatic/pull/2773) ([@xrstf](https://github.com/xrstf))
* New alerts for cert-manager created certificates about to expire [#2787](https://github.com/kubermatic/kubermatic/pull/2787) ([@xrstf](https://github.com/xrstf))
* Add blackbox-exporter chart [#2954](https://github.com/kubermatic/kubermatic/pull/2954) ([@xrstf](https://github.com/xrstf))
* Update Elasticsearch to 6.6.2 [#3062](https://github.com/kubermatic/kubermatic/pull/3062) ([@xrstf](https://github.com/xrstf))
* Add Grafana dashboards for kubelet metrics [#3081](https://github.com/kubermatic/kubermatic/pull/3081) ([@xrstf](https://github.com/xrstf))
* Prometheus was updated to 2.8.1 (Alertmanager 0.16.2), Grafana was updated to 6.1.3 [#3163](https://github.com/kubermatic/kubermatic/pull/3163) ([@xrstf](https://github.com/xrstf))
* Alertmanager PVC size is configurable [#3199](https://github.com/kubermatic/kubermatic/pull/3199) ([@kron4eg](https://github.com/kron4eg))
* Add lifecycle hooks to the Elasticsearch StatefulSet to make starting/stopping more graceful [#2933](https://github.com/kubermatic/kubermatic/pull/2933) ([@mrIncompetent](https://github.com/mrIncompetent))
* Pod annotations are no longer logged in Elasticsearch [#2959](https://github.com/kubermatic/kubermatic/pull/2959) ([@xrstf](https://github.com/xrstf))
* Improve Prometheus backups in high traffic environments [#3047](https://github.com/kubermatic/kubermatic/pull/3047) ([@xrstf](https://github.com/xrstf))
* Fix VolumeSnapshotLocations for Ark configuration [#3076](https://github.com/kubermatic/kubermatic/pull/3076) ([@xrstf](https://github.com/xrstf))
* node-exporter is not exposed on all host interfaces anymore [#3085](https://github.com/kubermatic/kubermatic/pull/3085) ([@xrstf](https://github.com/xrstf))
* Improve Kibana usability by auto-provisioning index patterns [#3099](https://github.com/kubermatic/kubermatic/pull/3099) ([@xrstf](https://github.com/xrstf))
* Configurable Prometheus backup timeout to accomodate larger seed clusters [#3223](https://github.com/kubermatic/kubermatic/pull/3223) ([@xrstf](https://github.com/xrstf))

### Other

* ACTION REQUIRED: update from Ark 0.10 to Velero 0.11 [#3077](https://github.com/kubermatic/kubermatic/pull/3077) ([@xrstf](https://github.com/xrstf))
* Replace hand written go tcp proxy with Envoy within the nodeport-proxy [#2916](https://github.com/kubermatic/kubermatic/pull/2916) ([@mrIncompetent](https://github.com/mrIncompetent))
* cert-manager was updated to 0.7.0, Dex was updated to 2.15.0,Minio was updated to RELEASE.2019-04-09T01-22-30Z [#3163](https://github.com/kubermatic/kubermatic/pull/3163) ([@xrstf](https://github.com/xrstf))
* update nginx-ingress-controller to 0.24.1 [#3200](https://github.com/kubermatic/kubermatic/pull/3200) ([@xrstf](https://github.com/xrstf))
* Allow scheduling Helm charts using affinities, node selectors and tolerations for more stable clusters [#3155](https://github.com/kubermatic/kubermatic/pull/3155) ([@xrstf](https://github.com/xrstf))
* Helm charts: Define configurable resource constraints [#3012](https://github.com/kubermatic/kubermatic/pull/3012) ([@xrstf](https://github.com/xrstf))
* improve Helm charts metadata to make Helm-based workflows easier and aid in cluster updates [#3221](https://github.com/kubermatic/kubermatic/pull/3221) ([@xrstf](https://github.com/xrstf))
* dex keys expirations can now be configured in helm chart [#3301](https://github.com/kubermatic/kubermatic/pull/3301) ([@kron4eg](https://github.com/kron4eg))
* Update the nodeport-proxy Envoy to v1.10 [#3274](https://github.com/kubermatic/kubermatic/pull/3274) ([@mrIncompetent](https://github.com/mrIncompetent))

## Bugfixes

* Fixed invalid variable caching in Grafana dashboards [#2792](https://github.com/kubermatic/kubermatic/pull/2792) ([@xrstf](https://github.com/xrstf))
* Migrations are now executed only after the leader lease was acquired [#3276](https://github.com/kubermatic/kubermatic/pull/3276) ([@alvaroaleman](https://github.com/alvaroaleman))


### [v2.9.3]()


- Errors when reconciling MachineDeployments and MachineSets will now result in an event on the object [#2930](https://github.com/kubermatic/kubermatic/issues/2930) ([kubermatic-bot](https://github.com/kubermatic-bot))
- Missing permissions have been added to the kube-state-metrics ClusterRole [#2978](https://github.com/kubermatic/kubermatic/issues/2978) ([kubermatic-bot](https://github.com/kubermatic-bot))
- Fixed invalid variable caching in Grafana dashboards [#2992](https://github.com/kubermatic/kubermatic/issues/2992) ([kubermatic-bot](https://github.com/kubermatic-bot))
- Kibana is automatically initialized in new installations. [#2995](https://github.com/kubermatic/kubermatic/issues/2995) ([kubermatic-bot](https://github.com/kubermatic-bot))
- Updated machine controller to `v1.1.0` [#3028](https://github.com/kubermatic/kubermatic/issues/3028) ([mrIncompetent](https://github.com/mrIncompetent))




### [v2.9.2]()

* The cleanup of services of type LoadBalancer on cluster deletion was fixed and re-enabled [#2780](https://github.com/kubermatic/kubermatic/pull/2780)
* The Kubernetes Dashboard addon was updated to 1.10.1 [#2848](https://github.com/kubermatic/kubermatic/pull/2848)
* Joining of nodes via the BYO functionality was fixed [#2835](https://github.com/kubermatic/kubermatic/pull/2835)
* [It is now possible to configure whether Openstack security groups for LoadBalancers should be managed by Kubernetes, check the sample `datacenters.yaml` in the docs for details](https://docs.kubermatic.io/installation/install_kubermatic/_manual/#defining-the-datacenters) [#2878](https://github.com/kubermatic/kubermatic/pull/2878)
* A bug that resulted in clusters being twice in the UI overview got resolved [#1088](https://github.com/kubermatic/dashboard-v2/pull/1088)
* A bug that could cause the image of a NodeDeployment to be set to the default when the NodeDeployment gets edited got resolved [#1076](https://github.com/kubermatic/dashboard-v2/pull/1076)
* A bug that caused the version of the UI to not be shown in the footer got resolved [#1096](https://github.com/kubermatic/dashboard-v2/pull/1096)
* A bug that caused updating and deleting of NodeDeployments in the NodeDeployment details page not to work got resolved [#1076](https://github.com/kubermatic/dashboard-v2/pull/1076)
* The NodeDeployment detail view now correctly displays the node datacenter instead of the seed datacenter [#1094](https://github.com/kubermatic/dashboard-v2/pull/1094)
* Support for Kubernetes 1.11.8, 1.12.6, 1.13.3 and 1.13.4 was added [#2894](https://github.com/kubermatic/kubermatic/pull/2894)




### [v2.9.1]()

* The Docker version used for all new machines with CoreOS or Ubuntu has a fix for CVE-2019-573. It s advised to roll over all your worker nodes to make sure that new version is used
* It is now possible to name NodeDeployments
* A bug that caused duplicate top level keys in the values.example.yaml got fixed
* A bug that made it impossible to choose a subnet on Openstack after a network was choosen got fixed
* Scraping of 1.13 user cluster Schedulers and Controller manager now works
* Scraping of the seed clusters Scheduler and Controller manager now works
* A bug that caused spurious failures when appplying the cert-manager chart was resolved
* NodeDeployment events are now shown in the UI
* It is now possible to configure the Kubernetes version of a NodeDeployment in the UI




### [v2.9.0]()


Supported Kubernetes versions:
- `1.11.5-7`
- `1.12.3-5`
- `1.13.0-2`


**Cloud Provider:**

- Added support for PersistentVolumes on **Hetzner Cloud** [#2613](https://github.com/kubermatic/kubermatic/issues/2613) ([alvaroaleman](https://github.com/alvaroaleman))
- Openstack Floating IPs will now be de-allocated from your project if they were allocated during node creation [#2675](https://github.com/kubermatic/kubermatic/issues/2675) ([alvaroaleman](https://github.com/alvaroaleman))


**Misc:**

- Added support for Kubernetes `v1.13`
- Kubermatic now supports Kubernetes 1.12 [#2132](https://github.com/kubermatic/kubermatic/issues/2132) ([alvaroaleman](https://github.com/alvaroaleman))
- The startup time for new clusters was improved [#2148](https://github.com/kubermatic/kubermatic/issues/2148) ([alvaroaleman](https://github.com/alvaroaleman))
- The EOL Kubernetes 1.9 is no longer supported [#2252](https://github.com/kubermatic/kubermatic/issues/2252) ([kdomanski](https://github.com/kdomanski))
- S3 metrics exporter has been moved out of the kubermatic chart into its own chart [#2256](https://github.com/kubermatic/kubermatic/issues/2256) ([xrstf](https://github.com/xrstf))
- Displaying the terms of service can now be toggled in values.yaml [#2277](https://github.com/kubermatic/kubermatic/issues/2277) ([kgroschoff](https://github.com/kgroschoff))
- [ACTION REQUIRED] added a new command line flag to API server that accepts a set of key=value pairs that enables/disables various features. Existing `enable-prometheus-endpoint` flag is deprecated, the users should use `-feature-gates=PrometheusEndpoint=true` instead.  [#2278](https://github.com/kubermatic/kubermatic/issues/2278) ([p0lyn0mial](https://github
.com/p0lyn0mial))
- etcd readiness check timeouts have been increased [#2312](https://github.com/kubermatic/kubermatic/issues/2312) ([mrIncompetent](https://github.com/mrIncompetent))
- Removed unused fields from cloud specs exposed in the API [#2314](https://github.com/kubermatic/kubermatic/issues/2314) ([maciaszczykm](https://github.com/maciaszczykm))
- Kubermatic now validates nodes synchronously [#2340](https://github.com/kubermatic/kubermatic/issues/2340) ([alvaroaleman](https://github.com/alvaroaleman))
- Kubermatic now manages Nodes as group via the NodeGroup feature [#2357](https://github.com/kubermatic/kubermatic/issues/2357) ([maciaszczykm](https://github.com/maciaszczykm))
- Components will no longer be shown as as unhealthy when only some replicas are up [#2358](https://github.com/kubermatic/kubermatic/issues/2358) ([mrIncompetent](https://github.com/mrIncompetent))
- Kubernetes API servers can now be used with OpenID authentication
  - [ACTION REQUIRED] to enable the OpenID for kubernetes API server the users must set `-feature-gates=OpenIDConnectTokens=true` and provide `-oidc-issuer-url`, `-oidc-issuer-client-id` when running the controller. [#2370](https://github.com/kubermatic/kubermatic/issues/2370) ([zreigz](https://git
hub.com/zreigz))
- [ACTION REQUIRED] Resource limits for control plane containers have been increased. This might require additional resources for the seed cluster [#2395](https://github.com/kubermatic/kubermatic/issues/2395) ([mrIncompetent](https://github.com/mrIncompetent))
  - Kubernetes API server: 4Gi RAM, 2 CPU
  - Kubernetes Controller Manager: 2Gi RAM, 2 CPU
  - Kubernetes scheduler: 512Mi RAM, 1 CPU
  - CoreDNS: 128Mi RAM, 0.1 CPU
  - etcd: 2Gi RAM, 2 CPU
  - kube state metrics: 1Gi, 0.1 CPU
  - OpenVPN: 128Mi RAM, 0.1 CPU
  - Prometheus: 1Gi RAM, 0.1 CPU
- [ACTION_REQUIRED] Kubermatic CustomResourceDefinitions have been extracted out of the helm chart. This requires the execution of the `charts/kubermatic/migrate/migrate-kubermatic-chart.sh` script in case the CRD&#39;s where installed without the `&#34;helm.sh/resource-policy&#34;: keep` annotation. [#2459](https://github.com/kubermatic/kubermatic/issues/2459
) ([mrIncompetent](https://github.com/mrIncompetent))
- Control plane components are no longer logging at debug level [#2471](https://github.com/kubermatic/kubermatic/issues/2471) ([mrIncompetent](https://github.com/mrIncompetent))
- Experimantal support for VerticalPodAutoscaler has been added. The VPA resources use the PodUpdatePolicy=initial [#2505](https://github.com/kubermatic/kubermatic/issues/2505) ([mrIncompetent](https://github.com/mrIncompetent))
- Added 1.11.6 &amp; 1.12.4 to supported Kubernetes versions [#2537](https://github.com/kubermatic/kubermatic/issues/2537) ([mrIncompetent](https://github.com/mrIncompetent))
- It's now possible to rename a project [#2588](https://github.com/kubermatic/kubermatic/issues/2588) ([glower](https://github.com/glower))
- It is now possible for a user to select whether PVCs/PVs and/or LBs should be cleaned up when deleting the cluster. [#2604](https://github.com/kubermatic/kubermatic/issues/2604) ([zreigz](https://
github.com/zreigz))
- Credentials for Docker Hub are no longer necessary. [#2605](https://github.com/kubermatic/kubermatic/issues/2605) ([kdomanski](https://github.com/kdomanski))
- Added support for Heptio Ark-based backups [#2617](https://github.com/kubermatic/kubermatic/issues/2617) ([xrstf](https://github.com/xrstf))
- Running `kubectl get cluster` in a seed now shows some more details [#2622](https://github.com/kubermatic/kubermatic/issues/2622) ([alvaroaleman](https://github.com/alvaroaleman))
- Kubernetes 1.10 was removed as officially supported version from Kubermatic as its EOL [#2712](https://github.com/kubermatic/kubermatic/issues/2712) ([alvaroaleman](https://github.com/alvaroaleman))
- Updated machine controller to `v0.10.5` [#2490](https://github.com/kubermatic/kubermatic/issues/2490) ([mrIncompetent](https://github.com/mrIncompetent))
- Updated dex to 2.12.0 [#2318](https://github.com/kubermatic/kubermatic/issues/2318) ([bashofmann](https://github.com/bashofmann))
- Updated nginx-ingress-controller to `v0.22.0` [#2668](https://github.com/kubermatic/kubermatic/issues/2668) ([xrstf](https://github.com/xrstf))
- [ACTION REQUIRED] Updated cert-manager to `v0.6.0` (see https://cert-manager.readthedocs.io/en/latest/admin/upgrading/index.html) [#2674](https://github.com/kubermatic/kubermatic/issues/2674) ([xrstf](https://github.com/xrstf))


**Dashboard**:
- It is now possible to edit the project name in UI. [#1003](https://github.com/kubermatic/dashboard-v2/issues/1003) ([kgroschoff](https://github.com/kgroschoff))
- Machine Networks for VSphere can now be set in the UI [#829](https://github.com/kubermatic/dashboard-v2/issues/829) ([kgroschoff](https://github.com/kgroschoff))
- VSphere: Setting a dedicated VSphere user for cloud provider functionalities is now possible. [#834](https://github.com/kubermatic/dashboard-v2/issues/834) ([kgroschoff](https://github.com/kgroschoff))
- Fixed that the cluster upgrade link did not appear directly when the details page is loaded [#836](https://github.com/kubermatic/dashboard-v2/issues/836) ([bashofmann](https://github.com/bashofmann))
- Kubeconfig can now be shared via a generated link from the UI [#857](https://github.com/kubermatic/dashboard-v2/issues/857) ([kgroschoff](https://github.com/kgroschoff))
    - See https://docs.kubermatic.io/advanced/oidc_auth/ for more information.
- Fixed duplicated SSH keys in summary view during cluster creation. [#879](https://github.com/kubermatic/dashboard-v2/issues/879) ([kgroschoff](https://github.com/kgroschoff))
- On project change, the user will stay on the same page, if he has the corresponding rights. [#889](https://github.com/kubermatic/dashboard-v2/issues/889) ([kgroschoff](https://github.com/kgroschoff))
- Fixed issues with caching the main page. [#893](https://github.com/kubermatic/dashboard-v2/issues/893) ([maciaszczykm](https://github.com/maciaszczykm))
- Nodes are now being managed as NodeDeployments, this allows to easily change settings for a group of Nodes. [#949](https://github.com/kubermatic/dashboard-v2/issues/949) ([maciaszczykm](https://github.com/maciaszczykm))
- Removed Container Runtime selection, which is no longer supported. [#828](https://github.com/kubermatic/dashboard-v2/issues/828) ([bashofmann](https://github.com/bashofmann))
- Menu entries will be disabled as long as selected project is not in active state.
- Selected project state icon was added in the project selector and in the list view.
- Input field inside add project dialog will be automatically focused after opening dialog.
- After adding new project user will be redirected to project list [#808](https://github.com/kubermatic/dashboard-v2/issues/808) ([maciaszczykm](https://github.com/maciaszczykm))
- Notifications timeout is now 10s.
- Close and copy to clipboard actions are available on notifications. [#798](https://github.com/kubermatic/dashboard-v2/issues/798) ([maciaszczykm](https://github.com/maciaszczykm))
- Provider-specific data will now be fetched without re-sending credentials. [#814](https://github.com/kubermatic/dashboard-v2/issues/814) ([maciaszczykm](https://github.com/maciaszczykm))
- Various minor visual improvements


**Monitoring:**

- Version v1.11.0 - 1.11.3 Clusters will no longer gather `rest_*` metrics from the controller-manager due to a [bug in kubernetes](https://github.com/kubernetes/kubernetes/pull/68530) [#2020](https://github.com/kubermatic/kubermatic/issues/2020) ([cbeneke](https://github.com/cbeneke))
- Enabled scraping of user cluster resources [#2149](https://github.com/kubermatic/kubermatic/issues/2149) ([thetechnick](https://github.com/thetechnick))
- Prometheus is now scraping user clustersNew `kubermatic-controller-manager` flag `monitoring-scrape-annotation-prefix` [#2219](https://github.com/kubermatic/kubermatic/issues/2219) ([thetechnick](https://github.com/thetechnick))
- UserCluster Prometheus: decreased storage.tsdb.retention to 1h [#2246](https://github.com/kubermatic/kubermatic/issues/2246) ([thetechnick](https://github.com/thetechnick))
- Add datacenter label to kubermatic_cluster_info metric [#2248](https://github.com/kubermatic/kubermatic/issues/2248) ([kron4eg](https://github.com/kron4eg))
- Fixed the trigger condition for `EtcdInsufficientMembers` alert [#2262](https://github.com/kubermatic/kubermatic/issues/2262) ([cbeneke](https://github.com/cbeneke))
- [ACTION REQUIRED] move the metrics-server into the seed cluster. The metrics-server addon must be removed from the list of addons to install. [#2320](https://github.com/kubermatic/kubermatic/issues/2320) ([mrIncompetent](https://github.com/mrIncompetent))
- ArkNoRecentBackup alert does not trigger on backups that are not part of a schedule [#2351](https://github.com/kubermatic/kubermatic/issues/2351) ([bashofmann](https://github.com/bashofmann))
- fluentd has been replaced with fluentbit [#2469](https://github.com/kubermatic/kubermatic/issues/2469) ([mrIncompetent](https://github.com/mrIncompetent))
- Cluster Prometheus resource requests and limits are now configurable in cluster resource [#2576](https://github.com/kubermatic/kubermatic/issues/2576) ([bashofmann](https://github.com/bashofmann))
- Alerts for for control-plane components now reside in cluster namespaces [#2583](https://github.com/kubermatic/kubermatic/issues/2583) ([xrstf](https://github.com/xrstf))
- Updated kube-state-metrics to 1.5.0 [#2627](https://github.com/kubermatic/kubermatic/issues/2627) ([xrstf](https://github.com/xrstf))
- Updated Prometheus to `v2.6.0` [#2597](https://github.com/kubermatic/kubermatic/issues/2597) ([xrstf](https://github.com/xrstf))
- Updated alertmanager to `v0.16` [#2661](https://github.com/kubermatic/kubermatic/issues/2661) ([xrstf](https://github.com/xrstf))
- Updated Grafana to `v5.4.3` [#2662](https://github.com/kubermatic/kubermatic/issues/2662) ([xrstf](https://github.com/xrstf))
- Updated node-exporter to `v0.17` (**note: breaking changes to metric names might require updates to customized dashboards**) [#2666](https://github.com/kubermatic/kubermatic/issues/2666) ([xrstf](https://github.com/xrstf))
- Updated Minio to `RELEASE.2019-01-16T21-44-08Z` [#2667](https://github.com/kubermatic/kubermatic/issues/2667) ([xrstf](https://github.com/xrstf))
- metrics-server will use 2 replicas [#2707](https://github.com/kubermatic/kubermatic/issues/2707) ([mrIncompetent](https://github.com/mrIncompetent))


**Security:**

- The admin token can no longer be read through the Kubermatic API. [#2105](https://github.com/kubermatic/kubermatic/issues/2105) ([p0lyn0mial](https://github.com/p0lyn0mial))
- Communicating with cloud providers through the project APIs no longer requires providing additional credentials. [#2180](https://github.com/kubermatic/kubermatic/issues/2180) ([p0lyn0mial](https://github.com/p0lyn0mial))
- Kubernetes will be automatically updated to versions that contain a fix for CVE-2018-1002105 [#2478](https://github.com/kubermatic/kubermatic/issues/2478) ([alvaroaleman](https://github.com/alvaroaleman))


**Bugfix:**

- Missing upgrade paths for K8S 1.10 and 1.11 have been addded. [#2159](https://github.com/kubermatic/kubermatic/issues/2159) ([mrIncompetent](https://github.com/mrIncompetent))
- Fixed migration of users from older versions of Kubermatic [#2294](https://github.com/kubermatic/kubermatic/issues/2294) ([mrIncompetent](https://github.com/mrIncompetent))
- Updated machine-controller to `v0.9.9`Fixed a bug in the machine-migration that caused cloud provider instances to not be properly identified anymore [#2307](https://github.com/kubermatic/kubermatic/issues/2307) ([alvaroaleman](https://github.com/alvaroaleman))
- Fixd missing permissions in kube-state-metrics ClusterRole [#2366](https://github.com/kubermatic/kubermatic/issues/2366) ([bashofmann](https://github.com/bashofmann))
- Missing ca-certificates have been added to s3-exporter image [#2464](https://github.com/kubermatic/kubermatic/issues/2464) ([bashofmann](https://github.com/bashofmann))
- Adedd missing configmap checksums to kubermatic-controller-manager chart [#2492](https://github.com/kubermatic/kubermatic/issues/2492) ([bashofmann](https://github.com/bashofmann))
- cloud-config files are now properly escaped [#2498](https://github.com/kubermatic/kubermatic/issues/2498) ([alvaroaleman](https://github.com/alvaroaleman))
- SSH keys can no longer be added with duplicate names [#2499](https://github.com/kubermatic/kubermatic/issues/2499) ([kgroschoff](https://github.com/kgroschoff))
- Fixed an issue with kubelets being unreachable by the apiserver on some OS configurations. [#2522](https://github.com/kubermatic/kubermatic/issues/2522) ([mrIncompetent](https://github.com/mrIncompetent))
- Timestamp format has been unified throughout the Kubermatic API. [#2534](https://github.com/kubermatic/kubermatic/issues/2534) ([zreigz](https://github.com/zreigz))
- Updated cert-manager to fix an issue which caused re-issuing of a certficate via the http01 challenge to fail [#2658](https://github.com/kubermatic/kubermatic/issues/2658) ([alvaroaleman](https://github.com/alvaroaleman))
- Nodes and NodeDeployments can no longer be configured to provision kubelets at versions incompatible with the control plane. [#2665](https://github.com/kubermatic/kubermatic/issues/2665) ([kdomanski](https://github.com/kdomanski))




### [v2.8.6]()

- Added support for Kubernetes `v1.13` [#2628](https://github.com/kubermatic/kubermatic/issues/2628) ([mrIncompetent](https://github.com/mrIncompetent))
- Fixed reconciling of deep objects [#2630](https://github.com/kubermatic/kubermatic/issues/2630) ([mrIncompetent](https://github.com/mrIncompetent))




### [v2.8.5]()

- Added `1.11.6` and `1.12.4` to supported versions [#2538](https://github.com/kubermatic/kubermatic/issues/2538) ([mrIncompetent](https://github.com/mrIncompetent))




### [v2.8.4]()

- Fixed an issue with kubelets being unreachable by the apiserver on some OS configurations. [#2522](https://github.com/kubermatic/kubermatic/issues/2522) ([mrIncompetent](https://github.com/mrIncompetent))




### [v2.8.3]()

Supported Kubernetes versions:
- `1.10.11`
- `1.11.5`
- `1.12.3`


**Misc:**

- Kubermatic now validates nodes synchronously [#2340](https://github.com/kubermatic/kubermatic/issues/2340) ([alvaroaleman](https://github.com/alvaroaleman))
- Components will no longer be shown as as unhealthy when only some replicas are up [#2358](https://github.com/kubermatic/kubermatic/issues/2358) ([mrIncompetent](https://github.com/mrIncompetent))
- Disabled debug logs on control plane components [#2471](https://github.com/kubermatic/kubermatic/issues/2471) ([mrIncompetent](https://github.com/mrIncompetent))
- Kubernetes will be automatically updated to versions that contain a fix for CVE-2018-1002105 [#2478](https://github.com/kubermatic/kubermatic/issues/2478) ([alvaroaleman](https://github.com/alvaroaleman))
- Updated the machine-controller to `v0.10.3` [#2479](https://github.com/kubermatic/kubermatic/issues/2479) ([mrIncompetent](https://github.com/mrIncompetent))

**Bugfix:**
- Fixed missing permissions in kube-state-metrics ClusterRole [#2366](https://github.com/kubermatic/kubermatic/issues/2366) ([bashofmann](https://github.com/bashofmann))




### [v2.7.8]()

Supported Kubernetes versions:
- `1.10.11`
- `1.11.5`


**Major changes:**

- Communicating with cloud providers APIs no longer requires providing additional credentials. [#2151](https://github.com/kubermatic/kubermatic/issues/2151) ([p0lyn0mial](https://github.com/p0lyn0mial))
- Updated the kubermatic dashboard to `v0.38.0` [#2165](https://github.com/kubermatic/kubermatic/issues/2165) ([mrIncompetent](https://github.com/mrIncompetent))
  - Provider-specific data will now be fetched without re-sending credentials. [#806](https://github.com/kubermatic/dashboard-v2/issues/806) ([maciaszczykm](https://github.com/maciaszczykm))
- Kubernetes will be automatically updated to versions that contain a fix for CVE-2018-1002105 and `v1.8`, `v1.9` cluster creation is now dropped [#2487](https://github.com/kubermatic/kubermatic/issues/2487) ([kdomanski](https://github.com/kdomanski))




### [v2.6.17]()

Supported Kubernetes versions:
- `1.10.11`


**Bugfix:**

- Fixed handling of very long user IDs [#2086](https://github.com/kubermatic/kubermatic/issues/2086) ([kdomanski](https://github.com/kdomanski))


**Misc:**

- Enabled the usage of Heapster for the HorizontalPodAutoscaler [#2199](https://github.com/kubermatic/kubermatic/issues/2199) ([mrIncompetent](https://github.com/mrIncompetent))
- Kubernetes will be automatically updated to versions that contain a fix for CVE-2018-1002105 and `v1.8`, `v1.9` cluster creation is now dropped [#2497](https://github.com/kubermatic/kubermatic/issues/2497) ([kdomanski](https://github.com/kdomanski))




### [v2.8.2]()


- Fixed migration of users from older versions of Kubermatic [#2294](https://github.com/kubermatic/kubermatic/issues/2294) ([mrIncompetent](https://github.com/mrIncompetent))
- Fixed a bug in the machine-migration that caused cloud provider instances to not be properly identified anymore [#2307](https://github.com/kubermatic/kubermatic/issues/2307) ([alvaroaleman](https://github.com/alvaroaleman))
- Increased etcd readiness check timeout [#2312](https://github.com/kubermatic/kubermatic/issues/2312) ([mrIncompetent](https://github.com/mrIncompetent))
- Updated machine-controller to `v0.9.9`




### [v2.8.1]()


**Misc:**

- Prometheus is now scraping user clusters [#2219](https://github.com/kubermatic/kubermatic/issues/2219) ([thetechnick](https://github.com/thetechnick))
- Updated the Kubermatic dashboard to `v1.0.2` [#2263](https://github.com/kubermatic/kubermatic/issues/2263) ([mrIncompetent](https://github.com/mrIncompetent))
- Update machine controller to `v0.9.8` [#2275](https://github.com/kubermatic/kubermatic/issues/2275) ([mrIncompetent](https://github.com/mrIncompetent))


**Dashboard:**

- Removed Container Runtime selection, which is no longer supported. [#828](https://github.com/kubermatic/dashboard-v2/issues/828) ([bashofmann](https://github.com/bashofmann))
- Various minor visual improvements




### [v2.8.0]()

Supported Kubernetes versions:
- `1.9.0` - `1.9.10`
- `1.10.0` - `1.10.8`
- `1.11.0` - `1.11.3`
- `1.12.0` - `1.12.1`


**Major changes:**


- Implemented user/project management
- Old clusters will be automatically migrated to each user&#39;s default project [#1829](https://github.com/kubermatic/kubermatic/issues/1829) ([p0lyn0mial](https://github.com/p0lyn0mial))
- Kubermatic now supports Kubernetes 1.12 [#2132](https://github.com/kubermatic/kubermatic/issues/2132) ([alvaroaleman](https://github.com/alvaroaleman))


**Dashboard:**

- The UI has been reworked for the new user/project management
- Fixed error appearing when trying to change selected OS [#699](https://github.com/kubermatic/dashboard-v2/issues/699) ([kgroschoff](https://github.com/kgroschoff))
- Openstack: fixed an issue, where list of tenants wouldn&#39;t get loaded when returning from summary page [#705](https://github.com/kubermatic/dashboard-v2/issues/705) ([kgroschoff](https://github.com/kgroschoff))
- Fixed confirmation of cluster deletion [#718](https://github.com/kubermatic/dashboard-v2/issues/718) ([kgroschoff](https://github.com/kgroschoff))
- Fixed the link to Kubernetes dashboard [#740](https://github.com/kubermatic/dashboard-v2/issues/740) ([guusvw](https://github.com/guusvw))
- Openstack: show selected image in cluster creation summary [#698](https://github.com/kubermatic/dashboard-v2/issues/698) ([bashofmann](https://github.com/bashofmann))
- vSphere: custom cluster vnet can now be selected [#708](https://github.com/kubermatic/dashboard-v2/issues/708) ([kgroschoff](https://github.com/kgroschoff))
- Openstack: the list of available networks and floating IP pools will be loaded from the API [#737](https://github.com/kubermatic/dashboard-v2/issues/737) ([j3ank](https://github.com/j3ank))
- Dashboard metrics can now be collected by Prometheus [#678](https://github.com/kubermatic/dashboard-v2/issues/678) ([pkavajin](https://github.com/pkavajin))
- Redesigned cluster creation summary page [#688](https://github.com/kubermatic/dashboard-v2/issues/688) ([kgroschoff](https://github.com/kgroschoff))
- Default template images for Openstack and vSphere are now taken from datacenter configuration [#689](https://github.com/kubermatic/dashboard-v2/issues/689) ([kgroschoff](https://github.com/kgroschoff))
- Fixed cluster settings view for Openstack [#746](https://github.com/kubermatic/dashboard-v2/issues/746) ([kgroschoff](https://github.com/kgroschoff))
- &#34;Upgrade Cluster&#34; link is no longer available for clusters that have no updates available or are not ready [#750](https://github.com/kubermatic/dashboard-v2/issues/750) ([bashofmann](https://github.com/bashofmann))
- Fixed initial nodes data being lost when the browser tab was closed right after cluster creation [#796](https://github.com/kubermatic/dashboard-v2/issues/796) ([kgroschoff](https://github.com/kgroschoff))
- Google Analytics code can now be optionally added by the administrator [#742](https://github.com/kubermatic/dashboard-v2/issues/742) ([bashofmann](https://github.com/bashofmann))
- OpenStack tenant can now be either chosen from dropdown or typed in by hand [#759](https://github.com/kubermatic/dashboard-v2/issues/759) ([kgroschoff](https://github.com/kgroschoff))
- vSphere: Network can now be selected from a list [#771](https://github.com/kubermatic/dashboard-v2/issues/771) ([kgroschoff](https://github.com/kgroschoff))
- Login token is now removed from URL for security reasons [#790](https://github.com/kubermatic/dashboard-v2/issues/790) ([bashofmann](https://github.com/bashofmann))
- `Admin` button has been removed from `Certificates and Keys` panel as it allowed to copy the admin token into the clipboard. Since this is a security concern we decided to remove this functionality. [#800](https://github.com/kubermatic/dashboard-v2/issues/800) ([p0lyn0mial](https://github.com/p0lyn0mial))
- Notifications timeout is now 10s
- Close and copy to clipboard actions are available on notifications. [#798](https://github.com/kubermatic/dashboard-v2/issues/798) ([maciaszczykm](https://github.com/maciaszczykm))
- Provider-specific data will now be fetched without re-sending credentials. [#814](https://github.com/kubermatic/dashboard-v2/issues/814) ([maciaszczykm](https://github.com/maciaszczykm))
- Various minor fixes and improvements


**Bugfix:**

- Kubernetes aggregation layer now uses a dedicated CA [#1787](https://github.com/kubermatic/kubermatic/issues/1787) ([mrIncompetent](https://github.com/mrIncompetent))
- fixed DNS/scheduler/controller-manager alerts in Prometheus [#1908](https://github.com/kubermatic/kubermatic/issues/1908) ([xrstf](https://github.com/xrstf))
- fixed bad rules.yaml format for Prometheus [#1924](https://github.com/kubermatic/kubermatic/issues/1924) ([xrstf](https://github.com/xrstf))
- Add missing RoleBinding for bootstrap tokens created with `kubeadm token create` [#1943](https://github.com/kubermatic/kubermatic/issues/1943) ([mrIncompetent](https://github.com/mrIncompetent))
- Fixed handling of very long user IDs [#2075](https://github.com/kubermatic/kubermatic/issues/2075) ([mrIncompetent](https://github.com/mrIncompetent))
- The API server will redact sensitive data from its legacy API responses. [#2079](https://github.com/kubermatic/kubermatic/issues/2079) ([p0lyn0mial](https://github.com/p0lyn0mial)), [#2087](https://github.com/kubermatic/kubermatic/issues/2087) ([p0lyn0mial](https://github.com/p0lyn0mial))
- Missing upgrade paths for K8S 1.10 and 1.11 have been addded. [#2159](https://github.com/kubermatic/kubermatic/issues/2159) ([mrIncompetent](https://github.com/mrIncompetent))


**Misc:**

- Added a controller for static ip address management [#1616](https://github.com/kubermatic/kubermatic/issues/1616) ([pkavajin](https://github.com/pkavajin))
- Activated kubelet certificate rotation feature flags [#1771](https://github.com/kubermatic/kubermatic/issues/1771) ([mrIncompetent](https://github.com/mrIncompetent))
- Made s3-exporter endpoint configurable [#1772](https://github.com/kubermatic/kubermatic/issues/1772) ([bashofmann](https://github.com/bashofmann))
- etcd StatefulSet uses default timings again [#1776](https://github.com/kubermatic/kubermatic/issues/1776) ([mrIncompetent](https://github.com/mrIncompetent))
- Breaking change: basic auth for kibana/grafana/prometheus/alertmanager has been replaced with oAuth [#1808](https://github.com/kubermatic/kubermatic/issues/1808) ([kron4eg](https://github.com/kron4eg))
- Added a controller which steers control plane traffic to the kubelets via VPN.  [#1817](https://github.com/kubermatic/kubermatic/issues/1817) ([thz](https://github.com/thz))
- Fixed a memory leak which occurs when using credentials for a container registry. [#1850](https://github.com/kubermatic/kubermatic/issues/1850) ([thz](https://github.com/thz))
- Combined ImagePullSecrets im the Kubermatic chart [#1877](https://github.com/kubermatic/kubermatic/issues/1877) ([mrIncompetent](https://github.com/mrIncompetent))
- Include cluster name as label on each pod [#1891](https://github.com/kubermatic/kubermatic/issues/1891) ([mrIncompetent](https://github.com/mrIncompetent))
- Ark-based seed-cluster backup infrastructure [#1894](https://github.com/kubermatic/kubermatic/issues/1894) ([xrstf](https://github.com/xrstf))
- Add AntiAffinity to the control plane pods to prevent scheduling of the same kind pod on the same node. [#1895](https://github.com/kubermatic/kubermatic/issues/1895) ([mrIncompetent](https://github.com/mrIncompetent))
- Enabled etcd auto-compaction [#1932](https://github.com/kubermatic/kubermatic/issues/1932) ([mrIncompetent](https://github.com/mrIncompetent))
- etcd in user cluser namespaces is defragmented every 3 hours [#1935](https://github.com/kubermatic/kubermatic/issues/1935) ([xrstf](https://github.com/xrstf))
- DNS names are now used inside the cluster namespaces, Scoped to the cluster namespace [#1959](https://github.com/kubermatic/kubermatic/issues/1959) ([mrIncompetent](https://github.com/mrIncompetent))
- Increased kubectl timeouts on AWS  [#1983](https://github.com/kubermatic/kubermatic/issues/1983) ([pkavajin](https://github.com/pkavajin))
- Support for Kubernetes v1.8 has been dropped. The control planes of all clusters running 1.8 will be automatically updated [#2013](https://github.com/kubermatic/kubermatic/issues/2013) ([mrIncompetent](https://github.com/mrIncompetent))
- OpenVPN status is now a part of cluster health [#2038](https://github.com/kubermatic/kubermatic/issues/2038) ([mrIncompetent](https://github.com/mrIncompetent))
- Improved detection of user-cluster apiserver health on startup [#2052](https://github.com/kubermatic/kubermatic/issues/2052) ([thz](https://github.com/thz))
- Kubermatic now uses the types from the [cluster api project](https://github.com/kubernetes-sigs/cluster-api) to manage nodes [#2056](https://github.com/kubermatic/kubermatic/issues/2056) ([alvaroaleman](https://github.com/alvaroaleman))
- CPU&amp;Memory limit for the Kubermatic controller manager deployment has been increased [#2081](https://github.com/kubermatic/kubermatic/issues/2081) ([mrIncompetent](https://github.com/mrIncompetent))
- controller-manager and its controllers will no longer run with cluster-admin permissions [#2096](https://github.com/kubermatic/kubermatic/issues/2096) ([alvaroaleman](https://github.com/alvaroaleman))
- PodDisruptionBudget is now configured for the API server deployment [#2098](https://github.com/kubermatic/kubermatic/issues/2098) ([mrIncompetent](https://github.com/mrIncompetent))
- The kubermatic-master chart has been merged into the main kubermatic chart [#2103](https://github.com/kubermatic/kubermatic/issues/2103) ([alvaroaleman](https://github.com/alvaroaleman))
- Version v1.11.0 - 1.11.3 Clusters will no longer gather `rest_*` metrics from the controller-manager due to a [bug in kubernetes](https://github.com/kubernetes/kubernetes/pull/68530) [#2020](https://github.com/kubermatic/kubermatic/issues/2020) ([cbeneke](https://github.
com/cbeneke))
- Communicating with cloud providers through the non-project APIs no longer requires providing additional credentials. [#2156](https://github.com/kubermatic/kubermatic/issues/2156) ([p0lyn0mial](https://github.com/p0lyn0mial))
- Communicating with cloud providers through the project APIs no longer requires providing additional credentials. [#2227](https://github.com/kubermatic/kubermatic/issues/2227) ([p0lyn0mial](https://github.com/p0lyn0mial))
- Updated dashboard to `v1.0.1` [#2228](https://github.com/kubermatic/kubermatic/issues/2228) ([mrIncompetent](https://github.com/mrIncompetent))
- Updated kubernetes-dashboard addon to `1.10.0` [#1874](https://github.com/kubermatic/kubermatic/issues/1874) ([bashofmann](https://github.com/bashofmann))
- Updated nginx ingress controller to `0.18.0` [#1800](https://github.com/kubermatic/kubermatic/issues/1800) ([bashofmann](https://github.com/bashofmann))
- Updated etcd to `v3.3.9` [#1961](https://github.com/kubermatic/kubermatic/issues/1961) ([mrIncompetent](https://github.com/mrIncompetent))
- Updated machine-controller to `v0.9.5` [#2224](https://github.com/kubermatic/kubermatic/issues/2224)
([mrIncompetent](https://github.com/mrIncompetent))
- updated cert-manager to `0.4.1` [#1925](https://github.com/kubermatic/kubermatic/issues/1925) ([xrstf](https://github.com/xrstf))
- Updated Prometheus to `v2.3.2` [#1830](https://github.com/kubermatic/kubermatic/issues/1830) ([mrIncompetent](https://github.com/mrIncompetent))
- Updated dex to `2.11.0` [#1986](https://github.com/kubermatic/kubermatic/issues/1986) ([bashofmann](https://github.com/bashofmann))
- Updated kube-proxy addon to match the cluster version [#2017](https://github.com/kubermatic/kubermatic/issues/2017) ([mrIncompetent](https://github.com/mrIncompetent))


**Monitoring:**

- Grafana dashboards now use the latest kubernetes-mixin dashboards. [#1705](https://github.com/kubermatic/kubermatic/issues/1705) ([metalmatze](https://github.com/metalmatze))
- nginx ingress controller metrics are now scraped [#1777](https://github.com/kubermatic/kubermatic/issues/1777) ([bashofmann](https://github.com/bashofmann))
- annotations will be used instead of labels for the nginx-ingress Prometheus configuration [#1823](https://github.com/kubermatic/kubermatic/issues/1823) ([xrstf](https://github.com/xrstf))
- `KubePersistentVolumeFullInFourDays` will only be predicted when there is at least 6h of historical data available [#1862](https://github.com/kubermatic/kubermatic/issues/1862) ([cbeneke](https://github.com/cbeneke))
- reorganized Grafana dashboards, including etcd dashboard [#1775](https://github.com/kubermatic/kubermatic/issues/1775) ([xrstf](https://github.com/xrstf))
- customizations of Grafana dashboard providers, datasources and dashboards themselves are now easier [#1812](https://github.com/kubermatic/kubermatic/issues/1812) ([xrstf](https://github.com/xrstf))
- new Prometheus and Kubernetes Volumes dashboards [#1838](https://github.com/kubermatic/kubermatic/issues/1838) ([xrstf](https://github.com/xrstf))
- Prometheus in the seed cluster can now be customized by extending the Helm chart&#39;s `values.yaml` [#1801](https://github.com/kubermatic/kubermatic/issues/1801) ([xrstf](https://github.com/xrstf))
- Prometheus alerts can now be customized in cluster namespaces [#1831](https://github.com/kubermatic/kubermatic/issues/1831) ([pkavajin](https://github.com/pkavajin))
- Added a way to customize scraping configs for in-cluster-namespace-prometheuses [#1837](https://github.com/kubermatic/kubermatic/issues/1837) ([pkavajin](https://github.com/pkavajin))




### [v2.7.7]()


**Misc:**

- Removed functionality to copy the admin token in the dashboard [#2083](https://github.com/kubermatic/kubermatic/issues/2083) ([mrIncompetent](https://github.com/mrIncompetent))




### [v2.7.6]()


**Misc:**

- Various minor fixes and improvements




### [v2.7.5]()


**Bugfix:**

- Fixed handling of very long user IDs [#2070](https://github.com/kubermatic/kubermatic/issues/2070) ([mrIncompetent](https://github.com/mrIncompetent))




### [v2.7.4]()


**Bugfix:**

- Updated machine controller to `v0.7.23`: write permissions on vSphere datacenters are no longer needed. [#2069](https://github.com/kubermatic/kubermatic/issues/2069) ([pkavajin](https://github.com/pkavajin))




### [v2.7.3]()


**Misc:**

- kube-proxy addon was updated to match the cluster version [#2019](https://github.com/kubermatic/kubermatic/issues/2019) ([mrIncompetent](https://github.com/mrIncompetent))




### [v2.7.2]()


**Monitoring:**

- `KubePersistentVolumeFullInFourDays` will only be predicted when there is at least 6h of historical data available [#1862](https://github.com/kubermatic/kubermatic/issues/1862) ([cbeneke](https://github.com/cbeneke))


**Misc:**

- Updated machine-controller to `v0.7.22` [#1999](https://github.com/kubermatic/kubermatic/issues/1999) ([mrIncompetent](https://github.com/mrIncompetent))




### [v2.7.1]()


**Bugfix:**

- fixed DNS/scheduler/controller-manager alerts in Prometheus [#1908](https://github.com/kubermatic/kubermatic/issues/1908) ([xrstf](https://github.com/xrstf))
- fix bad rules.yaml format for Prometheus [#1924](https://github.com/kubermatic/kubermatic/issues/1924) ([xrstf](https://github.com/xrstf))
- Add missing RoleBinding for bootstrap tokens created with `kubeadm token create` [#1943](https://github.com/kubermatic/kubermatic/issues/1943) ([mrIncompetent](https://github.com/mrIncompetent))
- Fix bug with endless resource updates being triggered due to a wrong comparison [#1964](https://github.com/kubermatic/kubermatic/issues/1964) ([mrIncompetent](https://github.com/mrIncompetent))
- Fix escaping of special characters in the cloud-config [#1976](https://github.com/kubermatic/kubermatic/issues/1976) ([mrIncompetent](https://github.com/mrIncompetent))


**Misc:**

- Update kubernetes-dashboard addon to `1.10.0` [#1874](https://github.com/kubermatic/kubermatic/issues/1874) ([bashofmann](https://github.com/bashofmann))
- Update machine-controller to `v0.7.21` [#1975](https://github.com/kubermatic/kubermatic/issues/1975) ([mrIncompetent](https://github.com/mrIncompetent))




### [v2.7.0]()


**Bugfix:**

- Fixed a rare issue with duplicate entries on the list of nodes [#1391](https://github.com/kubermatic/kubermatic/issues/1391) ([mrIncompetent](https://github.com/mrIncompetent))
- Fixed deletion of old etcd backups [#1394](https://github.com/kubermatic/kubermatic/issues/1394) ([mrIncompetent](https://github.com/mrIncompetent))
- Fix deadlock during backup cleanup when the etcd of the cluster never reached a healthy state. [#1612](https://github.com/kubermatic/kubermatic/issues/1612) ([mrIncompetent](https://github.com/mrIncompetent))
- Use dedicated CA for Kubernetes aggregation layer [#1787](https://github.com/kubermatic/kubermatic/issues/1787) ([mrIncompetent](https://github.com/mrIncompetent))


**Cloud Provider:**

- Non-ESXi vsphere hosts are now supported [#1306](https://github.com/kubermatic/kubermatic/issues/1306) ([alvaroaleman](https://github.com/alvaroaleman))
- VSphere target folder will be properly cleaned up on cluster deletion. [#1314](https://github.com/kubermatic/kubermatic/issues/1314) ([alvaroaleman](https://github.com/alvaroaleman))
- Fixed floating IP defaulting on openstack [#1332](https://github.com/kubermatic/kubermatic/issues/1332) ([mrIncompetent](https://github.com/mrIncompetent))
- Azure: added multi-AZ node support [#1354](https://github.com/kubermatic/kubermatic/issues/1354) ([mrIncompetent](https://github.com/mrIncompetent))
- Fixed premature logout from vsphere API [#1373](https://github.com/kubermatic/kubermatic/issues/1373) ([alvaroaleman](https://github.com/alvaroaleman))
- Image templates can now be configured in datacenter.yaml for Openstack and vSphere [#1397](https://github.com/kubermatic/kubermatic/issues/1397) ([mrIncompetent](https://github.com/mrIncompetent))
- AWS: allow multiple clusters per subnet/VPC [#1481](https://github.com/kubermatic/kubermatic/issues/1481) ([mrIncompetent](https://github.com/mrIncompetent))
- In a VSphere DC is is now possible to set a `infra_management_user` which when set will automatically be used for everything except the cloud provider functionality for all VSphere clusters in that DC.  [#1592](https://github.com/kubermatic/kubermatic/issues/1592) ([alvaroaleman](https://github.com/alvaroaleman))
- Always allocate public IP on new machines when using Azure [#1644](https://github.com/kubermatic/kubermatic/issues/1644) ([mrIncompetent](https://github.com/mrIncompetent))
- Add missing cloud provider flags on the apiserver and controller-manager for azure [#1646](https://github.com/kubermatic/kubermatic/issues/1646) ([mrIncompetent](https://github.com/mrIncompetent))
- Azure: fixed minor issue with seed clusters running on Azure [#1657](https://github.com/kubermatic/kubermatic/issues/1657) ([thz](https://github.com/thz))
- Create AvailabilitySet for Azure clusters and set it for each machine [#1661](https://github.com/kubermatic/kubermatic/issues/1661) ([mrIncompetent](https://github.com/mrIncompetent))
- OpenStack LoadBalancer manage-security-groups setting is set into cluster&#39;s cloud-config for Kubernetes versions where https://github.com/kubernetes/kubernetes/issues/58145 is fixed. [#1720](https://github.com/kubermatic/kubermatic/issues/1720) ([bashofmann](https://github.com/bashofmann))


**Misc:**

- Control plane can now reach the nodes via VPN [#1234](https://github.com/kubermatic/kubermatic/issues/1234) ([thz](https://github.com/thz))
- Addons in kubermatic charts can now be specified as a list [#1304](https://github.com/kubermatic/kubermatic/issues/1304) ([guusvw](https://github.com/guusvw))
- Added support for Kubernetes `1.8.14`, `1.9.8`, `1.9.9`, `1.10.4` and `1.10.5` [#1348](https://github.com/kubermatic/kubermatic/issues/1348) ([mrIncompetent](https://github.com/mrIncompetent))
- Enabled Mutating/Validating Admission Webhooks for K8S 1.9&#43; [#1352](https://github.com/kubermatic/kubermatic/issues/1352) ([alvaroaleman](https://github.com/alvaroaleman))
- Update addon manager to v0.1.0 [#1363](https://github.com/kubermatic/kubermatic/issues/1363) ([thz](https://github.com/thz))
- Master components can now talk to cluster DNS [#1379](https://github.com/kubermatic/kubermatic/issues/1379) ([thz](https://github.com/thz))
- Non-default IP can now be used for cluster DNS [#1393](https://github.com/kubermatic/kubermatic/issues/1393) ([glower](https://github.com/glower))
- SSH keypair can now be detached from a cluster [#1395](https://github.com/kubermatic/kubermatic/issues/1395) ([p0lyn0mial](https://github.com/p0lyn0mial))
- Removed Kubermatic API v2 [#1409](https://github.com/kubermatic/kubermatic/issues/1409) ([p0lyn0mial](https://github.com/p0lyn0mial))
- Added EFK stack in seed clusters [#1430](https://github.com/kubermatic/kubermatic/issues/1430) ([pkavajin](https://github.com/pkavajin))
- Fixed some issues with eleasticsearch [#1484](https://github.com/kubermatic/kubermatic/issues/1484) ([pkavajin](https://github.com/pkavajin))
- Master components will now talk to the apiserver over secure port [#1486](https://github.com/kubermatic/kubermatic/issues/1486) ([thz](https://github.com/thz))
- Added support for Kubernetes version 1.11.0 [#1493](https://github.com/kubermatic/kubermatic/issues/1493) ([alvaroaleman](https://github.com/alvaroaleman))
- Clients will now talk to etcd over TLS [#1495](https://github.com/kubermatic/kubermatic/issues/1495) ([mrIncompetent](https://github.com/mrIncompetent))
- Communication between apiserver and etcd is now encrypted [#1496](https://github.com/kubermatic/kubermatic/issues/1496) ([mrIncompetent](https://github.com/mrIncompetent))
- With the introduction of Kubermatic&#39;s addon manager, the K8S addon manager&#39;s deployments will be automatically cleaned up on old setups [#1513](https://github.com/kubermatic/kubermatic/issues/1513) ([mrIncompetent](https://github.com/mrIncompetent))
- controller-manager will now automatically restart on backup config change [#1548](https://github.com/kubermatic/kubermatic/issues/1548) ([bashofmann](https://github.com/bashofmann))
- The control plane now has its own DNS resolver [#1549](https://github.com/kubermatic/kubermatic/issues/1549) ([alvaroaleman](https://github.com/alvaroaleman))
- apiserver will now automatically restart on master-files change [#1552](https://github.com/kubermatic/kubermatic/issues/1552) ([cbeneke](https://github.com/cbeneke))
- Add missing reconciling of the OpenVPN config inside the user cluster [#1605](https://github.com/kubermatic/kubermatic/issues/1605) ([mrIncompetent](https://github.com/mrIncompetent))
- Add pod anti-affinity for the etcd StatefulSet [#1607](https://github.com/kubermatic/kubermatic/issues/1607) ([mrIncompetent](https://github.com/mrIncompetent))
- Add PodDisruptionBudget for the etcd StatefulSet [#1608](https://github.com/kubermatic/kubermatic/issues/1608) ([mrIncompetent](https://github.com/mrIncompetent))
- Add support for configuring component settings(Replicas &amp; Resources) via the cluster object [#1636](https://github.com/kubermatic/kubermatic/issues/1636) ([mrIncompetent](https://github.com/mrIncompetent))
- Update nodeport-proxy to v1.2 [#1640](https://github.com/kubermatic/kubermatic/issues/1640) ([mrIncompetent](https://github.com/mrIncompetent))
- Added  access to the private quay.io repos from the kubermatic helm template [#1652](https://github.com/kubermatic/kubermatic/issues/1652) ([glower](https://github.com/glower))
- the correct default StorageClass is now installed into the user cluster via an extra addon [#1670](https://github.com/kubermatic/kubermatic/issues/1670) ([glower](https://github.com/glower))
- Update machine-controller to `v0.7.18` [#1708](https://github.com/kubermatic/kubermatic/issues/1708) ([mrIncompetent](https://github.com/mrIncompetent))
- Add support for Kubernetes `1.9.10`, `1.10.6` and `1.11.1` [#1712](https://github.com/kubermatic/kubermatic/issues/1712) ([mrIncompetent](https://github.com/mrIncompetent))
- Add possibility to override the seed DNS name for a given node datacenter via the datacenters.yaml [#1715](https://github.com/kubermatic/kubermatic/issues/1715) ([mrIncompetent](https://github.com/mrIncompetent))
- Heapster is replaced by metrics-server. [#1730](https://github.com/kubermatic/kubermatic/issues/1730) ([glower](https://github.com/glower))
- Combine the two existing CA secrets into a single one [#1732](https://github.com/kubermatic/kubermatic/issues/1732) ([mrIncompetent](https://github.com/mrIncompetent))
- It is now possible to customize user cluster configmaps/secrets via a `MutatingAdmissionWebhook` [#1740](https://github.com/kubermatic/kubermatic/issues/1740) ([alvaroaleman](https://github.com/alvaroaleman))
- Make s3-exporter endpoint configurable [#1772](https://github.com/kubermatic/kubermatic/issues/1772) ([bashofmann](https://github.com/bashofmann))
- Update nginx ingress controller to 0.18.0 [#1800](https://github.com/kubermatic/kubermatic/issues/1800) ([bashofmann](https://github.com/bashofmann))


**Monitoring:**

- Fixed metric name for addon controller [#1323](https://github.com/kubermatic/kubermatic/issues/1323) ([alvaroaleman](https://github.com/alvaroaleman))
- Error metrics are now collected for Kubermatic API endpoints [#1376](https://github.com/kubermatic/kubermatic/issues/1376) ([pkavajin](https://github.com/pkavajin))
- Prometheus is now a Statefulset [#1399](https://github.com/kubermatic/kubermatic/issues/1399) ([metalmatze](https://github.com/metalmatze))
- Alert Manger is now a Statefulset [#1414](https://github.com/kubermatic/kubermatic/issues/1414) ([metalmatze](https://github.com/metalmatze))
- Fixed job labels for recording rules and alerts [#1415](https://github.com/kubermatic/kubermatic/issues/1415) ([metalmatze](https://github.com/metalmatze))
- Added official etcd alerts [#1417](https://github.com/kubermatic/kubermatic/issues/1417) ([metalmatze](https://github.com/metalmatze))
- Added an S3 exporter for metrics [#1482](https://github.com/kubermatic/kubermatic/issues/1482) ([alvaroaleman](https://github.com/alvaroaleman))
- Added alert rule for machines which stuck in deletion [#1606](https://github.com/kubermatic/kubermatic/issues/1606) ([mrIncompetent](https://github.com/mrIncompetent))
- The customer cluster Prometheus inside its namespace alerts on its own now. [#1703](https://github.com/kubermatic/kubermatic/issues/1703) ([metalmatze](https://github.com/metalmatze))
- Add kube-state-metrics to the cluster namespace [#1716](https://github.com/kubermatic/kubermatic/issues/1716) ([mrIncompetent](https://github.com/mrIncompetent))
- Scrape nginx ingress controller metrics [#1777](https://github.com/kubermatic/kubermatic/issues/1777) ([bashofmann](https://github.com/bashofmann))
- use annotations instead of labels for the nginx-ingress Prometheus configuration [#1823](https://github.com/kubermatic/kubermatic/issues/1823) ([xrstf](https://github.com/xrstf))


**Dashboard:**

- Fixed cluster settings view for Openstack [#746](https://github.com/kubermatic/dashboard-v2/issues/746) ([kgroschoff](https://github.com/kgroschoff))
- Fixed error appearing when trying to change selected OS [#699](https://github.com/kubermatic/dashboard-v2/issues/699) ([kgroschoff](https://github.com/kgroschoff))
- Openstack: fixed an issue, where list of tenants wouldn&#39;t get loaded when returning from summary page [#705](https://github.com/kubermatic/dashboard-v2/issues/705) ([kgroschoff](https://github.com/kgroschoff))
- Fixed confirmation of cluster deletion [#718](https://github.com/kubermatic/dashboard-v2/issues/718) ([kgroschoff](https://github.com/kgroschoff))
- Fixed the link to Kubernetes dashboard [#740](https://github.com/kubermatic/dashboard-v2/issues/740) ([guusvw](https://github.com/guusvw))
- vSphere: custom cluster vnet can now be selected [#708](https://github.com/kubermatic/dashboard-v2/issues/708) ([kgroschoff](https://github.com/kgroschoff))
- Openstack: the list of available networks and floating IP pools will be loaded from the API [#737](https://github.com/kubermatic/dashboard-v2/issues/737) ([j3ank](https://github.com/j3ank))
- Dashboard metrics can now be collected by Prometheus [#678](https://github.com/kubermatic/dashboard-v2/issues/678) ([pkavajin](https://github.com/pkavajin))
- Redesigned cluster creation summary page [#688](https://github.com/kubermatic/dashboard-v2/issues/688) ([kgroschoff](https://github.com/kgroschoff))
- Default template images for Openstack and vSphere are now taken from datacenter configuration [#689](https://github.com/kubermatic/dashboard-v2/issues/689) ([kgroschoff](https://github.com/kgroschoff))
- Various minor fixes and improvements




### [v2.6.16]()


- Updated machine-controller to `v0.7.18` [#1709](https://github.com/kubermatic/kubermatic/issues/1709) ([mrIncompetent](https://github.com/mrIncompetent))
- Added support for Kubernetes `1.8.14`, `1.9.8`, `1.9.9`, `1.9.10`, `1.10.4`, `1.10.5` and `1.10.6` [#1710](https://github.com/kubermatic/kubermatic/issues/1710) ([mrIncompetent](https://github.com/mrIncompetent))




### [v2.6.15]()


- Added addon for default StorageClass depending on a cloud provider [#1697](https://github.com/kubermatic/kubermatic/issues/1697) ([glower](https://github.com/glower))




### [v2.6.14]()

**Cloud Provider:**

- Azure: fixed minor issue with seed clusters running on Azure [#1657](https://github.com/kubermatic/kubermatic/issues/1657) ([thz](https://github.com/thz))


**Misc:**

- Updated machine-controller to `v0.7.17` [#1677](https://github.com/kubermatic/kubermatic/issues/1677) ([thz](https://github.com/thz))




### [v2.6.13]()


- Minor fixes for seed clusters running on Azure [#1646](https://github.com/kubermatic/kubermatic/issues/1646) ([mrIncompetent](https://github.com/mrIncompetent))




### [v2.6.11]()


**Cloud Provider:**

- Azure: public IPs will always be allocated on new machines [#1644](https://github.com/kubermatic/kubermatic/issues/1644) ([mrIncompetent](https://github.com/mrIncompetent))


**Misc:**

- Updated nodeport-proxy to v1.2 [#1640](https://github.com/kubermatic/kubermatic/issues/1640) ([mrIncompetent](https://github.com/mrIncompetent))




### [v2.6.10]()


- Updated machine-controller to v0.7.14 [#1635](https://github.com/kubermatic/kubermatic/issues/1635) ([mrIncompetent](https://github.com/mrIncompetent))




### [v2.6.9]()


- controller-manager will now automatically restart on backup config change [#1548](https://github.com/kubermatic/kubermatic/issues/1548) ([bashofmann](https://github.com/bashofmann))
- apiserver will now automatically restart on master-files change [#1552](https://github.com/kubermatic/kubermatic/issues/1552) ([cbeneke](https://github.com/cbeneke))




### [v2.6.8]()


- Minor fixes and improvements




### [v2.6.7]()


**Misc:**

- With the introduction of Kubermatic&#39;s addon manager, the K8S addon manager&#39;s deployments will be automatically cleaned up on old setups [#1513](https://github.com/kubermatic/kubermatic/issues/1513) ([mrIncompetent](https://github.com/mrIncompetent))




### [v2.6.6]()


- AWS: multiple clusters per subnet/VPC are now allowed [#1481](https://github.com/kubermatic/kubermatic/issues/1481) ([mrIncompetent](https://github.com/mrIncompetent))




### [v2.6.5]()


**Bugfix:**

- Fixed a rare issue with duplicate entries on the list of nodes [#1391](https://github.com/kubermatic/kubermatic/issues/1391) ([mrIncompetent](https://github.com/mrIncompetent))
- Fixed deletion of old etcd backups [#1394](https://github.com/kubermatic/kubermatic/issues/1394) ([mrIncompetent](https://github.com/mrIncompetent))


**Cloud Provider:**

- Image templates can now be configured in datacenter.yaml for Openstack and vSphere [#1397](https://github.com/kubermatic/kubermatic/issues/1397) ([mrIncompetent](https://github.com/mrIncompetent))


**Misc:**

- Non-default IP can now be used for cluster DNS [#1393](https://github.com/kubermatic/kubermatic/issues/1393) ([glower](https://github.com/glower))


**Monitoring:**

- Error metrics are now collected for Kubermatic API endpoints [#1376](https://github.com/kubermatic/kubermatic/issues/1376) ([pkavajin](https://github.com/pkavajin))


**Dashboard:**

- Minor visual improvements [#684](https://github.com/kubermatic/dashboard-v2/issues/684) ([kgroschoff](https://github.com/kgroschoff))
- The node list will no longer be expanded when clicking on an IP [#676](https://github.com/kubermatic/dashboard-v2/issues/676) ([kgroschoff](https://github.com/kgroschoff))
- Openstack: the tenant can now be picked from a list loaded from the API [#679](https://github.com/kubermatic/dashboard-v2/issues/679) ([kgroschoff](https://github.com/kgroschoff))
- Added a button to easily duplicate an existing node [#675](https://github.com/kubermatic/dashboard-v2/issues/675) ([kgroschoff](https://github.com/kgroschoff))
- A note has been added to the footer identifying whether the dashboard is a part of a demo system [#682](https://github.com/kubermatic/dashboard-v2/issues/682) ([kgroschoff](https://github.com/kgroschoff))
- Enabled CoreOS on Openstack [#673](https://github.com/kubermatic/dashboard-v2/issues/673) ([kgroschoff](https://github.com/kgroschoff))
- cri-o has been disabled [#670](https://github.com/kubermatic/dashboard-v2/issues/670) ([kgroschoff](https://github.com/kgroschoff))
- Node deletion can now be confirmed by pressing enter [#672](https://github.com/kubermatic/dashboard-v2/issues/672) ([kgroschoff](https://github.com/kgroschoff))




### [v2.6.3]()


**Cloud Provider:**

- Fixed floating IP defaulting on openstack [#1332](https://github.com/kubermatic/kubermatic/issues/1332) ([mrIncompetent](https://github.com/mrIncompetent))
- Azure: added multi-AZ node support [#1354](https://github.com/kubermatic/kubermatic/issues/1354) ([mrIncompetent](https://github.com/mrIncompetent))
- Fixed premature logout from vsphere API [#1373](https://github.com/kubermatic/kubermatic/issues/1373) ([alvaroaleman](https://github.com/alvaroaleman))


**Misc:**

- Control plane can now reach the nodes via VPN [#1234](https://github.com/kubermatic/kubermatic/issues/1234) ([thz](https://github.com/thz))
- Enabled Mutating/Validating Admission Webhooks for K8S 1.9&#43; [#1352](https://github.com/kubermatic/kubermatic/issues/1352) ([alvaroaleman](https://github.com/alvaroaleman))
- Updated addon manager to `v0.1.0` [#1363](https://github.com/kubermatic/kubermatic/issues/1363) ([thz](https://github.com/thz))
- Update machine-controller to `v0.7.5` [#1374](https://github.com/kubermatic/kubermatic/issues/1374) ([mrIncompetent](https://github.com/mrIncompetent))




### [v2.6.2]()


- Minor fixes and improvements for Openstack support




### [v2.6.1]()


**Cloud Provider:**

- Non-ESXi vsphere hosts are now supported [#1306](https://github.com/kubermatic/kubermatic/issues/1306) ([alvaroaleman](https://github.com/alvaroaleman))
- VSphere target folder will be properly cleaned up on cluster deletion. [#1314](https://github.com/kubermatic/kubermatic/issues/1314) ([alvaroaleman](https://github.com/alvaroaleman))


**Misc:**

- Addons in kubermatic charts can now be specified as a list [#1304](https://github.com/kubermatic/kubermatic/issues/1304) ([guusvw](https://github.com/guusvw))
- Updated machine-controller to `v0.7.3` [#1311](https://github.com/kubermatic/kubermatic/issues/1311) ([mrIncompetent](https://github.com/mrIncompetent))


**Monitoring:**

- Fixed metric name for addon controller [#1323](https://github.com/kubermatic/kubermatic/issues/1323) ([alvaroaleman](https://github.com/alvaroaleman))

### [v2.6.0]()


**Bugfix:**

- Cluster IPv6 addresses will be ignored on systems on which they are available [#1017](https://github.com/kubermatic/kubermatic/issues/1017) ([mrIncompetent](https://github.com/mrIncompetent))
- Fixed an issue with duplicate users being sometimes created [#990](https://github.com/kubermatic/kubermatic/issues/990) ([mrIncompetent](https://github.com/mrIncompetent))


**Cloud Provider:**

- Added Azure support [#1200](https://github.com/kubermatic/kubermatic/issues/1200) ([kdomanski](https://github.com/kdomanski))
- Openstack: made cluster resource cleanup idempotent [#961](https://github.com/kubermatic/kubermatic/issues/961) ([mrIncompetent](https://github.com/mrIncompetent))


**Misc:**

- Updated prometheus operator to `v0.19.0` [#1014](https://github.com/kubermatic/kubermatic/issues/1014) ([mrIncompetent](https://github.com/mrIncompetent))
- Updated dex to `v2.10.0` [#1052](https://github.com/kubermatic/kubermatic/issues/1052) ([mrIncompetent](https://github.com/mrIncompetent))
- etcd operator has been replaced with a `StatefulSet` [#1065](https://github.com/kubermatic/kubermatic/issues/1065) ([mrIncompetent](https://github.com/mrIncompetent))
- Nodeport range is now configurable [#1084](https://github.com/kubermatic/kubermatic/issues/1084) ([mrIncompetent](https://github.com/mrIncompetent))
- Bare-metal provider has been removed [#1087](https://github.com/kubermatic/kubermatic/issues/1087) ([mrIncompetent](https://github.com/mrIncompetent))
- Introduced addon manager [#1152](https://github.com/kubermatic/kubermatic/issues/1152) ([mrIncompetent](https://github.com/mrIncompetent))
- etcd data of user clusters can now be automatically backed up [#1170](https://github.com/kubermatic/kubermatic/issues/1170) ([alvaroaleman](https://github.com/alvaroaleman))
- Updated machine-controller to `v0.7.2` [#1227](https://github.com/kubermatic/kubermatic/issues/1227) ([mrIncompetent](https://github.com/mrIncompetent))
- etcd disk size can now be configured [#1301](https://github.com/kubermatic/kubermatic/issues/1301) ([mrIncompetent](https://github.com/mrIncompetent))
- Updated kube-state-metrics to `v1.3.1` [#933](https://github.com/kubermatic/kubermatic/issues/933) ([metalmatze](https://github.com/metalmatze))
- Added the ability to blacklist a cluster from reconciliation by the cluster-controller [#936](https://github.com/kubermatic/kubermatic/issues/936) ([mrIncompetent](https://github.com/mrIncompetent))
- Allow disabling TLS verification in offline environments [#968](https://github.com/kubermatic/kubermatic/issues/968) ([mrIncompetent](https://github.com/mrIncompetent))
- Updated nginx-ingress to `v0.14.0` [#983](https://github.com/kubermatic/kubermatic/issues/983) ([metalmatze](https://github.com/metalmatze))
- Kubernetes can now automatically allocate a nodeport if the default nodeport range is unavailable [#987](https://github.com/kubermatic/kubermatic/issues/987) ([mrIncompetent](https://github.com/mrIncompetent))
- Updated nodeport-proxy to `v1.1` [#988](https://github.com/kubermatic/kubermatic/issues/988) ([mrIncompetent](https://github.com/mrIncompetent))
- Added support for Kubernetes `v1.10.2` [#989](https://github.com/kubermatic/kubermatic/issues/989) ([mrIncompetent](https://github.com/mrIncompetent))
- Various other fixes and improvements


**Monitoring:**

- Added alerts for kubermatic master components being down [#1031](https://github.com/kubermatic/kubermatic/issues/1031) ([metalmatze](https://github.com/metalmatze))
- Massive amount of general improvements to alerting
