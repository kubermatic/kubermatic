# Kubernetes at Digital Ocean

This directory holds scripts for bootstrapping a Kubernetes cluster on Digital Ocean.
The following variables need to be defined:

```
export TOKEN=xxxxx73bf74b4d8eca066c71b172a5ba19ddf4c7910a9f5a7b6e39e26697c2d6
export SSH_FINGERPRINT=xx:xx:xx:xx:a0:3c:48:c1:15:ab:19:db:52:cb:08:97
```

The token must be from Digital Ocean web UI, see https://cloud.digitalocean.com/settings/api/tokens.
The ssh fingerprint can be generated using:
```
$ ssh-keygen -E md5 -lf ~/.ssh/id_rsa.pub | cut -d ' ' -f 2 | cut -d ":" -f 2-
```

The `SSH_FINGERPRINT` variable can hold more than one fingerprint separeted by comma `,`, i.e.:

```
export SSH_FINGERPRINT=xx:xx:xx:xx:a0:3c:48:c1:15:ab:19:db:52:cb:08:97,yy:yy:yy.....,zz:zz:zz.....
```

Register the corresponding public key in DigitalOcean.

Create the admin and the kubelet token:
```
$ export KUBELET_TOKEN=$(openssl rand -base64 18)
$ export ADMIN_TOKEN=$(openssl rand -base64 18)
```

To create a single-instance master execute:
```
$ export DISCOVERY_URL=$(curl -s 'https://discovery.etcd.io/new?size=1')
$ export K8S_DISCOVERY_URL=$(curl -s 'https://discovery.etcd.io/new?size=1')
$ REGION=fra1 SIZE=1gb NAME=master go run create.go cloud-config-master.yaml
```

To create a three- or five-instance master cluster, modify the `?size=...` discovery parameters
and execute `go run create.go cloud-config-mater.yaml` as many times as there are etcd cluster nodes.

Assign the floating ip in DigitalOcean to (one of) the new droplet(s).

Once the node is available update the kube config to point to `https://do-<region>.kubermatic.io`.

Create nodes using the UI.

To create the `kube-system` namespace, skydns and the ingress controller, execute:
```
$ go run addons/main.go
```
