# etcd Backup

**Author**: Henrik Schmidt

**Status**: Proposal

Since migrating from the etcd-operator to a etcd StatefulSet, we need to provide an automatic backup solution for etcd clusters.
The etcd-operator had integrated backup&recovery mechanisms which we now need to replace.
Goal of this proposal is to have a fairly dynamic backup solution which also provides good monitoring/alerting.
Recovery will be a documented, manual process. As we now use a StatefulSet with PVC's the case when a restore is needed should be less than before.

## Motivation and Background

The old way via the etcd operator had several downsides:

* etcd's were storing to `emptyDir`
* multiple etcd's on the same host caused very high disk-io. Having one noisy etcd affects all others
* high frequency of restore operations when a host got replaced. Resulted in data loss
* direct code dependency to the etcd-operator crd
* etcd-operator had issues with recovering when encountering a concurrent write to the crd. Resulting in the etcd-operator ending to operate

Since the move to the StatefulSet we already benefit from

* etcd's are more resilient thanks to PVC's
* noisy neighbors do not affect other's (PVC's + resource limits are now in place)
* Less external dependencies (no copied etcd crd anymore)

## Implementation

### Basic
For each cluster a Kubernetes CronJob will be created.
This Job will:
- create a snapshot of the etcd cluster
- Store snapshot to target (Default: S3)
- Cleanup old snapshots (Default: keep last 20 revisions)

### Creation of CronJob
As this feature is not planned to be open sourced, we'll write a simple controller which will listen on Cluster resources and create/update CronJobs for backups.
Each cluster will get a own CronJob.
The controller will be running inside our controller-manager.
The CronJob will be managed in a `reconciling` manner, meaning it should always be checked if it needs to be updated.

The Job which should be executed needs to come from a template file.
As we need to be able to let admins develop a own backup-solution.

### Job
The job should consist of 2 containers. Both share a emptyDir volume:
- `init-container`
  - Creates the snapshot
  - Stores the snapshot to the shared volume
- `store-container`
  - Takes snapshot from shared volume
  - Stores snapshot to S3
  - Cleans up old snapshots (Defined by `revision`-flag)

An admin should be able to replace the `store-container` with any custom container.

### Cleanup of backups after cluster deletion
The backup controller will register a finalizer on the cluster to be able to cleanup after a cluster has been deleted.
When a cluster gets deleted a Job will be created from a admin defined template to delete the backups for the given cluster.

### Store & Cleanup template
Both, the store & cleanup container can be specified by the admin.
To inject secrets into the containers the admin can use Environment variables:

```yaml
command:
- /bin/true
image: docker.io/busybox
name: store-container
env:
- name: SECRET_USERNAME
  valueFrom:
    secretKeyRef:
      name: mysecret
      key: username
- name: SECRET_PASSWORD
  valueFrom:
    secretKeyRef:
      name: mysecret
      key: password

volumeMounts:
- name: etcd-backup
  mountPath: /backup
```

****

## Task & effort:
* Implement BackupCronJobController - 1d
* Write `init-container` - 2h
* Write example tool to store to S3 & clean up old revisions - 1d
* Add Minio chart to have a in-cluster S3 solution
* Add chart for a generic S3 metrics exporter
* Define alerting rules for the generic S3 exporter
