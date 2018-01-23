# Kubermatic

## Setting up a master-seed cluster

### Create own configuration
Create a new seed-cluster configuration -> values.yaml.
For a reference see the values.yaml from the [dev.kubermatic.io](https://github.com/kubermatic/secrets/blob/master/seed-clusters/dev.kubermatic.io/values.yaml)

Make sure you set/update the following:
- `KubermaticURL`
- `Kubeconfig` -> Needs to be base64 encoded
- `KubermaticDatacenters` -> Needs to be base64 encoded
- Storage - Have a look at the helm chart helper under config/storage/templates/_helpers.tpl
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
kubectl  -n kubermatic-installer create secret docker-registry dockercfg --docker-username='' --docker-password='' --docker-email=''
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
