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

## Kubernetes defaulting

Our controllers reconcile a lot of objects within Kubernetes.
During each iteration our controller compares the objects which exist in Kubernetes with the ones it generated locally.
If there's a difference it would update the object in Kubernetes.

Kubernetes defaults certain fields on objects, which can lead to problems with our controllers.
Example:
- Our controller creates an object
- Kubernetes defaults a field
- The controller sees a difference between the object in Kubernetes and the one generated locally (The defaulted field)
- The controller updates the object (We now reached an endless loop of updates)

To avoid such loops, our controllers try to set all fields which would get defaulted.

This works well, except for situations where Kubernetes introduces new fields which get defaulted in a new version.
Example:
Kubernetes v1.12 introduced a new field on containers:
```bash
kubectl explain deployment.spec.template.spec.containers.securityContext.procMount

KIND:     Deployment
VERSION:  extensions/v1beta1

FIELD:    procMount <string>

DESCRIPTION:
     procMount denotes the type of proc mount to use for the containers. The
     default is DefaultProcMount which uses the container runtime defaults for
     readonly paths and masked paths. This requires the ProcMountType feature
     flag to be enabled.
```

`procMount` gets defaulted to `Default`.
In kubernetes v1.11 this field does not exist & will be removed.

This leads to an endless loop of updates on Kubernetes v1.11
On Kubernetes v1.11
- Our controller creates the object with `procMount=Default`
- Kubernetes removes that field (Since it does not exist in v1.11)
- Our controller sees a difference between the object in Kubernetes and the one generated locally (`procMount` got removed)
- The controller updates the object (We now reached an endless loop of updates)

### Caveat

In case of differences between the controllers & kubernetes defaulting, 
we created an alert for dev.kubermatic.io which gets triggered on excessive update actions. `KubermaticControllerManagerHighPutRate`
