# ETCD Backup/Restore Workflow Controllers
**Author**: Olaf Klischat(@multi-io)

**Status**: Discussion.

## Goals
Kubermatic regularly creates backups of the etcd storage of all user clusters automatically. Restoring from
those backups is a relatively tedious and manual, albeit
[documented](https://docs.kubermatic.com/kubermatic/v2.14/operation/etcd/restoring-from-backup/), process.

We want to provide direct support in the Kubermatic seed controller manager for etcd cluster backup and
restore in a safe, secure and predictable manner. In addition, we also want to provide more complex
operations on top of this -- specifically, there should be a way to safely backup, delete and recreate an
etcd cluster in order to migrate it to a different storage class or change other attributes that can't be
changed in an existing StafulSet.

## Implementation
Preliminary proposal is that the user or a high-level "workflow" controller (see below) would issue
"commands" like "shut down API server", "start backup", "delete etcd statefulset", "start restore",
which the cluster controller and possibly a specific backup controller (very much different from the
current one) would execute.

"Issuing" a command can work by setting a special field in the cluster
resource -- proposal `cluster.spec.etcd.command` -- to the command name. This solution feels not
very "K8s-native", and it gives the user direct control over sending these low-level commands, which
might complicate things. We may alternatively store this in `cluster.status.etcd.command` instead,
so only other controllers can set it.

Here's a first proposal for a set of commands and associated state transitions that can
be used as building blocks for implementing common backup, restore and migration workflows:

https://drive.google.com/file/d/18aWP6Q1uVlm0XuMlNo9Aau1zryWzJR20/view?usp=sharing

Remarks, possible improvements and open questions:

- current state (black boxes in the graph) stored in `cluster.status.etcd.lifecycleStatus`

- we want to automate whole workflows like etcd cluster migrations. See "Workflow Controllers" below

- the explicit "apiDown" state is necessary during migrations because we want to ensure that the API
  is down and thus no further modification of the etcd data can occur before we start the backup

- we should be able to reduce the number of states. I.e. get rid of
  the explicit intermediate
  "...Stopping/...Starting/...Deleting,...Restoring" states. Instead,
  those intermediate states could be inferred from the state of the
  etcd cluster itself.

  For example, if the state is "running" and a stopApi command is
  issued, the controller would just set the apiserver replica count to
  0 and the state to "apiDown" immediately (rather than to
  "apiStopping"). Aftwards, the controller would check on every
  reconciliation loop whether there are actually no apiserver pods
  running anymore, and only then it would accept startBackup and
  deleteEtcd commands. So we'd get rid of the explicit "apiStopping"
  state, which basically just means "some apiserver pods still exist".

- the attributes of the etcd cluster like replica count and storage class are
  stored in `cluster.spec.etcd.settings.*`. Some attributes, especially the
  replica count, can be changed without recreating the statefulset. Other
  attributes like the storage class will only be used when
  creating the statefulset anew from backup during a restore.

- `startBackup` probably shouldn't be a command and shouldn't involve a
  state transition. Instead, it should be a simple library function that a workflow
  controller (see below) would invoke at the appropriate time.

## Workflow Controllers

These implement workflows like "migrate etcd to newStorageClass" by issuing
a series of commands like stopApi, startBackup, deleteEtcd, startRestore,
startApi. Proposal is to have a new CRD for each type of workflow. So for migrations a new resource
of type "EtcdMigration" containing the desired new etcd settings in its `.spec` would be created
in the cluster-xxx namespace. The etcd migration workflow controller would be triggered
by this, and it would follow these steps:

- store the name of the resource in the cluster resource (`cluster.status.etcd.currentWorkflow`)
  if that's not set already. This mutual exclusion ensures that only one workflow operates on a
  cluster at the same time.

- issue the stopApi command to the cluster

- once the api is down (i.e. `cluster.status.etcd.lifecycleStatus == "apiDown"`),
  generate a random backup name and store it in the EtcdMigration's `.status.backupName`

- initiate a backup with that name

- once the backup finished successfully, issue deleteEtcd command

- once the etcd statefulset is gone (lifecycleStatus == "etcdDeleted"), copy
  the etcd settings from the EtcdMigration resource to `cluster.spec.etcd.settings` and
  issue a startRestore command from the backup named in the EtcdMigration's `.status.backupName`

- once the restore is done, issue a startApi command to bring the API up again

- clear the resource name from `cluster.status.etcd.currentWorkflow` again

Other workflows that we might want are explicit backup and restore, with the
corresponding CRDs named `EtcdBackup` and `EtcdRestore`, respectively. Both would probably
just use the resource's `.metadata.name` as the backup name. The current backup cronjob could be
changed to simply create an `EtcdBackup` resource to initiate a backup, wait for it to
finish, and delete the resource.

## Questions

- error handling isn't well-defined. W.g. what if the backup fails during a migration? Do we automatically
  transition back to the running state, and what if that fails?