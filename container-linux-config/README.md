# Container Linux Config Generator

This tool generates Container Linux configs (+Ignition) for creating Kubernetes clusters.

## Requirements
- Only single master clusters can be created at the moment
- Static IP's which are known beforehand
- Container Linux which supports ignition
- An already existing CA (Or can be generated with this tool)
- An already existing ServiceAccountKey (Or can be generated with this tool)

## Config

To create a new Kubernets cluster, a config needs to be created beforehand.

The config looks like the following:
```yaml
Global:
  CA:
    # Path to the CA key
    KeyPath: "/home/ca.key"
    # Inline CA key (Either KeyPath or Key must be set)
    Key: ""
    # Path to the CA cert
    CertPath: "/home/ca.crt"
    # Inline CA cert (Either CertPath or Cert must be set)
    Cert: ""
  ServiceAccount:
    # Path to the key
    KeyPath: "/home/sa.key"
    # Inline service account key # Inline key (Either KeyPath or Key must be set)
    Key: ""
  Kubernetes:
    # IP address for the master
    MasterIP: "10.0.2.147"
    # Kubernetes version. Must be >=v1.10.0
    Version: "v1.10.1"
    # Node port range for the cluster
    ServiceNodePortRange: "30000-32767"
    # Service network
    ServiceClusterIPRange: "10.10.10.0/24"
    # Pod network
    PodNetworkIPRange: "172.25.0.0/16"
    # Service IP for in cluster DNS
    DNSIP: "10.10.10.10"
    # in cluster dns domain
    DNSDomain: "cluster.local"
  SSHKeys:
  # List of ssh pub keys. Will get deployed onto every node (user: core)
  - "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQC496PqMk7zZuQKXpzcgl3g92LNZMcgEgkfnj4BNoa2CRs5f4wrHMt1Ii78CgY3Ziqw4y191v6IJ3Fby2gT26YWIaK6qSF35sXmjyG/MH+Iu96wIn7JcjlnTkizofGE7zOkRmEtMYcyOVbgnwEH4UNj1W1aalton5JCAqz8kFpKE7sp/2NpxZMqgO68NaqYSd0nDXdwM9HX1grWVF88lBXEJ5YSF03sEpnsOyw0QHhXHyNwHOpfL/wCG0/8OWel1dcXOeQKVVp5V045er0AyIFJVwHSTQCnfyKYCf5i0cLci4+61iSS1hBSl2qBqz6z6mdLy8O7XSP9kPoyQarnQguFO+DHOG8IyTP/Kk3PC/+yR8xGfhFxqM8fGdSdrRwpIgjGH0/7or1vbbUvp52evRMZIdT1YfndahRxafo0iZ36o+VgX1W5oWiZ8Bws/zrV2PCqrmuEbFYSpLati3XvxRQG+Om5p7dzdXx1mb86wqyNe4xpZ2gVInaUYINnwcUmLiaOtCHh4tE/RBMqjcRY7i75gl29H8nbDFwWHGLTULix1W+1FH0XyPiL831XdECYSlw8h+kJW/fYtlynTgeYSLSrIA0atEblf1ui0BPnd8imFX7h8mlUhPgxpfxasI+EQj+m4xjeJUalHTvFuE1wqls8A7A6+J8HP/Ol9cOlDFxexw== henrik@loodse.com"
# List of nodes for the cluster
# Only a single one must be master. This master should have etcd configured.
Nodes:
  # Name of the node
- Name: "master-1"
  # Node type. Either "master" or "worker"
  Type: master
  Mounts:
  # List of mounts
  - Name: "dev-sdb"
    Device: "/dev/sdb"
    Where: "/srv/etcd/data"
    Type: "ext4"
    DirectoryMode: "0777"
  # Run etcd on this node
  Etcd:
    Enabled: true
    Version: "3.1.12"
    DataDirectory: "/srv/etcd/data"
  # Network settings for the node
  Network:
    # If tue, a the network on this node will be configured based on the following specs
    Configure: true
    Interface: en*
    Address: 10.0.2.147/24
    Gateway: 10.0.2.1
    Broadcast: 10.0.2.255
    DNSServers:
    - 10.0.2.1
    Domains:
    - foo.local
    NTPServers:
    - 1.2.3.4

- Name: "worker-1"
  Type: worker
  Network:
    Configure: true
    Interface: en*
    Address: 10.0.2.149/24
    Gateway: 10.0.2.1
    Broadcast: 10.0.2.255
    DNSServers:
    - 10.0.2.1

```
