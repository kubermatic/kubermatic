# Seed Cluster Setup
The seed installer is for creating a self-contained Kubernetes cluster that hosts the Kubermatic components. The cluster has to pass the conformance tests and has to interact with a cloud provider.
---

## How it works.
This installer locally renders assets, copies them to the corresponding machines, installs dependencies on the machines and runs scripts. For this purpose it uses ssh to connect to the machines, thus it requires passwordless sudo. (e.g ubuntu@XMachine) ubuntu needs sudo permissions.

## Prerequisites.
* All machines need to be accessible over an keyfile via ssh.
* All public IP's of the etcd servers (count = 1,3,5,...), the IP's are just used to connect to the machines (can also be the private one if install from the network within).
* All private IP's of the etcd servers.
* All public IP's of the master servers (public as in public Etcd IP's, can be the same as the etcd servers, to have both on the same machine).
* All private IP's of the master server.
* All public IP's of the workers (public as in public Etcd IP's, have to be distinct from the master IP's).
* All private IP's of the workers.
* The LoadBalancer IP (If not existent use a master server IP).
* The default interface name for the private network (e.g eth1)
* The cloud-provider-config path.
* The cloud provider used (e.g openstack).
* The default user used during installation.

In the `config.sh` script edit the variables and run `./install.sh`

```bash
KUBERNETES_VERSION="v1.9.2"
CLOUD_PROVIDER_FLAG=openstack
CLOUD_CONFIG_FILE=./path-to-cloud-conf
DEFAULT_PRIVATE_IP4_INTERFACE=eth1
DEFAULT_LOGIN_USER=ubuntu
ETCD_HOSTNAMES=(etcd-hostname1 etcd-hostname2 etcd-hostname3)
ETCD_PRIVATE_IPS=(etcd1-private-ip etcd2-private-ip etcd3-private-ip)
ETCD_PUBLIC_IPS=(etcd1-public-ip etcd2-public-ip etcd3-public-ip)

POD_SUBNET="10.244.0.0/16" # Canal

MASTER_LOAD_BALANCER_ADDRS=(LoadBalancerIP)
MASTER_HOSTNAMES=(seed-master-1 seed-master-2 seed-master-3)
MASTER_PRIVATE_IPS=(master1-private-ip master2-private-ip master3-private-ip)
MASTER_PUBLIC_IPS=(master1-public-ip master2-public-ip master3-public-ip)

# Additional Worker IP's (Don't enter APISERVER IP)
WORKER_PRIVATE_IPS=(worker1-private-ip)
WORKER_PUBLIC_IPS=(worker1-public-ip)
```

## Upgrading the cluster

First drain the node you want to update.
```bash
kubectl drain <node name>
```
Next edit `/etc/kubernetes/kubeadm-config.yaml` and set the kubernetes version.

Now you can simply initialize this node with the new kubernetes version like:

`sudo kubeadm init --config /etc/kubernetes/kubeadm-config.yaml --ignore-preflight-errors all`

Once that's done you should see the apiserver, node-controller and scheduler restarting.
These components are now running in the new version.

Don't forget to undrain the node again.
```bash
kubectl uncordon <node name>
```

Repeat for all other nodes one by one.
