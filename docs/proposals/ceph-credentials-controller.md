# Ceph Credentials Controller

**Author**: Kamil Domański (@kdomanski)
**Status**: Draft proposal; prototype in progress.

## Motivation and Background

Some of our clients would like to use Ceph RBD as the underlying block storage
for Kubernetes volume claims. These Ceph clusters are existing installations
running next to Kubernetes. In order to allow child clusters to access
those, a `StorageClass` needs to be added to each of those clusters
and configured with Ceph access credentials.

Because of performance issues caused by too many Ceph pools, all client clusters
are expected to use the same pool. However, reusing the same credentials for
all of them would make them difficult to revoke or rotate. For that reason,
each of the child clusters should be provided with a separate set of credentials
granting access to the same pool.

## Implementation

Extend the datacenter definition with Ceph admin access credentials.

Create a new `ceph-credentials-controller` as a part of `kubermatic-controller-manager`.

The controller will:
- upon finding a newly created cluster in datacenters with Ceph configurations:
  1. create new access credentials for the Ceph pool
  1. use these credentials to generate a new `StorageClass` object
  1. apply this object to the new Kubermatic cluster
  1. mark the cluster with a `ceph-credentials` finalizer
- upon finding a cluster marked for deletion in datacenters with Ceph configurations:
  1. check for `ceph-credentials` finalizer and ignore the cluster if the finalizer is not found
  1. if the finalizer is found, the associated Ceph credentials shall be deleted from the Ceph cluster
  1. the `ceph-credentials` will be cleared

## Tasks

 * Prepare a Ceph cluster for testing
 * Exdend the datacenter definition with Ceph Credentials
 * Add Ceph test cluster admin credentials to Kubermatic VSphere datacenter
 * Write the `ceph-credentials-controller` as a part of `kubermatic-controller-manager`
 * Add e2e test runner that will:
   * create a cluster on dev in VSphere datacenter
   * check for the StorageClass
   * test creating & mounting a volume
   * destroy the volume
   * destroy the cluster
