# Kubermatic

## Setting up a master-seed cluster

### Creating the Configuration
Create a new seed-cluster configuration -> values.yaml.
For a reference see the example [values.yaml][1]

The following keys must be set:
- `KubermaticURL`
  This is the URL to access the dashboard of the cluster. This will also be the subdomain which the client clusters will be assigned under
- `Kubeconfig`
  This is a standard [Kubeconfig][2]. The correct context for the cluster does not need to be set, but the context name must match the key for the seed cluster definition in the `KubermaticDatacenters`

  When defining multiple seed clusters you must have a valid context for each seed cluster in the `Kubeconfig` with the context names must match to the correct keys in `KubermaticDatacenters`

  > The `Kubeconfig` value must be base64 encoded, without any linebreaks

- `KubermaticDatacenters`

  This is the Datacenter definition for Kubermatic. You can find detailed documentation on the Datacenter definition file [here][3].

  > The value for `KubermaticDatacenters` must be base64 encoded without any linebreaks


Make sure you set/update the following:
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

[1]: https://github.com/kubermatic/secrets/blob/master/seed-clusters/example/values.yaml
[2]: https://kubernetes.io/docs/tasks/access-application-cluster/configure-access-multiple-clusters/
[3]: Datacenter File Documentation
