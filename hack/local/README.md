# Run e2e tests on your local environment

E2E tests are designed for verifying the functional correctness by replicating end-user behavior from app deployment.
This describes how to run e2e tests in local environment.

## Prerequisites

 - install [kind](https://kind.sigs.k8s.io/)
 - install [jq](https://stedolan.github.io/jq/download/)

### Local DNS resolver

#### Network Manager with dnsmasq-base
Take a look at https://fedoramagazine.org/using-the-networkmanagers-dnsmasq-plugin/

#### Systemd-resolved with dnsmasq
Install and configure `dnsmasq` to set up local domain for the KKP installation.
Dnsmasq is needed only for `local` dns resolution.

Add the following configuration to `/etc/dnsmasq.conf`
```
bind-interfaces
listen-address=127.0.0.1
```
create `kubermatic.local` entry by adding `/etc/dnsmasq.d/kubermatic.local` with
```
address=/.kubermatic.local/172.18.255.200
```
The `address` indicates wildcard subdomains. It's needed to be able to create user clusters.


add a local DNS resolver to systemd-resolved by editing `/etc/systemd/resolved.conf`
```
[Resolve]
DNS=127.0.0.1
Domains=local
```
restart systemd-resolved and dnsmasq
```
systemctl restart dnsmasq
systemctl restart systemd-resolved
```

#### Only dnsmasq
Install dnsmasq and disable all DNS resolvers you have installed on the machine. 
Edit configuration `/etc/dnsmasq.conf` file with:
```
port=53
no-resolv
domain-needed
server=8.8.8.8
bogus-priv
listen-address=127.0.0.2
expand-hosts
domain=kubermatic.local
cache-size=1000
address=/kubermatic.local/172.18.255.200
```

#### Test DNS local resolution
The installation script `run-kubermatic-kind.sh` uses `KUBERMATIC_DOMAIN` env. The default value is: `kubermatic.local`.
Make sure when you install and configure the local DNS resolver the `kubermatic.local` is reachable.
```
$ nslookup kubermatic.local
Server:		127.0.0.2
Address:	127.0.0.2#53

Name:	kubermatic.local
Address: 172.18.255.200
```

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






