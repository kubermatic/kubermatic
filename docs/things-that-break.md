# Failures

There are a number of failures a Kubernetes cluster operator can encounter over time.
Some of them are caused by humans - some are caused by infrastructure.
This list aims to be a "nice-to-know" list of things that can break and have severe impact in the cluster's health.

## metrics-server not available

The [metrics-server](https://github.com/kubernetes-incubator/metrics-server) is a component which scrapes resource metrics from Pods & Nodes on a regular interval (default=30s).
It works as an [Extension API server](https://kubernetes.io/docs/tasks/access-kubernetes-api/setup-extension-api-server/) and metrics can be gathered using `kubectl top [pod|node]`.

### Caveat

When the metrics-server is not available anymore, the Kubernetes controller-manager stops the processing of namespace deletions as it requires access to the metrics API group.
It seems that the garbage-collector is trying to cleanup metric resources in a terminating namespace judging by the following line found in the logs:
```
W0125 10:48:39.278977       1 garbagecollector.go:647] failed to discover some groups: map[metrics.k8s.io/v1beta1:the server is currently unable to handle the request]
```

