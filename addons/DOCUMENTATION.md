# Addons

Addons are specific services and tools extending functionality of kubernetes. In `kubermatic` we have a set of default addons installed on each user-cluster. The default addons are:

* [Canal](https://github.com/projectcalico/canal): policy based networking for cloud native applications
* [Dashboard](https://github.com/kubernetes/dashboard): General-purpose web UI for kubernetes clusters
* [DNS](https://github.com/kubernetes/dns): kubernetes DNS service
* [heapster](https://github.com/kubernetes/heapster): Compute Resource Usage Analysis and Monitoring of Container Clusters
* [kube-proxy](https://kubernetes.io/docs/reference/command-line-tools-reference/kube-proxy/): kubernetes network proxy
* [rbac](https://kubernetes.io/docs/reference/access-authn-authz/rbac/): kubernetes Role-Based Access Control
* OpenVPN client

## How it works

### Configuration

The configuration of `kubermatic-controller-manager` and `kubermatic-api` is done with `helm` and is stored in `kubermatic-installer/charts/kubermatic/templates/`. `kubermatic-api` controls which addons should be installed. `kubermatic-controller-manager` controls where to get the manifests for the addons and the installation process of the addons.

Configurations for all default addons are stored in `kubermatic` repository, in the `kubermatic/addon` folder. Each addon is represented by manifest files in a sub-folder. All addons will be build into a docker container which the `kubermatic-controller-manager` uses to install addons. The docker image should be freely accessible to let customers extend & modify this image for their own purpose. `kubermatic-controller-manager` will read all addon manifests from a specified folder. The default folder is `/opt/addons` and it should contain sub-folders for each addon. This folder is created during the deployment of `kubermatic-controller-manager` and is specified in `kubermatic-controller-manager-dep.yaml` in `kubermatic-install` repository.


### Install and run addons

`kubermatic-api` component will add all default addons (`canal,dashboard,dns,heapster,kube-proxy,openvpn,rbac`) to the user cluster. You can override the default plugins with the command line parameter `kubermatic-api -adons="canal,dns,heapster,kube-proxy,openvpn,rbac"` if you don't want to install `dashboard` addon. Or you can change a list of addons in `.Values.kubermatic.addons.defaultAddons` in the kubermatic `values.yaml` file before the installation.

## Template variables

Following variables can be used in all addon manifests:
* `{{first .Cluster.Spec.ClusterNetwork.Pods.CIDRBlocks}}` - will render a CIDR IP of the cluster
* `{{default "k8s.gcr.io/" .OverwriteRegistry}}` - will give you a path to the alternative docker image registry. You can set this path with `kubermatic-controller-manager -overwrite-registry="..."` You can set this parameter in the helm chart for `kubermatic-controller-manager`
* `{{.DNSClusterIP}}` - will render IP address of the dns server
