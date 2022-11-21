# Kubermatic 2.17

- [v2.17.0](#v2170)
- [v2.17.1](#v2171)
- [v2.17.2](#v2172)
- [v2.17.3](#v2173)
- [v2.17.4](#v2174)
- [v2.17.5](#v2175)

## [v2.17.5](https://github.com/kubermatic/kubermatic/releases/tag/v2.17.5)

### Bugfixes

- Fix a bug where `$$` in the environment-variables for machine-controller was interpreted in the Kubernetes Manifest and caused machine-controller to be unable to deploy resources, when for e.g. the password contains two consecutive `$` signs ([#7984](https://github.com/kubermatic/kubermatic/issues/7984))
- Fix for Seed API PATCH endpoint which sometimes removed Seed fields unrelated to the PATCH. Fixes the issue where Seed API was using seed clients to update the Seeds on master cluster instead of using the master client. This was causing Seed API not to work on Seeds which were not also the master clusters ([#7925](https://github.com/kubermatic/kubermatic/issues/7925))
- Fix setting of nodeport-proxy resource requests/limits, relax default nodeport-proxy envoy limits ([#8169](https://github.com/kubermatic/kubermatic/issues/8169))


## [v2.17.4](https://github.com/kubermatic/kubermatic/releases/tag/v2.17.4)

### Security

Two vulnerabilities were identified in Kubernetes ([CVE-2021-25741](https://github.com/kubernetes/kubernetes/issues/104980) and [CVE-2020-8561](https://github.com/kubernetes/kubernetes/issues/104720)) of which one (CVE-2021-25741) was fixed in Kubernetes 1.19.15 / 1.20.11 / 1.21.5. CVE-2020-8561 is mitigated by Kubermatic not allowing users to reconfigure the kube-apiserver.

Because of these updates, this KKP release includes automatic update rules for all 1.19/1.20/1.21 clusters older than these patch releases. This release also removes all affected Kubernetes versions from the list of supported versions. While CVE-2020-8561 affects the controlplane, CVE-2021-25741 affects the kubelets, which means that updating the controlplane is not enough. Once the automated controlplane updates have completed, an administrator must manually patch all vulnerable `MachineDeployment`s in all affected userclusters.

To lower the resource consumption on the seed clusters during the reconciliation / node rotation, it's recommended to adjust the `spec.seedControllerManager.maximumParallelReconciles` option in the `KubermaticConfiguration` to restrict the number of parallel updates. Users of the legacy `kubermatic` Helm chart need to update `kubermatic.maxParallelReconcile` in their `values.yaml` to achieve the same effect.

The automatic update rules can, if needed, be overwritten using the `spec.versions.kubernetes.updates` field in the `KubermaticConfiguration` or updating the `updates.yaml` if using the legacy `kubermatic` Helm chart. See [#7825](https://github.com/kubermatic/kubermatic/issues/7824) for how the versions and updates are configured. It is however not recommended to deviate from the default and leave userclusters vulnerable.

### Misc

- Add support of Kubernetes 1.20 and 1.21 in cluster-autoscaler addon ([#7511](https://github.com/kubermatic/kubermatic/issues/7511))
- Remove Gatekeeper from the default accessible addon list ([#7533](https://github.com/kubermatic/kubermatic/issues/7533))
- Fix dashboard source in the Prometheus Exporter dashboard ([#7640](https://github.com/kubermatic/kubermatic/issues/7640))


## [v2.17.3](https://github.com/kubermatic/kubermatic/releases/tag/v2.17.3)

### Bugfixes

- Prometheus and Promtail resources are not mistakenly deleted from userclusters anymore ([#6881](https://github.com/kubermatic/kubermatic/issues/6881))
- Paused userclusters do not reconcile in-cluster resources via the usercluster-controller-manager anymore ([#7470](https://github.com/kubermatic/kubermatic/issues/7470))

### Misc

- Redesign Openstack provider settings step to better support different types of credentials ([#3531](https://github.com/kubermatic/dashboard/issues/3531))
- Changes to the tolerations on the node-local-dns DaemonSet will now be kept instead of being overwritten ([#7466](https://github.com/kubermatic/kubermatic/issues/7466))


## [v2.17.2](https://github.com/kubermatic/kubermatic/releases/tag/v2.17.2)

### Bugfixes

- Kubermatic API, etcd-launcher, and dnat-controller images are defaulted to the docker.io registry only if the provided custom image has less than 3 parts ([#7287](https://github.com/kubermatic/kubermatic/issues/7287))
- Fix a bug that always applies default values to container resources ([#7302](https://github.com/kubermatic/kubermatic/issues/7302))
- Add `ClusterFeatureCCMClusterName` feature for OpenStack clusters. This feature adds the `--cluster-name` flag to the OpenStack external CCM deployment. The feature gate is enabled by default for newly created clusters. Enabling this feature gate for existing clusters will cause the external CCM to lose the track of the existing cloud resources (such as Load Balancers), so it's up to the users to manually clean up any leftover resources ([#7330](https://github.com/kubermatic/kubermatic/issues/7330))
- Explicitly set the namespace for Dex pods in the oauth chart. This fixes the problem with KKP installation failing on Kubernetes 1.21 clusters ([#7348](https://github.com/kubermatic/kubermatic/issues/7348))

### Misc

- allow service account to create projects when belongs to projectmanagers group ([#7043](https://github.com/kubermatic/kubermatic/issues/7043))
- Added option to set the Load Balancer SKU when creating Azure clusters ([#7208](https://github.com/kubermatic/kubermatic/issues/7208))
- add application credentials and OIDC token for OpenStack ([#7221](https://github.com/kubermatic/kubermatic/issues/7221))
- Add `projectmanagers` group for RBAC controller. The new group will be assigned to service accounts ([#7263](https://github.com/kubermatic/kubermatic/issues/7263))
- Allow configuring remote_write in Prometheus Helm chart ([#7288](https://github.com/kubermatic/kubermatic/issues/7288))
- Support standard load balancers for azure in KKP ([#7308](https://github.com/kubermatic/kubermatic/issues/7308))
- Upgrade machine-controller to v1.27.11 ([#7347](https://github.com/kubermatic/kubermatic/issues/7347))

### Dashboard

- Add project managers group as an option for service accounts ([#3375](https://github.com/kubermatic/dashboard/issues/3375))
- Added option to select Azure LoadBalancer SKU in cluster creation ([#3463](https://github.com/kubermatic/dashboard/issues/3463))
- Added support for Application Credentials to the Openstack provider ([#3489](https://github.com/kubermatic/dashboard/issues/3489))


## [v2.17.1](https://github.com/kubermatic/kubermatic/releases/tag/v2.17.1)

### Security

- Upgrade machine-controller to v1.27.8 to address [runC vulnerability CVE-2021-30465](https://github.com/opencontainers/runc/security/advisories/GHSA-c3xm-pvg7-gh7r) ([#7209](https://github.com/kubermatic/kubermatic/pull/7166))

### Bugfixes

- Fixed using a custom CA Bundle for Openstack by authenticating after setting the proper CA bundle ([#7192](https://github.com/kubermatic/kubermatic/issues/7192))
- Fix user ssh key agent migration ([#7193](https://github.com/kubermatic/kubermatic/issues/7193))
- Fix issue where Kubermatic non-admin users were not allowed to manage Kubermatic Constraints ([#6942](https://github.com/kubermatic/kubermatic/issues/6942))
- Fix KKP vSphere client not using the provided custom CA bundle ([#6973](https://github.com/kubermatic/kubermatic/issues/6973))
- Use optimistic lock when adding finalizers to prevent lost updates, and avoiding resource leaks ([#7153](https://github.com/kubermatic/kubermatic/issues/6759))

### Misc

- Use the systemd cgroup driver for newly-created Kubernetes 1.19+ clusters using the kubeadm provider. Since the kubelet-configmap addon is not reconciled, this change will not affect existing clusters, only newly-created clusters ([#7065](https://github.com/kubermatic/kubermatic/issues/7065))
- Re-enable NodeLocal DNS Cache in user clusters ([#7075](https://github.com/kubermatic/kubermatic/issues/7075))
- Open NodePort range in openstack ([#7131](https://github.com/kubermatic/kubermatic/issues/7131))
- Upgrade machine controller to v1.27.8 ([#7209](https://github.com/kubermatic/kubermatic/issues/7209))


## [v2.17.0](https://github.com/kubermatic/kubermatic/releases/tag/v2.17.0)

### Supported Kubernetes Versions

* 1.18.6
* 1.18.8
* 1.18.10
* 1.18.14
* 1.18.17
* 1.19.0
* 1.19.2
* 1.19.3
* 1.19.8
* 1.19.9
* 1.20.2
* 1.20.5
* 1.21.0

### Highlights

- Add support for Kubernetes 1.21 ([#6778](https://github.com/kubermatic/kubermatic/issues/6778))
- [ACTION REQUIRED] Overhaul CA handling, allow to configure a global CA bundle for every component. The OIDC CA file has been removed, manual updates can be necessary. ([#6538](https://github.com/kubermatic/kubermatic/issues/6538))

### Breaking Changes

- Update cert-manager to 1.2.0 ([#6739](https://github.com/kubermatic/kubermatic/issues/6739))
- Helm Chart installations are not supported any longer at KKP 2.17, hence KKP 2.16 chart-based installations have to be imperatively migrated.

### Misc

- New etcd backup and restore controllers ([#5668](https://github.com/kubermatic/kubermatic/issues/5668))
- Add `kubermatic-seed` stack target to the Installer ([#6435](https://github.com/kubermatic/kubermatic/issues/6435))
- Add an endpoint to list Vsphere datastores: GET /api/v2/providers/vsphere/datastores ([#6442](https://github.com/kubermatic/kubermatic/issues/6442))
- KKP Installer will update Helm releases automatically if the values have changed (no need for `--force` in most cases). ([#6449](https://github.com/kubermatic/kubermatic/issues/6449))
- Introduce resource quota for Alibaba provider ([#6458](https://github.com/kubermatic/kubermatic/issues/6458))
- Add Gatekeeper health to cluster health status ([#6461](https://github.com/kubermatic/kubermatic/issues/6461))
- Remove CoreOS Support ([#6465](https://github.com/kubermatic/kubermatic/issues/6465))
- Relax memory limit for openvpn container ([#6467](https://github.com/kubermatic/kubermatic/issues/6467))
- Enable VM resource quota for AWS provider ([#6468](https://github.com/kubermatic/kubermatic/issues/6468))
- Multus has been added as an addon.  ([#6477](https://github.com/kubermatic/kubermatic/issues/6477))
- CoreDNS version now based on Kubernetes version ([#6501](https://github.com/kubermatic/kubermatic/issues/6501))
- Add support for "use-octavia" setting in Openstack provider specs. It defaults to "true" but leaves the possibility to set it to "false" if your provider doesn't support Octavia yet but Neutron LBaaSv2
  ([#6529](https://github.com/kubermatic/kubermatic/issues/6529))
- Add components override field to set nodeportrange for apiserver ([#6533](https://github.com/kubermatic/kubermatic/issues/6533))
- OpenShift support is removed.  ([#6539](https://github.com/kubermatic/kubermatic/issues/6539))
- OpenStack: Add support for "use-octavia" setting in Cluster Openstack cloud specs ([#6565](https://github.com/kubermatic/kubermatic/issues/6565))
- Add support for Hetzner CCM ([#6588](https://github.com/kubermatic/kubermatic/issues/6588))
- Change default gatekeeper webhook timeout to 3 sec, and added option in cluster settings to configure it. ([#6709](https://github.com/kubermatic/kubermatic/issues/6709))
- Add support in Openstack datacenters to explicitly enable certain flavor types. ([#6612](https://github.com/kubermatic/kubermatic/issues/6612))
- Provide the possibility of configuring leader election parameters for user cluster components. ([#6641](https://github.com/kubermatic/kubermatic/issues/6641))
- Remove unused deprecated `certs` chart ([#6656](https://github.com/kubermatic/kubermatic/issues/6656))
- Add `registry_mirrors` to Seed node settings ([#6667](https://github.com/kubermatic/kubermatic/issues/6667))
- Upgrad Gatekeeper from 3.1.0-beta-9 to 3.1.3. NOTICE: this change also moves the Gatekeeper deployment from the Seed to the User clusters. This means that the user clusters will need some additional resources to run the Gatekeeper Pods. Admins please refer to the upgrade guidelines in the documentation. ([#6706](https://github.com/kubermatic/kubermatic/issues/6706))
- Add spot instances as an option for the aws machines in the API  ([#6726](https://github.com/kubermatic/kubermatic/issues/6726))
- Add Multus-CNI to accessible addons. ([#6731](https://github.com/kubermatic/kubermatic/issues/6731))
- Allow to disable the s3-credentials Secret in the Minio chart ([#6760](https://github.com/kubermatic/kubermatic/issues/6760))
- Add `enable` and `enforce` OPA options to Admin Settings ([#6787](https://github.com/kubermatic/kubermatic/issues/6787))
- Installer does not listen on port 8080 anymore ([#6788](https://github.com/kubermatic/kubermatic/issues/6788))
- Node-local-dns is now using UDP for external queries ([#6796](https://github.com/kubermatic/kubermatic/issues/6796))
- Add validation for Kubermatic Constraint Template API. ([#6841](https://github.com/kubermatic/kubermatic/issues/6841))
- Fetch the provisioning cloud-init over the api-server  ([#6843](https://github.com/kubermatic/kubermatic/issues/6843))
- Add `FELIX_IGNORELOOSERPF=true` to `calico-node` container env to allow running on nodes with `net.ipv4.conf.*.rp_filter = 2` set. ([#6865](https://github.com/kubermatic/kubermatic/issues/6865))
- Hetzner AMD Cloud Server (CPX) now selectable when creating a user cluster ([#6872](https://github.com/kubermatic/kubermatic/issues/6872))
- Add GPU support for Azure provider ([#6605](https://github.com/kubermatic/kubermatic/issues/6605))

### Bugfixes

- Fix kube-system/coredns PodDisruptionBudget matchLabels in user clusters ([#6398](https://github.com/kubermatic/kubermatic/issues/6398))
- Fix S3 storage uploader CA bundle option flag ([#6732](https://github.com/kubermatic/kubermatic/issues/6732))
- Fix cases where GET and LIST endpoints for Kubermatic Constraints failed or didn't return all results because there were no related synced Gatekeeper Constraints on the user cluster by just taking the Status from the Gatekeeper Constraints and setting the Synced status to false if the Gatekeeper Constraint is missing. ([#6800])(https://github.com/kubermatic/kubermatic/issues/6800)
- Fix KAS service port in Tunneling agent configuration. ([#6569](https://github.com/kubermatic/kubermatic/issues/6569))
- Fix a bug in OPA-integration where deleting a Constraint Template in the seed cluster, when the user cluster Constraint Template is already deleted caused the deletion to get stuck.Fixed a bug in OPA-integration where creating a cluster with OPA-integration enabled didn't trigger the Constraint Template reconcile loop. ([#6580])(https://github.com/kubermatic/kubermatic/issues/6580)
- Fix issue with gatekeeper not recognizing the AdmissionReview v1 version by changing the webhook to use v1beta1 ([#6550](https://github.com/kubermatic/kubermatic/issues/6550))
- Fix a bug with kubermatic constraints delete getting stuck when corresponding user cluster constraint is missing ([#6598](https://github.com/kubermatic/kubermatic/issues/6598))
- Fix CE installer binary in EE downloads ([#6673](https://github.com/kubermatic/kubermatic/issues/6673))
- Fix nodeport-proxy role used with LoadBalancer expose strategy. ([#6646](https://github.com/kubermatic/kubermatic/issues/6646))
- Fix the operator failing to reconcile the ValidatingWebhookConfiguration object for the cluster validation webhook ([#6639](https://github.com/kubermatic/kubermatic/issues/6639))
- Fix installer trying an invalid certificate to test cert-manager ([#6761](https://github.com/kubermatic/kubermatic/issues/6761))

### Updates

- controller-runtime 0.8.1 ([#6450](https://github.com/kubermatic/kubermatic/issues/6450))
- CSI drivers ([#6594](https://github.com/kubermatic/kubermatic/issues/6594))
- Hetzner CSI, move to `csi` addon ([#6615](https://github.com/kubermatic/kubermatic/issues/6615))
- Prometheus to 0.25.0 ([#6647](https://github.com/kubermatic/kubermatic/issues/6647))
- Dex to 2.27.0 ([#6648](https://github.com/kubermatic/kubermatic/issues/6648))
- Minio to RELEASE.2021-03-04T00-53-13Z ([#6649](https://github.com/kubermatic/kubermatic/issues/6649))
- Loki to 2.1, use boltdb-shipper starting June 1st ([#6650](https://github.com/kubermatic/kubermatic/issues/6650))
- nginx-ingress-controller to 0.44.0 ([#6651](https://github.com/kubermatic/kubermatic/issues/6651))
- blackbox-exporter to 0.18 ([#6652](https://github.com/kubermatic/kubermatic/issues/6652))
- node-exporter to 1.1.2 ([#6653](https://github.com/kubermatic/kubermatic/issues/6653))
- Karma to 0.80 ([#6654](https://github.com/kubermatic/kubermatic/issues/6654))
- Grafana to 7.4.3 ([#6655](https://github.com/kubermatic/kubermatic/issues/6655))
- oauth2-proxy to 7.0.1 ([#6657](https://github.com/kubermatic/kubermatic/issues/6657))
- Go 1.16.1 ([#6684](https://github.com/kubermatic/kubermatic/issues/6684))
- machine-controller to v1.27.1 ([#6695](https://github.com/kubermatic/kubermatic/issues/6695))
- OpenVPN image to version v2.5.2-r0. ([#6697](https://github.com/kubermatic/kubermatic/issues/6697))
- Velero to 1.5.3. ([#6701](https://github.com/kubermatic/kubermatic/issues/6701))

### Dashboard

- Add resource quota settings to the admin panel. ([#3019](https://github.com/kubermatic/dashboard/issues/3019))
- Add autocompletions for vSphere datastores. ([#3020](https://github.com/kubermatic/dashboard/issues/3020))
- Add option to disable User SSH Key Agent from the cluster wizard. ([#3025](https://github.com/kubermatic/dashboard/issues/3025))
- Remove CoreOS ([#3027](https://github.com/kubermatic/dashboard/issues/3027))
- AWS node sizes in the wizard now provide GPU information. ([#3038](https://github.com/kubermatic/dashboard/issues/3038))
- Filter external openstack networks during cluster creation ([#3053](https://github.com/kubermatic/dashboard/issues/3053))
- Add changelog support ([#3081](https://github.com/kubermatic/dashboard/issues/3081))
- Remove OpenShift support. ([#3100](https://github.com/kubermatic/dashboard/issues/3100))
- Redesign add/edit member dialog ([#3104](https://github.com/kubermatic/dashboard/issues/3104))
- Add GPU count display for Alibaba instance types. ([#3113](https://github.com/kubermatic/dashboard/issues/3113))
- Remove duplicated KubeAdm hints from cluster page. ([#3114](https://github.com/kubermatic/dashboard/issues/3114))
- Redesign manage SSH keys dialog on cluster details to improve user experience. ([#3120](https://github.com/kubermatic/dashboard/issues/3120))
- Change VSPhere's diskSizeGB option from optional to required. ([#3121](https://github.com/kubermatic/dashboard/issues/3121))
- Redesign autocomplete inputs. Right now spinner will be displayed next to the input that loads autocompletions in the background. ([#3122](https://github.com/kubermatic/dashboard/issues/3122))
- Add info about GPU count for Azure instances. ([#3140](https://github.com/kubermatic/dashboard/issues/3140))
- Allow custom links to be placed in the help and support panel ([#3141](https://github.com/kubermatic/dashboard/issues/3141))
- Add support for OPA to UI ([#3147](https://github.com/kubermatic/dashboard/issues/3147))
- Add network to Hetzner ([#3158](https://github.com/kubermatic/dashboard/issues/3158))
- Add `enable` and `enforce` OPA options to Admin Settings ([#3206](https://github.com/kubermatic/dashboard/issues/3206))

### Bugfixes

- Fix bug with changing the theme based on the color scheme if enforced_theme was set. ([#3163](https://github.com/kubermatic/dashboard/issues/3163))
