# CRD Group Migration to K8c.io

**Author**: Mohamed Elsayed (@moelsayed)

**Status**: Draft proposal;

## Goals
Kubernets API groups are organized by namespaces. Most of KPP CRDs are currently using the `k8s.io` namespace. 

The `k8s.io` namespace is owned/managed by the Kubernets community and is currently protected by an API review process. At this point, the Kube-API implements checks to validate if a specific CRD is allowed to use this namespace or not. Unfortunately, we are not allowed. 

This proposal plans to migrate all KPP CRDs from the `k8s.io` namespace into the `k8c.io` namespace. 


## Non-Goals
- Handle KPP global/namespaced resources cleanup.

## Implementation

Implementing this is relatively complex using controller-runtime. It makes more sense to implement this logic in the simplest possible way and try to keep it contained since it should only run once per existing deployment. Newly deployed setups should not have any problems since they will start using the new CRDs from the get go.

The plan is to use client-go dynamic clients to implement the migration code. Ideally it will be included the `kubermatic-installer` and it should apply the following steps during upgrades for existing deployments:
- Deploy the new CRDs with the updated group namespace. At this point the KPP deployment will contain _both_ CRDs.
- Scale down all KPP related deployments. This is a safety precaution to avoid any changes in existing resources.
- List all existing resources with old group namespace.
- Copy over the existing resources with the new group namespace. At this point, the old KPP controllers are still running and will not reconcile the newly created resources.
- KPP upgrade is executed. During this, the new controllers will be deployed and will reconcile the newly created resources with the new group namespace. 
- Once the upgrade is completed, this installer should remove the old CRDs and the old resources from the cluster.

All the converted resources must also checked for `owner reference` to fix any references to the old group namespace.
## Task & effort:

TBA