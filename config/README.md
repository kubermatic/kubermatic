# Kubermatic

## Setting up a master-seed cluster

### Create own configuration
Create a new seed-cluster config, if needed, via for your domain & modify it according to your needs:
```bash
export DOMAIN=new-setup.kubermatic.io
cp -r deploy seed-clusters/$DOMAIN
```
Put your `kubeconfig` for the cluster in `seed-clusters/$DOMAIN` and modify the `datacenters.yaml` in the same directory if needed. Replace `- context: name: ` and `current-context: ` in kubeconfig by datacenter name defined in `datacenters.yaml`

### Deploy kubermatic

Run the script `./deploy.sh` and answer all questions

```bash
./deploy.sh
```
### Create DNS entry for your domain
Go to https://www.ovh.ie/ and add a dns entry for the dashboard:
- $DOMAIN  

The external ip for the DNS entry can be fetched via
```bash
kubectl -n nginx describe service nginx-ingress-controller | grep "LoadBalancer Ingress"
```

Go to https://www.ovh.ie/ and add a dns entry for k8sniff:
$DATACENTER=us-central1
- *.$DATACENTER.$DOMAIN  =  *.us-central1.dev.kubermatic.io  

The external ip for the DNS entry can be fetched via
```bash
kubectl -n k8sniff describe service k8sniff-ingress-lb | grep "LoadBalancer Ingress"
```
