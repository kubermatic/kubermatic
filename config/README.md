# Kubermatic

## Setting up a master-seed cluster

### Install helm
Get helm from https://github.com/kubernetes/helm/releases/tag/v2.3.0

Install the tiller:
```bash
helm init
kubectl -n kube-system create sa tiller
kubectl create clusterrolebinding tiller --clusterrole cluster-admin --serviceaccount=kube-system:tiller
kubectl -n kube-system patch deploy/tiller-deploy -p '{"spec": {"template": {"spec": {"serviceAccountName": "tiller"}}}}'
```

### Create own configuration
Create a new seed-cluster config, if needed, via for your domain & modify it according to your needs:
```bash
export DOMAIN=new-setup.kubermatic.io
cp -r seed-clusters/example/ seed-clusters/$DOMAIN
```
Put your `kubeconfig` for the cluster in `seed-clusters/$DOMAIN` and modify the `datacenters.yaml` in the same directory if needed. Replace `- context: name: ` and `current-context: ` in kubeconfig by datacenter name defined in `datacenters.yaml`
Base64 encode the kubeconfig & datacenters.yaml and put them in the the `seed-clusters/$DOMAIN/values.yaml` file.
```bash
cat seed-clusters/$DOMAIN/kubeconfig | base64 -w0
cat seed-clusters/$DOMAIN/datacenters.yaml | base64 -w0
#Update seed-clusters/$DOMAIN/values.yaml
```

### Deploy k8sniff
```bash
helm install -n k8sniff -f seed-clusters/$DOMAIN/values.yaml k8sniff/
```
Go to https://www.ovh.ie/ and add a dns entry for k8sniff:
$DATACENTER=us-central1
- *.$DATACENTER.$DOMAIN  =  *.us-central1.dev.kubermatic.io  

The external ip for the DNS entry can be fetched via
```bash
kubectl -n k8sniff describe service k8sniff-ingress-lb | grep "LoadBalancer Ingress"
```

### Deploy Kubermatic API/Controller
```bash
helm install -f seed-clusters/$DOMAIN/values.yaml -n kubermatic kubermatic/
```

### Deploy ThirdPartyResources
```bash
helm install -f seed-clusters/$DOMAIN/values.yaml -n tpr thirdpartyresources/
```

### Create StorageClass
For every seed cluster (including master) we need to create some generic resources

**Before installing, make sure, that the storageClass is applicable to the seed cluster. (region, provider) Check `seed-clusters/$DOMAIN/values.yaml`**
```bash
helm install -n storage -f seed-clusters/$DOMAIN/values.yaml storage/
```

### Deploy nginx
```bash
helm install -n nginx -f seed-clusters/$DOMAIN/values.yaml nginx-ingress-controller
```

### Create DNS entry for your domain
Go to https://www.ovh.ie/ and add a dns entry for the dashboard:
- $DOMAIN  

The external ip for the DNS entry can be fetched via
```bash
kubectl -n nginx describe service nginx-lb | grep "LoadBalancer Ingress"
```

### Deploy prometheus
```bash
helm install -n prometheus -f seed-clusters/$DOMAIN/values.yaml prometheus
```

### Deploy bare-metal-provider (optional)
```bash
helm install -n bare-metal-provider -f seed-clusters/$DOMAIN/values.yaml bare-metal-provider
```

## Setting up a seed cluster (GKE)
See "Deploy k8sniff" & "Create StorageClass" in "Setting up a master-seed cluster (GKE)"
