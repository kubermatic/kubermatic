# Title of the Proposal: e.g. **kube-node**

**Author**: Sebastian Scheele (@scheeles)

**Status**: Draft proposal; prototype in progress.

## Goals

*short description of the topic e.g.*
kube-node is a minimalistic API which enables Kubernetes to manage its nodes by itself. The goal is to have a higher abstraction layer for managed k8s nodes and to be able to integrate Kubernetes with different providers in a generic way. It is intended to live outside of core Kubernetes and add optional node management features to Kubernetes clusters.

## Non-Goals

A short description on where the scope of the proposal ends

## Motivation and Background

*What is the background and why do we want to deploy it e.g.*


The current approaches to deploy and manage k8s nodes come with several downsides:

*   Require Ops to scale the cluster
*   No generic approach for different providers
*   Each implementation requires specific knowledge
*   Scaling of new nodes requires external dependencies
*   No generic node auto-scaling across different providers available
*   Huge effort to deploy k8s on a different platform

The idea is to make the deployment of kubernetes much simpler and create a native integration for kubernetes node deployment.

With kube-node, to create a native integration for kubernetes nodes and achieve that:

*   Dev can scale the cluster
*   Nodes become cattles, not pets
*   Implementation of generic auto scaling possible
*   Very similar setup for different providers


## Implementation

A short and concise description of all steps of the intended implementation

## Alternatives considered

A list of alternative implementations to solve the problem at hand and a short description of why they were discarded

## Task & effort:
*Specify the tasks and the effort in days (samples unit 0.5days) e.g.*
* Implement NodeClass - 1d
* Enhance kubermatic controller - 2d
