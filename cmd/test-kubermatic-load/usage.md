## Load Tester:

### Build:
In the `test-kubermatic-load` directory execute:
```bash
$ go build
```

### Run:
#### Get the jwt:
To get our authentication token we need to login under https://dev.kubermatic.io.
In the Chrome LocalStorage in the developer windows, we copy the value of the field `token`.
The final authentication string is `"Bearer <token>"`, this string will be passed with the `-jwt-auth` flag every time we call the script.
#### Execute script:
```bash
$ ./test-kubermatic-load -jwt-auth="...." -datacenter-name="cluster1" -cluster-count=10 up
```
This command creates 10 clusters.
*You can only run one load test. After you run it once you have to ourge your clusters*

### Cleanup:
```bash
$ ./test-kubermatic-load -jwt-auth="...." -datacenter-name="cluster1" purge
```
This command deletes all running clusters from an users account.

### Flags
Flag|Optional|Default|Description
---|---|---|---
`-jwt-auth`          | No  | `""`                 | `"The String of the authorization header"`
`-node-count`        | Yes | `0`                  | `"The amount of nodes to create in one cluster"`
`-cluster-count`     | Yes | `0`                  | `"The amount of clusters to deploy"`
`-datacenter-name`   | Yes | `"master"`           | `"The master dc"`
`-max-workers`       | Yes | `10`                 | `"The amount of maximum concurrent requests"`
`-ns-retry-interval` | Yes | `10`                 | `"The duration in seconds to wait between namespace alive requests"`
`-domain`            | Yes | `dev.kubermatic.io"` | `"The domain to api is running on"`
