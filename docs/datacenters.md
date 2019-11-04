# The `datacenters.yaml` File

The `datacenters.yaml` file describes the datacenters that Kubermatic can deploy clusters to. This includes the cloud provider details, regions or zones that are supported, if it is a seed cluster, authorization credentials, API addresses, etc...

### What is a Seed Cluster?

Seed clusters are Kubernetes clusters that are used by Kubermatic to deploy Kubernetes controlplanes of user clusters in.
They must be an already deployed Kubernetes cluster, and must have a Kubeconfig entry with a matching name. Though the Kubermatic components only get deployed into a single seed cluster, the other seed clusters are controlled via those same Kubermatic components.

Seed datacenters are identified by setting `is_seed` to `true` in the Datacenter block.

Alternatively, it is also possible to enable the `--dynamic-datacenters` feature on the controllers, which will make them read the `Seeds` from a custom resource definition.
Check out [the generated example](/docs/zz_generated.seed.yaml) for a full sample.

### Datacenter Block

  The datacenter block is defined by the name of the datacenter.

  The datacenter block has the following keys:
  - `location`
    Free text identifying the geographical or other location information about the datacenter
  - `country`
    The [ISO 3166 Alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2#Current_codes) Country Code for the Datacenter
  - `provider`
    Name identifying the datacenter provider
  - `is_seed`
    Boolean specifying if this is a seed cluster
  - `spec`
    A [Datacenter Spec](#spec) block

### Datacenter `spec` Block

The `spec` block for a datacenter identifies which datacenter `provider` to use, and specifies the options for it.

Currently supported providers are:
- `aws`
- `bringyourown`
  **Keys:**
  - `region`
    A name identifying the region for the 'datacenter'
- `digitalocean`
  **Keys:**
  - `region`
    The name of the DigitalOcean region. You can find a full list of the DigitalOcean regions on their [status page](https://status.digitalocean.com/)
- `openstack`


Full Example:
```
datacenters:
  europe-west3-c:
    location: Frankfurt
    country: DE
    provider: Loodse
    is_seed: true
    spec:
      bringyourown:
        region: DE
      seed:
        bringyourown:
  do-ams2:
    location: Amsterdam
    seed: europe-west3-c
    country: NL
    spec:
      digitalocean:
        region: ams2
  aws-us-east-1a:
    location: US East (N. Virginia)
    seed: europe-west3-c
    country: US
    provider: aws
    spec:
      aws:
        ami: ami-ac7a68d7
        region: us-east-1
        zone_character: a
  os-hamburg:
    location: Hamburg
    seed: europe-west3-c
    country: DE
    provider: openstack
    spec:
      openstack:
        auth_url: https://api.openstack.org:5000/v3
        availability_zone: openstack1
        dns_servers:
        - "1.2.3.4"
        - "1.2.3.5"
  bm-dc1:
    location: US-Central
    country: US
    spec:
      baremetal:
        url: "http://bare-metal-provider.kubermatic.io"
        auth-user: "<username>"
        auth-password: "<password>"
```
