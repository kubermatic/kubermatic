# Kubermatic

## Setting up a master-seed cluster

### Creating the Seed Cluster Configuration
Installation of Kubermatic uses the [Kubermatic Installer][4], which is essentially a container with [Helm][5] and the required charts to install Kubermatic and it's associated resources.

Customization of the cluster configuration is done using a seed-cluster specific _values.yaml_, stored as an encrypted file in the Kubermatic [secrets repository][6]

For reference you can see the dev clusters [values.yaml][1] file.

#### `values.yaml` Keys

- ##### `KubermaticURL` _(required)_

  This is the URL to access the dashboard of the cluster. This will also be the subdomain which the client clusters will be assigned under (_i.e. <cluster-address.subdomain.domain.tld>_)

  **The value should be quoted.**

  Example:
  > `KubermaticURL: "cloud.kubermatic.io"`

- ##### `Kubeconfig` _(required)_

  This is a standard [Kubeconfig][2]. The correct context for the cluster does not need to be set, but the context name must match the key for the seed cluster definition in the `KubermaticDatacenters`

  When defining multiple seed clusters you must have a valid context for each seed cluster in the `Kubeconfig` with the context names must match to the correct keys in `KubermaticDatacenters`

  > The `Kubeconfig` value must be base64 encoded, without any linebreaks

- ##### `KubermaticDatacenters` _(required)_

  This is the Datacenter definition for Kubermatic. You can find detailed documentation on the Datacenter definition file [here][3].

  > The `KubermaticDatacenters` value must be base64 encoded, without any linebreaks

- ##### `Storage` Block _(optional)_

  This defines the default storage for the cluster, creating a [StorageClass][8] with the name `generic` and setting it as the default StorageClass.

  Please see the Storage chart's [_helpers.tpl][7] for specific implementation details

  - ###### `Provider` _(required)_

    This defines the Provider to use for provisioning storage with the storage class.

    Currently supported providers are:
    - `aws`: AWS Elastic Block Storage
    - `gke`: GCE Persistent Disk
    - `openstack-cinder`: OpenStack Cinder
    - `bare-metal`: GlusterFS Heketi

    **The value should be quoted.**

    Example:
    > `- Provider: "gke"`

  - ###### `Zone`

    This defines the zone in which the default StorageClass should create [PersistentVolume][9] resources. This typically is the cloud-providers geographic region.

    _Applicable Providers_:
    - `aws`
    - `gke`
    - `openstack-cinder`

    **The value should be quoted.**

    Example:
    > `Zone: "us-central1-c"`

  - ###### `Type`

    This defines the type of PersistentVolume device the default StorageClass should create. This typically relates to the devices available IOPS and throughput.

    _Applicable Providers_:
    - `aws`
    - `gke`
    - `openstack-cinder`

    **The value should be quoted.**

    Example:
    > `- Type: "pd-ssd"`

  - ###### `URL`

    This defines the type of PersistentVolume device the default StorageClass should create. This typically relates to the devices available IOPS and throughput.

    _Applicable Providers_:
    - `bare-metal`

    **The value should be quoted.**

    Example:
    > `- URL: "http://heketi:8080"`

- Certificates - These are the domains we are trying to pull certificates via letsencrypt

There are more options than the above listed. The reference values.yaml should be self-explaining.

### Deploy installer
```bash
kubectl create -f installer/namespace.yaml
kubectl create -f installer/serviceaccount.yaml
kubectl create -f installer/clusterrolebinding.yaml
# values.yaml is the file you created during the step above
kubectl -n kubermatic-installer create secret generic values --from-file=values.yaml
#Create the docker secret - needs to have read access to kubermatic/installer
kubectl -n kubermatic-installer create secret dockercfg regsecret --docker-server=<your-registry-server> --docker-username=<your-name> --docker-password=<your-pword> --docker-email=<your-email>
# Create and run the installer job
kubectl create -f installer/install-job.yaml
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
