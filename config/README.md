# Setting up a master-seed cluster

**Note:** This documentation is a work-in-progress, and still has many points that need to be added.

Some topics that still need to be covered:
- [ ] Bare Metal Configurations and Provider
- [ ] Detailed Installation Guide
- [ ] Auth using Dex
- [ ] Nginx Customization
- [ ] Prometheus, Grafana and AlertManager

## Creating the Seed Cluster Configuration
Installation of Kubermatic uses the [Kubermatic Installer][4], which is essentially a container with [Helm][5] and the required charts to install Kubermatic and it's associated resources.

Customization of the cluster configuration is done using a seed-cluster specific _values.yaml_, stored as an encrypted file in the Kubermatic [secrets repository][6]

For reference you can see the dev clusters [values.yaml][1] file.

### `values.yaml` Keys

- #### `KubermaticURL` _(required)_

  This is the URL to access the dashboard of the cluster. This will also be the subdomain which the client clusters will be assigned under (_i.e. <cluster-address.subdomain.domain.tld>_)

  **The value should be quoted.**

  Example:
  > `KubermaticURL: "dev.kubermatic.io"`

- #### `Kubeconfig` _(required)_

  This is a standard [Kubeconfig][2]. A context does not need to be set, but a context name must match the key for each seed cluster definition in the [datacenters.yaml][3] in the `KubermaticDatacenters` parameter

  When defining multiple seed clusters you must have a valid context for each seed cluster in the `Kubeconfig` with the context names must match to the correct keys in `KubermaticDatacenters`

  **The value should be quoted.**

  > The `Kubeconfig` value must be base64 encoded, without any linebreaks.
    You can encode it using the command `base64 <path to Kubermatic kubeconfig>`

- #### `KubermaticDatacenters` _(required)_

  This is the Datacenter definition for Kubermatic. You can find detailed documentation on the Datacenter definition file [here][3].

  **The value should be quoted.**

  > The `KubermaticDatacenters` value must be base64 encoded, without any linebreaks
  You can encode it using the command `base64 <path to datacenters.yaml>`

