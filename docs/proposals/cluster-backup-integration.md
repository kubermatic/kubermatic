# Cluster Backup Integration
**Author**: @ahmadhamzh @moelsayed

**Status**: Draft proposal

## Goals
Provide an automated backup solution for KKP user clusters, based on [velero](https://velero.io/). The velero integration solution should initially support the following use cases:
- Specific user cluster applications and/or namespaces backup and recovery, including volumes.
- User cluster disaster recovery (e.g. restore accidentally deleted clusters).
- User cluster migration to different KKP deployment, DC or seed.

In addition to this, the following basic features should be supported:
- Define backup destinations per provider.
- Select backup destination for backup configuration.
- Define scheduled and one time backups.
- Delete backups/backup configuration
- Define automatic backup expiration/rotation.
- Provide sufficient UI interface elements to support these requirements.

The initial implementation will target support for the Openstack Provider.

## Non-Goals
- Implement a cluster backup solution from scratch.
- Surface all velero features and capabilities through the UI.
- Support for community maintained cloud providers is out of scope.
- Managing and provisioning storage providers configuration.
- Backup and restore hooks.
- Backup and restore of Seed cluster objects are out of scope.
- Validation of restored resources is out of scope. 

 ## Motivation and Background
Our current backup implementation is focused on user cluster etcd backup only. It's designed as a recovery mechanism to improve the reliability of the user cluster control plane. This doesn't allow user to fine tune their backup scope to specific applications, namespaces or labels. It doesn't backup user cluster application data (e.g. volumes)

Additionally, the etcd backups are tightly coupled to the user cluster life-cycle. It doesn't support full cluster restores and/or cluster migration.

With velero integration, user will be able to create more granular backup configuration with focus on application level data and with persistent volume backup support. It will also allow for disaster recovery and migration scenarios that are not currently supported by the existing solution.

## Implementation

Initially, the integration will be modeled on the UI and workflow of the current etcdbackup feature.

The integration will be enabled by default for newly provisioned clusters only. For existing user cluster, it must be manually enabled. 

No default backup configuration will be created for newly created clusters. Admins or project/cluster owners will be required to create backup configuration and point them to a seed level backup destination.


### Cluster Migration
Velero doesn't provide actual cluster backup or migration functionality. Meaning, it doesn't support rebuilding a full cluster with it's resources from backup. Instead, it provides a different model; it allows users to migrate _all resources_ from one existing cluster to another, by restoring all resources from backup on the new destination cluster. 

The same model will be using to implement this functionality in KKP. For full cluster restore, the following scenarios can be applied:

- During configuring a restore request, the user will have the option to select an existing cluster as a restore target. The default value will be the source user cluster. It's only possible to restore to user clusters with Cluster Backup enabled. The user can create a fresh cluster ahead of the restore configuration and select it as a restore target.

- During the cluster creation process, an option will be provided to select a backup to restore. In this case, an initial restore request will be created as a step of the cluster provisioning process. 

### UI/Rest API

 **Admin Setting**

 The backup feature can be enabled or disabled by the admin. In the admin settings under the `Interface => Defaults` section, there is an option to enable or disable the feature.

 Admin can add the destination for the backup storage from the admin settings under the `Manage Resources => Cluster Backup Destination` section.
 
 Admin can add different BackupStorageLocation and VolumeSnapshotLocation values on multi providers(first,only Openstack)

**Cluster**

 In case of enable the feature from the admin settings, within the cluster creation wizard, under the `Cluster'` step, users can activate the backup feature for the created cluster.

 Backups can only be created for clusters with the backup feature enabled. If the feature is not enabled for a cluster, users can enable it from the `Edit Cluster` dialog."

**Project View**

 on the side nave bar there will be a section for `Cluster Backups`, under this section there will be two sub navigate item `Backups` and `Restore` to navigate to Backups/Restores pages.

**Backups Page**

 In the backup page, users can see the list of backups for all user clusters belonging to the current project. Depending on the user's role, users can create new backups, delete existing backups, and restore cluster resources (namespaces) from specific backups.

 From the backup page, users with the appropriate role can perform single or batch backup deletions with custom filtering and selection.

**Restore Page**

 In the restore page, users can view all the restored objects created from a backup and their details(backup name, cluster ID, resources, ...).

**Create Backup**

 From the backup page users with appropriate role can click on `Add Backup` button to add new backup.

 Backup dialog fields:
  
  * Backup Name
  * cluster (only clusters with backup feature enabled can be chosen)
  * Destination (from the list of destination that been add in `Admin Settings => Manage Resources => Cluster Backup Destination` )
  * namespaces 
  * select ondemand backup or scheduled backup
  * expired date
  * labels (optional)

**Restore Backup**

 From the backup list in the backup page users with appropriate role can restore resources from an existing backup.

 Restore dialog fields:
  
  * Restore Name
  * cluster (the cluster that i want to create the resources (namespaces) on it)
  * namespaces (select from the namespaces in the backup or all of them)


### API

The API will include any KKP integration parameters needed, along with references to any KKP specific objects (e.g. provider presets).

Additionally, those objects will wrap velero native objects. This pattern gives the users the ability to customize the backup/restore/destination configuration further then exposed via the UI, while simplifying the the implementation process, object reconciliation  and reduced the effort to maintain the future updates from velero side. 


The integration will provide the user with the following types:
- **CLusterBackup**
```yaml
TBD
```
This type will wrap the `Schedule` velero type. Additionally, it will specify the source cluster from which the backup is created. th The status of this type will reflect the last executed backup status and the last successful backup status.

- **ClusterRestore**
```yaml
TBD
```
This type will wrap the `Restore` velero type. Additionally, it will specify which cluster to apply the restore resources on. This will enable the cluster migration feature. The status for this object should reflect the current status of the velero restore running on the user cluster.

- **CLusterBackupDestination**
```yaml
TBD
```
This type will wrap both the `BackupStorageLocation` and `VolumeSnapshotLocation` velero types. Additionally, it will reference presets and related provider configuration from KKP to include in the velero types.


### Controller(s)

#### ClusterBackup Controller
The  ClusterBackup controller will be deployed on the seed cluster in the user cluster namespace. It will watch the ClusterBackup API types on the seed cluster and apply them to the user cluster API. 

The objects will be applied in a top-down fashion. The ClusterBackup will make sure only the changes applied via the KKP API is kept on the user cluster. Changes to the velero CRs applied by the user on the cluster API side will be reverted to maintain consistency.

Velero will be deployed on the seed cluster in the user cluster namespace. This is provides additional resilience to the setup, since the backups will not be affected by the state of the user cluster (e.g. limited capacity, issues with the nodes) 

**Note:** We need to investigate the feasibility of using this with other providers. Technically it should work with all providers that support CSI snapshots, since we will be connecting to the provider API from the same plane as the kubernetes API. However, we should confirm this.

The controller will also watch backups on the user cluster and update respective KKP CRs with the their latest status update.




## Task & effort:

tracked [here](https://github.com/kubermatic/kubermatic/issues/12646)