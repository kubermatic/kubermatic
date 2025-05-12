# Kubermatic development getting started guide

The basic development workflow for Kubermatic works by creating a cluster in a shared
environment, then configuring that cluster to be managed by a locally running controller.

The basic steps to get started on this are these:

* Login to our vault; this is required for the scripts mentioned below to work.
* Clone the dashboard repo onto your `GOPATH`: `git clone git@github.com:kubermatic/dashboard.git $(go env GOPATH)/src/k8c.io/dashboard`
* Start all the components via the respective scripts. All of them are blocking, so it is suggested to start a terminal instance for each:
  * API: `$(go env GOPATH)/src/k8c.io/dashboard/modules/api/hack/run-api.sh`
  * Dashboard: `$(go env GOPATH)/src/k8c.io/dashboard/modules/web/hack/run-local-dashboard.sh`
  * Run the desired controller:
    * Seed Controller Manager: `$(go env GOPATH)/src/k8c.io/kubermatic/hack/run-seed-controller-manager.sh`
    * User Cluster Controller Manager: `$(go env GOPATH)/src/k8c.io/kubermatic/hack/run-user-cluster-controller-manager.sh`
    * Master Controller Manager: `$(go env GOPATH)/src/k8c.io/kubermatic/hack/run-master-controller-manager.sh`
    * KKP Operator: `$(go env GOPATH)/src/k8c.io/kubermatic/hack/run-operator.sh`

Now you can visit <http://localhost:8000> in your web browser, log in and create a cluster
at a provider of your choice. The result can then be viewed by looking into the respective
seed cluster:

* For convenience, add an alias to access the `dev.kubermatic` seed clusters kubeconfig to
  your `~/.bashrc`: `echo "dev='export KUBECONFIG=$(go env GOPATH)/src/github.com/kubermatic/secrets/seed-clusters/dev.kubermatic.io/kubeconfig'" >> ~/.bashrc`
* Test if access to the seed works: `source ~/.bashrc && dev && kubectl get cluster`, you
  should see an output like this:

```
$ kubectl get cluster
NAME                  AGE       HUMANREADABLENAME            OWNER
745rswfsn7            2h        stoic-mccarthy               user1@example.com
9j4q7xh96t            42d       nervous-pasteur              user2@example.com
fp5lzdp6kx            8h        objective-hopper             user3@example.com
```

* Every time you use the `dev` alias in your terminal, your `kubectl` command will now be
  configured to point at the `dev.kubermatic.io` seed cluster :)
* You can now change code, then restart the controller(s) and watch them doing their work.

*Hint:* If you only work on the API you can skip starting the controllers. If you only work
on controllers, you can skip starting the UI and API and instead create a cluster at
`https://dev.kubermatic.io`,
and add a label `worker-name:` to it with the output of `uname -n | tr -cd '[:alnum:]' | tr '[:upper:]' '[:lower:]'` as value. This will make
the controllers running in that seed cluster ignore your cluster and make the local controllers
take care of it.

There are also other controllers like the `machine-controller` that do not talk to the seed
cluster but to the user cluster directly. This means they need a different Kubeconfig. You can
get them running by following the following steps:

* Configure your shell to point to the seed cluster's `kubeconfig` by executing the `dev` alias
  created above.
* Verify it works by executing `kubectl get cluster`.
* Change the namespace for your shell to point to the namespace of your cluster. The namespace is
  always called `cluster-$CLUSTERNAME`: `kubectl config set-context $(kubectl config current-context) --namespace=cluster-$CLUSTERNAME`.
* You can put this command into a function to make your life easier:
  `echo 'function cn { kubectl config set-context $(kubectl config current-context) --namespace=$1; }' >> ~/.bashrc`,
  this allows you to run `cn $NAMESPACE_NAME`, e.G. `cn cluster-$YOUR_CLUSTER_ID`
* Now execute `kubectl get pod`, you should see an output similar to this:

```
$ kubectl get pod
NAME                                          READY     STATUS    RESTARTS   AGE
apiserver-567dd9c866-xrfnp                    3/3       Running   0          57m
controller-manager-8f79b4859-rszcs            2/2       Running   0          57m
dns-resolver-6455f9dbd6-dnvpb                 2/2       Running   0          57m
dns-resolver-6455f9dbd6-lz9l7                 2/2       Running   0          57m
etcd-0                                        1/1       Running   0          57m
etcd-1                                        1/1       Running   0          57m
etcd-2                                        1/1       Running   0          57m
kube-state-metrics-55fc4ddbd-xr64z            1/1       Running   0          55m
machine-controller-78fddf9bd7-lczj4           1/1       Running   0          57m
machine-controller-webhook-79b4c48ff7-rnr5c   1/1       Running   0          57m
metrics-server-5b7848478d-tw79g               3/3       Running   0          57m
openvpn-server-b7bd9864-bgq5w                 2/2       Running   0          57m
prometheus-0                                  1/1       Running   0          55m
scheduler-77c956dbf6-c7cgh                    2/2       Running   0          57m
```

* Now that your shell is configured to know about the appropriate namespace, you can start the
  `machine-controller`. The corresponding script knows how to extract your user cluster's
  kubeconfig out of a `secret` that is in the namespace you just configured and points the
  local `machine-controller` at it: `./hack/run-machinecontroller.sh`
* The controller will now run. It does run in foreground, this means that it will block the
  terminal window which is why it is suggested to use a dedicated terminal. You can stop the
  controller by pressing `ctrl + c`.

## Pausing resource updates

If you ever have to change something on a KKP-managed resource manually, you can annotate that resource with
`hacking.k8c.io/pause: "true"`. This will prevent Kubermatic controllers from modifying the resource (but not
protect it from being deleted).

Great for development and debugging, but really bad to find this in production - so please never use that
outside development.