- #### `Certificates` Block _(required)_
  Certificates defines the domains to pull certificates for via ACME, typically from [Let's Encrypt][14]. You will need to add the domain from `KubermaticURL`, as well as any additional domains you need certificates for.

  Example:
  ```
  Certificates:
    Domains:
    - "dev.kubermatic.io"
    - "alertmanager.dev.kubermatic.io"
    - "grafana.dev.kubermatic.io"
    - "prometheus.dev.kubermatic.io"
  ```

- #### `Storage` Block _(optional)_

  This defines the default storage for the cluster, creating a [StorageClass][8] with the name `generic` and setting it as the default StorageClass.

  Example:
  ```
  Storage:
  - Provider: "gke"
  - Zone: "europe-west3-c"
  - Type: "pd-ssd"
  ```

  Please see the Storage chart's [_helpers.tpl][7] for specific implementation details

  - ##### `Provider` _(required)_

    This defines the Provider to use for provisioning storage with the storage class.

    Currently supported providers are:
    - `aws`: AWS Elastic Block Storage
    - `gke`: GCE Persistent Disk
    - `openstack-cinder`: OpenStack Cinder
    - `bare-metal`: GlusterFS Heketi

    **The value should be quoted.**

    Example:
    > `- Provider: "gke"`

  - ##### `Zone`

    This defines the zone in which the default StorageClass should create [PersistentVolume][9] resources. This typically is the cloud-providers geographic region.

    _Applicable Providers_:
    - `aws`
    - `gke`
    - `openstack-cinder`

    **The value should be quoted.**

    Example:
    > `Zone: "us-central1-c"`

  - ##### `Type`

    This defines the type of PersistentVolume device the default StorageClass should create. This typically relates to the devices available IOPS and throughput.

    _Applicable Providers_:
    - `aws`
    - `gke`
    - `openstack-cinder`

    **The value should be quoted.**

    Example:
    > `- Type: "pd-ssd"`

  - ##### `URL`

    This defines the type of PersistentVolume device the default StorageClass should create. This typically relates to the devices available IOPS and throughput.

    _Applicable Providers_:
    - `bare-metal`

    **The value should be quoted.**

    Example:
    > `- URL: "http://heketi:8080"`

- #### `NodePortExposer` _(optional)_

  The NodePort Exposer component replaces K8Sniff in newer deployments. Additionally it uses the block-style syntax for it's keys and values.

  The NodePort Exposer only needs to be used in deployments that do not already expose the [NodePort range][15], typically in cloud deployment scenarios. If the NodePort range is already exposed you do not need to deploy the NodePort exposer

  Example:
  ```
  NodePortExposer:
    Image: "kubermatic/nodeport-exposer"
    ImageTag: "v1.0"
    Namespace: "nodeport-exposer"
    LBServiceName: "nodeport-exposer"
  ```

  - ##### `Image`

    This defines the Docker image to use for the NodePortExposer deployment.

    **The value should be quoted.**

    Example:
    > `  Image: "kubermatic/nodeport-exposer"`

  - ##### `ImageTag`

    This defines the tag for the Docker image to use for the NodePortExposer deployment. Possible values can be found on the [Docker Hub repo][13]

    **The value should be quoted.**

    Example:
    > `  ImageTag: "v1.0"`

  - ##### `Namespace`

    This defines the namespace in which to deploy the NodePortExposer resources.

    **The value should be quoted.**

    Example:
    > `  Namespace: "nodeport-exposer"`

  - ##### `LBServiceName`

    This defines the Service name to create for the NodePortExposer.

    **The value should be quoted.**

    Example:
    > `  LBServiceName: "nodeport-exposer"`

- #### Replicas
  These values set the number of replicas of the Kubermatic components running in the cluster.

  **Keys**:
  - `KubermaticAPIReplicaCount`
  - `KubermaticUIReplicaCount`
  - `K8SniffReplicaCount`

  Example:
  ```
  KubermaticAPIReplicaCount: "2"
  KubermaticUIReplicaCount: "1"
  K8SniffReplicaCount: "2"
  ```

- #### Image Tags _(optional)_

  These values set the image tag to use for the Kubermatic installation. You can find the possible tag values for each key at the link provided with the key.

  **Keys**:
  - [KubermaticAPIImageTag][10]
  - [KubermaticControllerImageTag][10]
  - [KubermaticDashboardImageTag][11]
  - [KubermaticBareMetalProviderImageTag][12]
  - [K8sniffImageTag][13]

  Example:
  ```
  KubermaticAPIImageTag: "master"
  KubermaticControllerImageTag: "master"
  KubermaticDashboardImageTag: "master"
  KubermaticBareMetalProviderImageTag: "master"
  K8sniffImageTag: "v1.1"
  ```

There are more options than the above listed. The reference values.yaml should be self-explaining.

### Deploy installer
```bash
kubectl create -f installer/namespace.yaml
kubectl create -f installer/serviceaccount.yaml
kubectl create -f installer/clusterrolebinding.yaml
# values.yaml is the file you created during the step above
kubectl -n kubermatic-installer create secret generic values --from-file=values.yaml
#Create the docker secret - needs to have read access to kubermatic/installer
kubectl  -n kubermatic-installer create secret docker-registry dockercfg --docker-username='' --docker-password='' --docker-email=''
kubectl  -n kubermatic-installer create secret docker-registry quay --docker-username='' --docker-password='' --docker-email=''
# Create and run the installer job
# Replace the version in the installer job template
cp installer/install-job.template.yaml install-job.yaml
sed -i "s/{INSTALLER_TAG}/master/g" install-job.yaml
kubectl create -f install-job.yaml
```

### Create DNS entry for your domain
Go to https://www.ovh.ie/ and add a dns entry for the dashboard:
- $DOMAIN

The external ip for the DNS entry can be fetched via
```bash
kubectl -n ingress-nginx describe service nginx-ingress-controller | grep "LoadBalancer Ingress"
```

Go to https://www.ovh.ie/ and add a dns entry for the nodeport-exposer:
$DATACENTER=us-central1
- *.$DATACENTER.$DOMAIN  =  *.us-central1.dev.kubermatic.io

The external ip for the DNS entry can be fetched via
```bash
kubectl -n nodeport-exposer describe service nodeport-exposer | grep "LoadBalancer Ingress"
```

[1]: https://github.com/kubermatic/secrets/blob/master/seed-clusters/dev.kubermatic.io/values.yaml
[2]: https://kubernetes.io/docs/tasks/access-application-cluster/configure-access-multiple-clusters/
[3]: ../docs/datacenters.md
[4]: installer/README.md
[5]: https://github.com/kubernetes/helm
[6]: https://github.com/kubermatic/secrets
[7]: https://github.com/kubermatic/kubermatic/blob/master/config/storage/templates/_helpers.tpl
[8]: https://kubernetes.io/docs/concepts/storage/storage-classes/
[9]: https://kubernetes.io/docs/concepts/storage/persistent-volumes/
[10]: https://hub.docker.com/r/kubermatic/api/tags/
[11]: https://hub.docker.com/r/kubermatic/ui-v2/tags/
[12]: https://hub.docker.com/r/kubermatic/bare-metal-provider/tags/
[13]: https://hub.docker.com/r/kubermatic/k8sniff-internal/tags/
[13]: https://hub.docker.com/r/kubermatic/nodeport-exposer/tags/
[14]: https://letsencrypt.org/
[15]: https://kubernetes.io/docs/concepts/services-networking/service/#type-nodeport
