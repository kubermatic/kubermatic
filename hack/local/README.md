# Run e2e tests on your local environment

E2E tests are designed for verifying the functional correctness by replicating end-user behavior from app deployment.
This describes how to run e2e tests in local environment.

## Prerequisites

 - install [kind](https://kind.sigs.k8s.io/)
 - install [jq](https://stedolan.github.io/jq/download/)

### dnsmasq
Install and configure `dnsmasq` to set up local domain for the KKP installation.
The installation script `run-kubermatic-kind.sh` uses `KUBERMATIC_DOMAIN` for this. The default value is: `kubermatic.local`.
The same domain should be set in `/etc/dnsmasq.conf` file. The example looks like this:
```
port=53
no-resolv
domain-needed
server=8.8.8.8
bogus-priv
listen-address=127.0.0.1
expand-hosts
domain=kubermatic.local
cache-size=1000
address=/kubermatic.local/172.18.255.200
```

Make sure when you install and configure the `dnsmasq` the `kubermatic.local` is reachable.
```
$ nslookup kubermatic.local
Server:		127.0.0.1
Address:	127.0.0.1#53

Name:	kubermatic.local
Address: 172.18.255.200

```

The `address` indicates wildcard subdomains. It's needed to be able to create user clusters.
The kind cluster uses [metallb](https://metallb.universe.tf/) to have LoadBalancer service type.
After running `hack/local/run-kubermatic-kind.sh` check assigned external IP address to `nodeport-proxy` service and use it in 
dnsmasq configuration file in `address` section.
```
$ kubectl get service nodeport-proxy -n kubermatic
NAME             TYPE           CLUSTER-IP     EXTERNAL-IP      PORT(S)          AGE
nodeport-proxy   LoadBalancer   10.96.104.73   172.18.255.200   8002:31170/TCP   4m52s
```

The metallb can assign the external IP address from range: 172.18.255.200-172.18.255.204
so it's important to have correct value in `/etc/dnsmasq.conf` file. After changing the IP address restart your `dnsmasq`
service to apply changes:
```
$ sudo systemctl restart dnsmasq
```

## Run e2e tests
### Install KKP
First step spin up KKP cluster in kind.

Execute the following command:
```
$ /hack/local/run-kubermatic-kind.sh 
```

After this you should have all pod running:
```
$ kubectl get pods -n kubermatic
NAME                                                    READY   STATUS    RESTARTS   AGE
kubermatic-api-5bf7c9ff96-zh6cm                         1/1     Running   0          18m
kubermatic-master-controller-manager-66bc6f8679-txsms   1/1     Running   0          21m
kubermatic-operator-775f8fbf9f-csvck                    1/1     Running   0          21m
kubermatic-seed-controller-manager-5dfd9c877-khhxv      1/1     Running   0          18m
nodeport-proxy-envoy-64c9f48744-jl2bs                   2/2     Running   0          18m
nodeport-proxy-envoy-64c9f48744-nvd4m                   2/2     Running   0          18m
nodeport-proxy-envoy-64c9f48744-r4rpf                   2/2     Running   0          18m
nodeport-proxy-updater-7dccbd59b5-5rp7t                 1/1     Running   0          18m
seed-proxy-kubermatic-54678d774b-r96hr                  1/1     Running   0          18m

```

The next step is to check services and set up `nodeport-proxy` external IP in `dnsmasq` configuration file. It will allow
to connect the machine deployment to the user cluster API server. It's described in [dnsmasq](#dnsmasq) section.

#### Run local UI

You can also connect your KKP to the dashboard. You have to only make one change for the DEX in `dashboard/src/environments/environment.ts`
Enter the new DEX address:
```
oidcProviderUrl: 'http://dex.oauth:5556/dex/auth',
```

Now you can start the UI:

```
$ npm run start:local
```

For the login use static user:
```
login: roxy@loodse.com
password: password
```

### Set up and run e2e tests

Execute the following command:
```
$ /hack/local/run-api-e2e.sh 
```






