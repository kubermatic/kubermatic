## Load Tester:

### Build:
```bash
$ go build
```

### Run:
```bash
$ ./test-kubermatic-load -jwt-auth="...." -datacenter-name="cluster1" -cluster-count=10 up
```
This command creates 10 clusters.

### Cleanup:
```bash
$ ./test-kubermatic-load -jwt-auth="...." -datacenter-name="cluster1" purge
```
This command deletes all running clusters from an users account.

### Flags

  Flag|Optional|Default|Description
  ---|---|---|---
  `-jwt-auth`          | No  | `""`       | `"The String of the authorization header"`
	`-node-count`        | Yes | `0`        | `"The amount of nodes to create in one cluster"`
	`-cluster-count`     | Yes | `0`        | `"The amount of clusters to deploy"`
  `-datacenter-name`   | Yes | `"master"` | `"The master dc"`
	`-max-workers`       | Yes | `10`       | `"The amount of request running at the same time"`
	`-ns-retry-interval` | Yes | `10`       | `"The amonut of time until a NS alive request is send again"`
