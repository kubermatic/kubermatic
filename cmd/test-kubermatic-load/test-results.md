# Test results.

## First test
Spec:
* No nodes
* 500 Clusters over time

```bash
  $  go run main.go -jwt-auth="${JWT_AUTH}" -domain="staging.kubermatic.io" -datacenter-name="us-central1" -cluster-count=500 -max-workers=2 -ns-retry-interval=200 up
```
####Results:
After roughly 160-170 cluster creations the first servers started to fail.
####Reason:
The master kubernetes server (hosted by Google) could not keep up with creating new pods.
Due to this the clusters switched after 5 min from `Pending` to `Failing`.
####Fixes:
* Increase the time until a cluster is marked `Failing`
* Pre create clusters
* Create quotas
* Self hosted kubernetes

## Second test
* 150 Clusters over time
####Results:
After roughly 130-140 cluster creations the first servers started to fail.
####Observations:
* The dashboard/api displays most clusters as healthy.
Manual verification shows most those clusters could not be started due to hardware limitations.
It was therefore not possible to connect to the cluster with the generated kubeconfig.
```kubectl --kubeconfig ~/Downloads/kubeconfig get ns
The connection to the server gqm0b0vnj.us-central1.staging.kubermatic.io was refused - did you specify the right host or port?
``` 
* k8s is constantly dieing (25 times)
* 
####Reason:
* The cluster is to small
####Fixes:
* Increase the time until a cluster is marked `Failing`
* Pre create clusters
* Create quotas
* Self hosted kubernetes
