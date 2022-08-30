# Kubermatic 2.12

- [v2.12.0](#v2120)
- [v2.12.1](#v2121)
- [v2.12.2](#v2122)
- [v2.12.3](#v2123)
- [v2.12.4](#v2124)
- [v2.12.5](#v2125)
- [v2.12.6](#v2126)
- [v2.12.7](#v2127)
- [v2.12.8](#v2128)
- [v2.12.9](#v2129)

## [v2.12.9](https://github.com/kubermatic/kubermatic/releases/tag/v2.12.9)

- Added Kubernetes v1.16.13, and removed v1.16.2-7 ([#5662](https://github.com/kubermatic/kubermatic/issues/5662))


## [v2.12.8](https://github.com/kubermatic/kubermatic/releases/tag/v2.12.8)

- Updated machine-controller to v1.8.4 to address issue in CNI plugins ([#5442](https://github.com/kubermatic/kubermatic/issues/5442))


## [v2.12.7](https://github.com/kubermatic/kubermatic/releases/tag/v2.12.7)

- Openstack: fixed a bug preventing the usage of pre-existing subnets connected to distributed routers ([#5334](https://github.com/kubermatic/kubermatic/issues/5334))
- Update machine-controller to v1.8.2 to fix the Docker daemon/CLI version incompatibility ([#5426](https://github.com/kubermatic/kubermatic/issues/5426))


## [v2.12.6](https://github.com/kubermatic/kubermatic/releases/tag/v2.12.6)

### Misc

- System labels can no longer be removed by the user. ([#4983](https://github.com/kubermatic/kubermatic/issues/4983))
- End-of-Life Kubernetes v1.14 is no longer supported. ([#4988](https://github.com/kubermatic/kubermatic/issues/4988))
- Added Kubernetes v1.15.7, v1.15.9, v1.16.4, v1.16.6 ([#4995](https://github.com/kubermatic/kubermatic/issues/4995))


## [v2.12.5](https://github.com/kubermatic/kubermatic/releases/tag/v2.12.5)

- A bug that occasionally resulted in a `Error: no matches for kind "MachineDeployment" in version "cluster.k8s.io/v1alpha1"` visible in the UI was fixed. ([#4870](https://github.com/kubermatic/kubermatic/issues/4870))
- A memory leak in the port-forwarding of the Kubernetes dashboard and Openshift console endpoints was fixed ([#4879](https://github.com/kubermatic/kubermatic/issues/4879))
- Enabled edit options for kubeAdm ([#1873](https://github.com/kubermatic/dashboard/issues/1873))


## [v2.12.4](https://github.com/kubermatic/kubermatic/releases/tag/v2.12.4)

- Fixed an issue with adding new node deployments on Openstack ([#1836](https://github.com/kubermatic/dashboard/issues/1836))
- Added migration for cluster user labels ([#4744](https://github.com/kubermatic/kubermatic/issues/4744))
- Added Kubernetes v1.14.9, v1.15.6 and v1.16.3 ([#4752](https://github.com/kubermatic/kubermatic/issues/4752))
- Openstack: A bug that caused cluster reconciliation to fail if the controller crashed at the wrong time was fixed ([#4754](https://github.com/kubermatic/kubermatic/issues/4754))


## [v2.12.3](https://github.com/kubermatic/kubermatic/releases/tag/v2.12.3)

- Fixed extended cluster options not being properly applied ([#1812](https://github.com/kubermatic/dashboard/issues/1812))
- A panic that could occur on clusters that lack both credentials and a credentialsSecret was fixed. ([#4742](https://github.com/kubermatic/kubermatic/issues/4742))


## [v2.12.2](https://github.com/kubermatic/kubermatic/releases/tag/v2.12.2)

- The robustness of vSphere machine reconciliation has been improved. ([#4651](https://github.com/kubermatic/kubermatic/issues/4651))
- Fixed Seed Validation Webhook rejecting new Seeds in certain situations ([#4662](https://github.com/kubermatic/kubermatic/issues/4662))
- Rolled nginx-ingress-controller back to 0.25.1 to fix SSL redirect issues. ([#4693](https://github.com/kubermatic/kubermatic/issues/4693))


## [v2.12.1](https://github.com/kubermatic/kubermatic/releases/tag/v2.12.1)

- VSphere: Fixed a bug that resulted in a faulty cloud config when using a non-default port ([#4562](https://github.com/kubermatic/kubermatic/issues/4562))
- Fixed master-controller failing to create project-label-synchronizer controllers. ([#4577](https://github.com/kubermatic/kubermatic/issues/4577))
- Fixed broken NodePort-Proxy for user clusters with LoadBalancer expose strategy. ([#4590](https://github.com/kubermatic/kubermatic/issues/4590))


## [v2.12.0](https://github.com/kubermatic/kubermatic/releases/tag/v2.12.0)

### Supported Kubernetes versions

- `1.14.8`
- `1.15.5`
- `1.16.2`
- Openshift `v4.1.18` preview

### Major new features

- Kubernetes 1.16 support was added ([#4313](https://github.com/kubermatic/kubermatic/issues/4313))
- It is now possible to also configure automatic node updates by setting `automaticNodeUpdate: true` in the `updates.yaml`. This option implies `automatic: true` as node versions must not be newer than the version of the corresponding controlplane. ([#4258](https://github.com/kubermatic/kubermatic/issues/4258))
- Cloud credentials can now be configured as presets ([#3723](https://github.com/kubermatic/kubermatic/issues/3723))
- Access to datacenters can now be restricted based on the user's email domain. ([#4470](https://github.com/kubermatic/kubermatic/issues/4470))
- It is now possible to open the Kubernetes Dashboard from the Kubermatic UI. ([#4460](https://github.com/kubermatic/kubermatic/issues/4460))
- An option to use AWS Route53 DNS validation was added to the `certs` chart. ([#4397](https://github.com/kubermatic/kubermatic/issues/4397))
- Added possibility to add labels to projects and clusters and have these labels inherited by node objects.
- Added support for Kubernetes audit logging ([#4151](https://github.com/kubermatic/kubermatic/issues/4151))
- Connect button on cluster details will now open Kubernetes Dashboard/Openshift Console ([#1667](https://github.com/kubermatic/dashboard/issues/1667))
- Pod Security Policies can now be enabled ([#4062](https://github.com/kubermatic/kubermatic/issues/4062))
- Added support for optional cluster addons ([#1683](https://github.com/kubermatic/dashboard/issues/1683))

### Installation and updating

- **ACTION REQUIRED:** the `zone_character` field must be removed from all AWS datacenters in `datacenters.yaml` ([#3986](https://github.com/kubermatic/kubermatic/issues/3986))
- **ACTION REQUIRED:** The default number of apiserver replicas was increased to 2. You can revert to the old behavior by setting `.Kubermatic.apiserverDefaultReplicas` in the `values.yaml` ([#3885](https://github.com/kubermatic/kubermatic/issues/3885))
- **ACTION REQUIRED:** The literal credentials on the `Cluster` object are being deprecated in favor of storing them in a secret. If you have addons that use credentials, replace `.Cluster.Spec.Cloud` with `.Credentials`. ([#4463](https://github.com/kubermatic/kubermatic/issues/4463))
- **ACTION REQUIRED:** Kubermatic now doesn't accept unknown keys in its config files anymore and will crash if an unknown key is present
- **ACTION REQUIRED:** BYO datacenters now need to be specific in the `datacenters.yaml` with a value of `{}`, e.G `bringyourown: {}` ([#3794](https://github.com/kubermatic/kubermatic/issues/3794))
- **ACTION REQUIRED:** Velero does not backup Prometheus, Elasticsearch and Minio by default anymore. ([#4482](https://github.com/kubermatic/kubermatic/issues/4482))
- **ACTION REQUIRED:** On AWS, the nodeport-proxy will be recreated as NLB. DNS entries must be updated to point to the new LB. ([#3840](https://github.com/kubermatic/kubermatic/pull/3840))
- The deprecated nodePortPoxy key for Helm values has been removed. ([#3830](https://github.com/kubermatic/kubermatic/issues/3830))
- Support setting oidc authentication settings on cluster ([#3751](https://github.com/kubermatic/kubermatic/issues/3751))
- The worker-count of controller-manager and master-controller are now configurable ([#3918](https://github.com/kubermatic/kubermatic/issues/3918))
- master-controller-manager can now be deployed with multiple replicas ([#4307](https://github.com/kubermatic/kubermatic/issues/4307))
- It is now possible to configure an http proxy on a Seed. This will result in the proxy being used for all control plane pods in that seed that talk to a cloudprovider and for all machines in that Seed, unless its overridden on Datacenter level. ([#4459](https://github.com/kubermatic/kubermatic/issues/4459))
- The cert-manager Helm chart now allows configuring extra values for its controllers args and env vars. ([#4398](https://github.com/kubermatic/kubermatic/issues/4398))
- A fix for CVE-2019-11253 for clusters that were created with a Kubernetes version < 1.14 was deployed ([#4520](https://github.com/kubermatic/kubermatic/issues/4520))

### Dashboard

- Added Swagger UI for Kubermatic API ([#1418](https://github.com/kubermatic/dashboard/issues/1418))
- Redesign dialog to manage SSH keys on cluster ([#1353](https://github.com/kubermatic/dashboard/issues/1353))
- GCP zones are now fetched from API. ([#1379](https://github.com/kubermatic/dashboard/issues/1379))
- Redesign Wizard: Summary ([#1409](https://github.com/kubermatic/dashboard/issues/1409))
- Cluster type toggle in wizard is now hidden if only one cluster type is active ([#1425](https://github.com/kubermatic/dashboard/issues/1425))
- Disabled the possibility of adding new node deployments until the cluster is fully ready. ([#1439](https://github.com/kubermatic/dashboard/issues/1439))
- The cluster name is now editable from the dashboard ([#1455](https://github.com/kubermatic/dashboard/issues/1455))
- Added warning about node deployment changes that will recreate all nodes. ([#1479](https://github.com/kubermatic/dashboard/issues/1479))
- OIDC client id is now configurable ([#1505](https://github.com/kubermatic/dashboard/issues/1505))
- Replaced particles with a static background. ([#1578](https://github.com/kubermatic/dashboard/issues/1578))
- Pod Security Policy can now be activated from the wizard. ([#1647](https://github.com/kubermatic/dashboard/issues/1647))
- Redesigned extended options in wizard ([#1609](https://github.com/kubermatic/dashboard/issues/1609))
- Various security improvements in authentication
- Various other visual improvements

### Monitoring and logging

- Alertmanager's inhibition feature is now used to hide consequential alerts. ([#3833](https://github.com/kubermatic/kubermatic/issues/3833))
- Removed cluster owner name and email labels from kubermatic_cluster_info metric to prevent leaking PII ([#3854](https://github.com/kubermatic/kubermatic/issues/3854))
- New Prometheus metrics kubermatic_addon_created kubermatic_addon_deleted
- New alert KubermaticAddonDeletionTakesTooLong ([#3941](https://github.com/kubermatic/kubermatic/issues/3941))
- FluentBit will now collect the journald logs ([#4001](https://github.com/kubermatic/kubermatic/issues/4001))
- FluentBit can now collect the kernel messages ([#4007](https://github.com/kubermatic/kubermatic/issues/4007))
- FluentBit now always sets the node name in logs ([#4010](https://github.com/kubermatic/kubermatic/issues/4010))
- Added new KubermaticClusterPaused alert with "none" severity for inhibiting alerts from paused clusters ([#3846](https://github.com/kubermatic/kubermatic/issues/3846))
- Removed Helm-based templating in Grafana dashboards ([#4475](https://github.com/kubermatic/kubermatic/issues/4475))
- Added type label (kubernetes/openshift) to kubermatic_cluster_info metric. ([#4452](https://github.com/kubermatic/kubermatic/issues/4452))
- Added metrics endpoint for cluster control plane: `GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/metrics` ([#4208](https://github.com/kubermatic/kubermatic/issues/4208))
- Added a new endpoint for node deployment metrics: `GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodedeployments/{nodedeployment_id}/metrics` ([#4176](https://github.com/kubermatic/kubermatic/issues/4176))

### Cloud providers

- Openstack: A bug that could result in many security groups being created when the creation of security group rules failed was fixed ([#3848](https://github.com/kubermatic/kubermatic/issues/3848))
- Openstack: Fixed a bug preventing an interrupted cluster creation from being resumed. ([#4476](https://github.com/kubermatic/kubermatic/issues/4476))
- Openstack: Disk size of nodes is now configurable ([#4153](https://github.com/kubermatic/kubermatic/issues/4153))
- Openstack: Added a security group API compatibility workaround for very old versions of Openstack. ([#4479](https://github.com/kubermatic/kubermatic/issues/4479))
- Openstack: Fixed fetching the list of tenants on some OpenStack configurations with one region ([#4182](https://github.com/kubermatic/kubermatic/issues/4182))
- Openstack: Added support for Project ID to the wizard ([#1386](https://github.com/kubermatic/dashboard/issues/1386))
- Openstack: The project name can now be provided manually ([#1423](https://github.com/kubermatic/dashboard/issues/1423))
- Openstack: Fixed API usage for datacenters with only one region ([#4538](https://github.com/kubermatic/kubermatic/issues/4538))
- Openstack: Fixed a bug that resulted in the router not being attached to the subnet when the subnet was manually created ([#4521](https://github.com/kubermatic/kubermatic/issues/4521))
- AWS: MachineDeployments can now be created in any availability zone of the cluster's region ([#3870](https://github.com/kubermatic/kubermatic/issues/3870))
- AWS: Reduced the role permissions for the control-plane & worker role to the minimum ([#3995](https://github.com/kubermatic/kubermatic/issues/3995))
- AWS: The subnet can now be selected ([#1499](https://github.com/kubermatic/dashboard/issues/1499))
- AWS: Setting `Control plane role (ARN)` now is possible ([#1512](https://github.com/kubermatic/dashboard/issues/1512))
- AWS: VM sizes are fetched from the API now. ([#1513](https://github.com/kubermatic/dashboard/issues/1513))
- AWS: Worker nodes can now be provisioned without a public IP ([#1591](https://github.com/kubermatic/dashboard/issues/1591))
- GCP: machine and disk types are now fetched from GCP.([#1363](https://github.com/kubermatic/dashboard/issues/1363))
- vSphere: the VM folder can now be configured
- Added support for KubeVirt provider ([#1608](https://github.com/kubermatic/dashboard/issues/1608))

### Bugfixes

- A bug that sometimes resulted in the creation of the initial NodeDeployment failing was fixed ([#3894](https://github.com/kubermatic/kubermatic/issues/3894))
- `kubeadm join` has been fixed for v1.15 clusters ([#4161](https://github.com/kubermatic/kubermatic/issues/4161))
- Fixed a bug that could cause intermittent delays when using kubectl logs/exec with `exposeStrategy: LoadBalancer` ([#4278](https://github.com/kubermatic/kubermatic/issues/4278))
- A bug that prevented node Labels, Taints and Annotations from getting applied correctly was fixed. ([#4368](https://github.com/kubermatic/kubermatic/issues/4368))
- Fixed worker nodes provisioning for instances with a Kernel >= 4.19 ([#4178](https://github.com/kubermatic/kubermatic/issues/4178))
- Fixed an issue that kept clusters stuck if their creation didn't succeed and they got deleted with LB and/or PV cleanup enabled ([#3973](https://github.com/kubermatic/kubermatic/issues/3973))
- Fixed an issue where deleted project owners would come back after a while ([#4025](https://github.com/kubermatic/kubermatic/issues/4025))
- Enabling the OIDC feature flag in clusters has been fixed. ([#4127](https://github.com/kubermatic/kubermatic/issues/4127))

### Misc

- The share cluster feature now allows to use groups, if passed by the IDP. All groups are prefixed with `oidc:` ([#4244](https://github.com/kubermatic/kubermatic/issues/4244))
- The kube-proxy mode (ipvs/iptables) can now be configured. If not specified, it defaults to ipvs. ([#4247](https://github.com/kubermatic/kubermatic/issues/4247))
- Addons can now read the AWS region  from the `kubermatic.io/aws-region` annotation on the cluster ([#4434](https://github.com/kubermatic/kubermatic/issues/4434))
- Allow disabling of apiserver endpoint reconciling. ([#4396](https://github.com/kubermatic/kubermatic/issues/4396))
- Allow cluster owner to manage RBACs from Kubermatic API ([#4321](https://github.com/kubermatic/kubermatic/issues/4321))
- The default service CIDR for new clusters was increased and changed from 10.10.10.0/24 to 10.240.16.0/20 ([#4227](https://github.com/kubermatic/kubermatic/issues/4227))
- Retries of the initial node deployment creation do not create an event anymore but continue to be logged at debug level. ([#4226](https://github.com/kubermatic/kubermatic/issues/4226))
- Added option to enforce cluster cleanup in UI ([#3966](https://github.com/kubermatic/kubermatic/issues/3966))
- Support PodSecurityPolicies in addons ([#4174](https://github.com/kubermatic/kubermatic/issues/4174))
- Kubernetes versions affected by CVE-2019-9512 and CVE-2019-9514 have been dropped ([#4113](https://github.com/kubermatic/kubermatic/issues/4113))
- Kubernetes versions affected by CVE-2019-11247 and CVE-2019-11249 have been dropped ([#4066](https://github.com/kubermatic/kubermatic/issues/4066))
- Kubernetes 1.13 which is end-of-life has been removed. ([#4327](https://github.com/kubermatic/kubermatic/issues/4327))
- Updated Alertmanager to 0.19 ([#4340](https://github.com/kubermatic/kubermatic/issues/4340))
- Updated blackbox-exporter to 0.15.1 ([#4341](https://github.com/kubermatic/kubermatic/issues/4341))
- Updated Canal to v3.8 ([#3791](https://github.com/kubermatic/kubermatic/issues/3791))
- Updated cert-manager to 0.10.1 ([#4407](https://github.com/kubermatic/kubermatic/issues/4407))
- Updated Dex to 2.19 ([#4343](https://github.com/kubermatic/kubermatic/issues/4343))
- Updated Envoy to 1.11.1 ([#4075](https://github.com/kubermatic/kubermatic/issues/4075))
- Updated etcd to 3.3.15 ([#4199](https://github.com/kubermatic/kubermatic/issues/4199))
- Updated FluentBit to v1.2.2 ([#4022](https://github.com/kubermatic/kubermatic/issues/4022))
- Updated Grafana to 6.3.5 ([#4342](https://github.com/kubermatic/kubermatic/issues/4342))
- Updated helm-exporter to 0.4.2 ([#4124](https://github.com/kubermatic/kubermatic/issues/4124))
- Updated kube-state-metrics to 1.7.2 ([#4129](https://github.com/kubermatic/kubermatic/issues/4129))
- Updated Minio to 2019-09-18T21-55-05Z ([#4339](https://github.com/kubermatic/kubermatic/issues/4339))
- Updated machine-controller to v1.5.6 ([#4310](https://github.com/kubermatic/kubermatic/issues/4310))
- Updated nginx-ingress-controller to 0.26.1 ([#4400](https://github.com/kubermatic/kubermatic/issues/4400))
- Updated Prometheus to 2.12.0 ([#4131](https://github.com/kubermatic/kubermatic/issues/4131))
- Updated Velero to v1.1.0 ([#4468](https://github.com/kubermatic/kubermatic/issues/4468))
