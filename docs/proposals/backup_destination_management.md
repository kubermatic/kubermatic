**ETCD Backup Destination Management**

**Author**: Lovro Sviben

**Status**: Draft proposal. Prototype in progress

## Goals

Allow managing multiple etcd backup destinations per seed.

## Non-Goals

This does not include supporting other providers, our backups can still only be used with S3-compatible clients.

## Motivation and Background

Since 2.18, we support automatic etcd backups and restores to an S3-compatible bucket. The backup are enabled and its
destination are set in the Seed Object. The destination consists of a bucket name, and an endpoint, paired with a 
corresponding secret which contains the credentials. So all backups for Cluster within this seed end up in one bucket.

The goal of this feature is to provide an option to:
- allow admins to set up multiple destinations per Seed (with their credentials)
- When creating Etcd Backup Configs, Cluster admins can choose destination. based on the Cluster Seed

## Implementation

### Seed extension
Seed has the following Backup data:

```go
// SeedBackupRestoreConfiguration are s3 settings used for backups and restores of user cluster etcds.
type SeedBackupRestoreConfiguration struct {
	// S3Endpoint is the S3 API endpoint to use for backup and restore. Defaults to s3.amazonaws.com.
	S3Endpoint string `json:"s3Endpoint,omitempty"`
	// S3BucketName is the S3 bucket name to use for backup and restore.
	S3BucketName string `json:"s3BucketName,omitempty"`
}
```

As we want to avoid breaking compatibility with the older version, we can deprecate these fields and in addition add a 
map of BackupDestinations. When giving users options for backup destinations, add the old ones as well for now.

```go
// SeedBackupRestoreConfiguration are s3 settings used for backups and restores of user cluster etcds.
type SeedBackupRestoreConfiguration struct {
	// S3Endpoint is the S3 API endpoint to use for backup and restore. Defaults to s3.amazonaws.com.
	// Deprecated: - use the BackupDestinations for configuring backup destination
	S3Endpoint string `json:"s3Endpoint,omitempty"`
	// S3BucketName is the S3 bucket name to use for backup and restore. 
	// Deprecated: - use the BackupDestinations for configuring backup destination
	S3BucketName string `json:"s3BucketName,omitempty"`
	
	// BackupDestinations contains possible etcd backup destinations for this Seed 
	BackupDestinations map[string]*BackupDestination `json:"backupDestinations,omitempty"`
}

// BackupDestination holds the bucket and endpoint info for an etcd backup destination
type BackupDestinations struct {
	// BucketName is the name of the bucket in which the etcd backups will be stored
	BucketName string `json:"bucketName,omitempty"`
	// Endpoint is the endpoint through which to communicate with the storage provider
	Endpoint string `json:"endpoint,omitempty"`
}

```

### EtcdBackupConfig extension

The EtcdBackupConfig also needs to be extended with the backup destination instead of just using the Seed one. 

```go
// EtcdBackupConfigSpec specifies details of an etcd backup
type EtcdBackupConfigSpec struct {
	// Name defines the name of the backup
	// The name of the backup file in S3 will be <cluster>-<backup name>
	// If a schedule is set (see below), -<timestamp> will be appended.
	Name string `json:"name"`
	// Cluster is the reference to the cluster whose etcd will be backed up
	Cluster corev1.ObjectReference `json:"cluster"`
	// Schedule is a cron expression defining when to perform
	// the backup. If not set, the backup is performed exactly
	// once, immediately.
	Schedule string `json:"schedule,omitempty"`
	// Keep is the number of backups to keep around before deleting the oldest one
	// If not set, defaults to DefaultKeptBackupsCount. Only used if Schedule is set.
	Keep *int `json:"keep,omitempty"`
	// Destination defines where the backup will be stored, and restored from if it`s used in a restore
	Destination *BackupDestination `json:"destination,omitempty"`
}
```

To keep backwards compatibility, as old EtcdBackupConfigs won't have the backup destination set, we can either do a migration,
in which we set the backup destination as the current Seed one. 

Or, if its empty, we can use the old destinations from the Seed. When a destination is used in the backup controller, 
we can fill up this field. As the Backups are scheduled, in time all will have the field set. For new ones we can enforce
setting the destination.

### Credentials

For the credentials, we need to allow setting credentials for each destination. For the credentials secret, we can either
use the current secret, and prefix the keys with the destination name. Or we can create a secret for each destination.

### Controllers
Furthermore, we will need to update the backup and restore controllers to support the destinations, but that should be 
fairly simple.

## Constraints

* If an admin removes/modifies a destination, it doesn't affect already existing EtcdBackupConfigs, unless it also changes its secret. 
This could make the backups fail.
* Changing a backup destination for an etcd backup config should not be allowed, as it could break restoring from previously made backups. 
Best to delete and recreate.

## Tasks

* Extend Seed with backup destinations
* Extend EtcdBackupConfig with destination
* Update backup controller with destination
* Update restore controller with destination
* API for destination management
