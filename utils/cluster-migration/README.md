# Migrate a cluster

## Basic cluster resources

The following resources will be dumped:
- deployment
- ingress
- daemonset
- secrets
- configmap-
- service
- serviceaccount
- statefulsets

The dump contains json objects which can be imported into a new cluster (UUID, etc. are getting removed). 

```bash
./dump-nfs-pvs.sh
```

The result will be stored in `./dump/namespaces.json` & `./dump/basic-resources.json`

## NFS storage
 
The script will dump PV's and PVC's

```bash
./dump-nfs-pvs.sh
```

The result will be stored in `./dump/pv.json` & `./dump/pvc.json`
