# **KCP**

**Author**: Patryk Przekwas (@pkprzekwas)

**Status**: Draft proposal; research.

**Issue**: https://github.com/kubermatic/kubermatic/issues/8948

## Goals

The document focuses on an evaluation of [kcp](https://github.com/kcp-dev/kcp) to expose Kube API to KKP end-users.

## Non-Goals

This text will not cover extensively `kcp`'s approach to transparent multi-clustering. The main reasons are the premature state of implementation and the clash with solutions that are already working fine in the KKP.

## Motivation and Background

### Current state

The KKP API cannot be exposed through tools like `kubectl` to non-admin platform users. The main reasons are:
* many Kubermatic resources are cluster scoped
* no mechanism for suppressing fields in CRDs (e.g. exposing Clusters CRD but without `DebugLog`)

### Expected state

As a KKP project owner, I want to be able to interact with the project using native tools, like `kubectl`, `helm`, `flux`, or `argoCD`. At the same time, some options ought not to be exposed, e.g. enabling `DebugLog` in the user clusters (only administrators should be able to do it).

### What is kcp?

`kcp` is a minimal Kubernetes api-server. It doesn't know about `pods`, `deployments`, `services`, or even `nodes`. Some resources like `namespaces` or `serviceAccounts` have been preserved for multi-tenancy purposes. The tool is a good base for creating declarative APIs with the use of CRDs and a reconciliation loop. Putting it shortly, imagine a declarative Kubernetes-style API for anything separate from the Kubernetes container orchestrator. 

Moreover, additional controllers have been embedded inside `kcp`. Those are designed for advanced multi-tenancy and multi-clustering scenarios. This document will cover only the former, as multi-clustering functionalities double some of existing KKP functionalities.

This is still a prototype. APIs are not fixed and are expected to change.

### Terminology

* `kcp` - the primary executable which includes functionalities of minimal api-server. Also, the name of the prototype.

* `logical-cluster` - a logical cluster is a way to subdivide a single kube-apiserver + etcd storage into multiple clusters (different APIs, separate semantics for access, policy, and control) without requiring multiple instances. A logical cluster is a mechanism for achieving separation but it's possible to modele it differently in different use cases. A logical cluster is a storage level concept that adds an attribute to an object’s identifier on a kube-apiserver. Regular servers identify objects by (group, version, resource, optional namespace, name).  A logical cluster prepends an identifier: (logical cluster name, group, version, resource, optional namespace, name).

* `physical-cluster` - a physical cluster is a “real Kubernetes cluster”, i.e. one that can run Kubernetes workloads and accepts standard Kubernetes API objects. For the near term, it is assumed that a physical cluster is a distribution of Kubernetes and passes the conformance tests and exposes the behavior a regular Kubernetes admin or user expects.

* `workspace`- a workspace models a set of user-facing APIs for CRUD.  Each workspace is backed by a logical cluster, but not all logical clusters may be exposed as workspaces. Creating a Workspace object results in a logical cluster being available via a URL for the client to connect and create resources supported by the APIs in that workspace. Workspace is intended to be the most generic representation of the concept with the broadest possible utility to anyone building control planes. To a user, a workspace appears to be a Kubernetes cluster minus all the container orchestration-specific resources. It has its own discovery, its own OpenAPI spec, and follows the kube-like constraints about the uniqueness of Group-Version-Resource. A user can define a workspace as a context in a kubeconfig file and `kubectl get all -A` would return all objects in all namespaces of that workspace.

* `virtual workspace` - an API object has one source of truth (is stored transactionally in one system), but may be exposed to different use cases with different fields or schemas. Since a workspace is a user-facing interaction with an API object, if we want to deal with Workspaces in aggregate, we need to be able to list them. Since a user may have access to workspaces in multiple different contexts, or for different use cases (a workspace that belongs to the user personally, or one that belongs to a business organization), the list of “all workspaces” itself needs to be exposed as an API object to an end-user inside a workspace. That workspace is “virtual” - it adapts or transforms the underlying source of truth for the object and potentially the schema the user sees.

* `index` (e.g. workspace index) - An index is the authoritative list of a particular API in their source of truth across the system. For instance, for a user to see all the workspaces they have available, they must consult the workspace index to return a list of their workspaces. 

### Forking upstream Kubernetes

In order to build a minimal api-server without base resources like pods and add support for logical clusters backed by etcd modifications, the `kcp` prototype forked the official Kubernetes repository and introduced changes described in the following [doc](https://github.com/kcp-dev/kubernetes/blob/feature-logical-clusters-1.23/KCP_RELATED_CHANGES.md). Authors intend to pursue these changes through the usual KEP process.

## Implementation

In the terminology section of this document, we see that some of the described terms are fitting nicely to the KKP's problem with building Kube-like API for end-users. Concepts like Projects in KKP could benefit combined or replaced with Workspaces.

Unfortunately, `kcp` is still a prototype, meaning that the vast majority of ideas described in the terminology section are either not implemented or don't have a fixed API. Building PoC based on `kcp` is hard or not possible at this stage of the prototype.

## Alternatives considered

While `kcp` and its Workspaces have great multi-tenancy capabilities (at least on paper), there are other ongoing initiatives to solve K8s multi-tenancy issues, e.g. [A Multi Tenant Framework for Cloud Container Services](https://github.com/kubernetes-sigs/multi-tenancy/blob/master/incubator/virtualcluster/doc/vc-icdcs.pdf).

Let's imagine having a control plane instance per KKP project. It is a bit of overhead from the resource perspective. But it gives a lot of flexibility. For instance, it allows for installation CRDs designed to be exposed for end-user or enables user to switch between projects by simply updating context in an active kubeconfig. This solution would require introducing some mechanism to synchronize states between different control planes.

## Task & effort:

For now, I suggest waiting till `kcp` promotes from prototype phase to project. It wants to solve too many problems at the same time.

I suggest checking the status of the prototype every 3-4 months.
