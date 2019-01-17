# Kubermatic development getting started guide

The basic development workflow for Kubermatic works by creating a cluster in a shared
environment, then configuring that cluster to be managed by a locally running controller.

The basic steps to get started on this are these:

* Create a new cluster via the UI on https://dev.kubermatic.io in your browser, it does not matter which provider you use
* Clone the [secrets](https://github.com/kubermatic/secrets/) repo onto your `GOPATH`: `git clone git@github.com:kubermatic/secrets.git $(go env GOPATH)/src/github.com/kubermatic/secrets`
* Decrypt it: `cd $GOPATH/src/github.com/kubermatic/secrets && git-crypt unlock`
    * Note: This requires `git-crypt` to be installed on your computer
* For convenience, add an alias to access the `dev.kubermatic` kubeconfig to your `~/.bashrc`: `echo "dev='export KUBECONFIG=$(go env GOPATH)/src/github.com/kubermatic/secrets/seed-clusters/dev.kubermatic.io/kubeconfig'" >> ~/.bashrc`
* Test if access to the seed works: `source ~/.bashrc && dev && kubectl get cluster`, you should see an output like this:

```
$ kubectl get cluster
NAME                  AGE       HUMANREADABLENAME            OWNER
745rswfsn7            2h        stoic-mccarthy               lukasz.zajaczkowski@loodse.com
9j4q7xh96t            3h        nervous-pasteur              alvaro@loodse.com
fp5lzdp6kx            3h        objective-hopper             lukasz@loodse.com
```

* Every time you use the `dev` alias in your terminal, your `kubectl` command will now be configured to point at the `dev.kubermatic.io` seed cluster :)
* Find out the hostname of your computer by running `uname -n`
* Execute a `kubectl edit $NAME_OF_YOUR_CLUSTER`, replacing `NAME_OF_YOUR_CLUSTER` with the name of the cluster you created in step one, you can see it by using the `kubectl get clsuter`
* Add a label `worker-name: $YOUR_HOSTNAME_FROM_uname -n` to your cluster. This makes the main Kubermatic controller running on dev.kubermatic.io ignore your cluster and the local controller
you will be starting in the next step manage your cluster
* Change your working dir to the `api` subfolder of the Kubermatic repository: `cd $(go env GOPATH)/src/github.com/kubermatic/kubermatic/api`
* If you execute `./hack/run-controller.sh`, a local controller will be built and started in order to manage your cluster :)
* The controller runs in the foreground, meaning it will block the terminal window which is why it is suggested to use a dedicated terminal. You can stop it via `ctrl+c`
* You can now change code, then restart the controller and watch it doing its work

There are also other controllers like the `machine-controller` that do not talk to the seed cluster but to the user cluster directly. This means they need a different Kubeconfig. You can
get them running by following the following steps:

* Configure your shell to point to the seed cluster's `kubeconfig` by executing the `dev` alias created above
* Verify it works by executing `kubectl get cluster`
* Change the namespace for your shell to point to the namespace of your cluster. The namespace is always called `cluster-$CLUSTERNAME`: `kubectl config set-context $(kubectl config current-context) --namespace=cluster-$CLUSTERNAME`
* You can put this command into a function to make your life easier: `echo 'function cn { kubectl config set-context $(kubectl config current-context) --namespace=$1; }' >> ~/.bashrc`, this allows you to run `cn $NAMESPACE_NAME`, e.G. `cn cluster-$YOUR_CLUSTER_ID`
* Now execute `kubectl get pod`, you should see an output similiar to this:

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

* Now that your shell is configured to know about the appropriate namespace, you can start the `machine-controller`. The corresponding script knows how to extract your user cluster's kubeconfig out
of a `secret` that is in the namespace you just configured and points the local `machine-controller` at it: `./hack/run-machinecontroller.sh`
* The controller will now run. It does run in foreground, this means that it will block the terminal window which is why it is suggested to use a dedicated terminal. You can stop the controller by pressing `ctrl + c`
