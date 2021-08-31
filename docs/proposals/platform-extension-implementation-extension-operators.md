# Platform Extension Implementation: Extension Operators

**Author**: Moritz Bracht

**Status**: Draft proposal.

## Goals

This proposal explores an operator based approach for KKP extensions.

## Motivation and Background

An implementation of platform-extension based on Helm charts provides flexible yet simplistic way to
manage the lifecycle of platform-extensions.

In case of more complex extensions that can not simply be installed with Helm ([like
KubeVirt](https://kubevirt.io/user-guide/operations/installation/), the extensions could be managed
by their own operators.

To cover simple Helm based extensions either a Helm operator could be installed or simple operators
could be [generated directly from Helm
charts](https://sdk.operatorframework.io/docs/building-operators/helm/tutorial/).

[Check this for requirements](./platform-extensions.md#requirements-for-implementations)

## Implementation

Every extension is managed by its own operator. These operators are being managed by
[OLM](https://olm.operatorframework.io/).

### Main Concept Points

The idea is to profit from the life-cycle management of OLM, but abstract away some of the
complexity away from KKP users and use it as "package manager for extension operators".

To make operators available to install with OLM, they first need to be
[packaged](https://olm.operatorframework.io/docs/tasks/creating-operator-manifests/). After creating
[bundles](https://olm.operatorframework.io/docs/tasks/creating-operator-bundle/) from these package
manifests, these bundles are added to an
[Index](https://olm.operatorframework.io/docs/tasks/creating-an-index/) which is a docker image.
This Index image can be referenced in OLM's CatalogSource CRs and will be constantly pulled by OLM's
catalog operator to watch for new Versions of Operators.

* Extensions by KKP are provided by an operator Index image from KKP
* Extensions by users can be added by bundling and indexing them. See CatalogSource Reference in the
  Extension CRD

Life cycle management for Extensions comes with OLM. ClusterServiceVersions of operators can have
minimum Kubernetes versions, so if InstallPlans for Operators are defined accordingly and the
Kubernetes version is upgraded, OLM could automatically upgrade the operator.

### Extension Registration

`Extension`
* Reference to a
  [ClusterServiceVersion](https://olm.operatorframework.io/docs/concepts/crds/clusterserviceversion/)
* Reference to a [CatalogSource](https://olm.operatorframework.io/docs/concepts/crds/catalogsource/)
  (only required for user extensions, will default to KKP extension catalog)

`ExtensionOperator`
* Reference to a [Subscription](https://olm.operatorframework.io/docs/concepts/crds/subscription/)
  * Channel
  * Name
  * Source
  * installApprovalPlan
* Parameters to create CRs managed by the operator
* Status
  * condeses [OperatorCondition](https://olm.operatorframework.io/docs/concepts/crds/operatorcondition/)

### Extension Catalog Controller

Reconciles `Extension` CRs and takes care of registering Operators for Extensions in the Extension
catalog, making them available for installation.  It basically translates `Extensions` to CRs needed
by the OLM operator either wrapping OLM's catalog operator or implementing the necessary parts to
fit our requirements.

### Extension Controller

The extension controller reconciles ExtensionOperator CRs. It runs in the KKP controller manager in
the seed cluster and manages OLM CRs on all user clusters. It acts as a wrapping layer around the
OLM operator which taking care of the whole lifecycle of application operators.

It initially handles ExtensionOperator CRs defined in the cluster template or KKP configuration.
Based on the cluster configuration it allows you to install some extensions by default.

### Architecture

tbd.

## Glossary

See https://olm.operatorframework.io/docs/glossary/

## References

* https://olm.operatorframework.io/docs
