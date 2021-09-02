# Integration of external cloud controllers

**Author**: Alvaro Aleman

**Status**: Draft

Currently, we use the in-tree cloud providers to offer cloud provider functionality. Those are deprecated,
which means they are not getting new features and will eventually be removed. Thus, the development shifted
to the external cloud controller.


## Goals

* Deploy the external cloud controller manager for newly created clusters on all platforms where it had
  at least two stable releases
* Deploy the corresponding CSI driver for newly creates clusters on all platforms where it had at least
  two stable releases

## Non-Goals

* Migrate clusters that have or had an in-tree provider running to the external one

## Implementation

Since this should be deployed for new clusters only, we must introduce a feature gate mechanism on cluster
level. The following addition to the `ClusterSpec` type is proposed:

```
Features map[string]bool
```

All features and their status will be shown in the `Features` property, allowing operators to easily discover Features
and enable or disable them.

The external cloud provider feature will be recognized by a `const FeatureNameExternalCloudProvider = "externalCloudProvider"`

The API will set this feature gate on all newly creates clusters on providers that support it.
If the gate is present:

* The kubermatic-controller-manager will deploy the machine-controller with the `--external-cloud-provider=true` flag
* The cloud-controller loops will be disabled on the Kubernetes controller manager
* An addon for CSI will get automatically deployed into the cluster
* The LoadBalancer cleanup must be adjusted to not wait for an event anymore

Additionally, the `conformance-tests` CLI needs an extra arg to control if it should set the feature gate and a distinct
e2e test must be added that verifies the functionality of LoadBalancers and Persistent Volume provisioning.



## Task & effort:

* Extend the `Cluster` CRD, add branching in the existing deployments for the feature and extend the conformance tester: 0.5d
* Figure out the configuration details of the external CCM and its corresponding CSI extension, write and execute test job: 0.5d per platform
