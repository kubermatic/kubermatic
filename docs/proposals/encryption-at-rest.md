# Encryption at Rest in etcd

**Author**: Marvin Beckers (@embik)

**Status**: Implementation in progress

## Goals

This proposal has the following goals. Also check [Motivation and Background](#motivation-and-background).

* Offer an optional configuration to enable etcd encryption at rest. Also allow "going back" to unencrypted data
* For cloud providers that provide a KMS integration for encryption at rest, support that KMS plugin as a turn-key solution
* Support a secure "static key" encryption provider for environments that do not have a KMS plugin or for users that do not want to use KMS
* Allow users to rotate their encryption key or KMS key reference (internally, this requires changing the encryption configuration at least two times, restarting the apiserver, and forcing re-encryption of all data)
* Provide a mechanism to provide the right encryption key during an etcd restore from backup (otherwise, the data is unreadable)

## Non-Goals

* Support KMS plugins across cloud providers (so, for example, support AWS KMS on Azure, or the other way around)
* Store corresponding encryption key alongside etcd backup (_information_ about the right key might be part of the backup to ease restores though)
* Automatically rotate static encryption keys (this might come as a follow-up feature, but the initial implementation should focus on getting the rotation mechanism right)

## Motivation and Background

etcd is a distributed key-value store that is used by the Kubernetes API as data storage. By default, data in etcd is not encrypted at rest. KKP only encrypts etcd data in transit right now. Kubernetes provides the ability to configure a pluggable encryption mechanism that allows encrypting arbitrary resources (usually, `Secrets`) when stored in etcd. It supports a couple of "static" encryption schemes where a key is provided in the encryption configuration (secretbox, aesgcm, aescbc) and integration with an external KMS system via a plugin mechanism.

Encrypting data in etcd for sensitive information like secret data is recommended by security benchmarks and best practices. It further improves the security of our etcd backup feature, as an attacker that gets hold of a backup archive cannot extract `Secrets` content from it.

KKP users might want to encrypt their data at rest in user clusters to improve their security posture and/or fulfill regulatory requirements or prepare their environments for audits.

## Implementation

Overall, the implementation of this proposal will follow the [official documentation](https://kubernetes.io/docs/tasks/administer-cluster/encrypt-data/). They key idea for KKP is to ease the configuration and migration processes to provide a turn-key solution that is safe and auditable.

In general, the `EncryptionConfiguration` will be provided as a `Secret` that is mounted in kube-apiserver Pods. The `Secret` will be updated by the `seed-controller-manager` based on changes to the `Cluster` resource.

### KMS Plugins

KMS plugins expose a unix socket for communication between kube-apiserver and the plugin. It needs to be available to every instance of kube-apiserver. As such, it should run as a sidecar to the kube-apiserver, if the KMS provider for encryption is enabled. The socket will be mapped to an emptyDir shared between the main apiserver container and the KMS plugin container.

During a migration, both KMS plugin containers (old and new configuration) need to be part of the Pod.

### Static Encryption Provider

The encryption at rest feature needs to support a static key encryption provider to support environments without a KMS service. [Hashicorp Vault](https://www.vaultproject.io) would be a great solution for such environments, but right now (November 2021) no KMS plugin for Hashicorp Vault exists. Therefore, static key encryption should be an option. As per [the official upstream documentation](https://kubernetes.io/docs/tasks/administer-cluster/encrypt-data/#providers), there are three providers: `secretbox`, `aesgcm` and `aescbc`. The latter two are considered weak as they are vulnerable or require automatic key rotation. Therefore, KKP should support `secretbox` as static encryption provider.

### Key Rotation

Key rotation is a necessary feature to support in the initial version of encryption at rest, as many users will have policies or requirements around rotating encryption keys on a regular basis. As stated in the non-goals, this initial version of the release will try to avoid automatic key rotation as much as possible and rely on users to rotate their keys. The process to rotate the encryption keys is described [in the documentation](https://kubernetes.io/docs/tasks/administer-cluster/encrypt-data/#rotating-a-decryption-key).

KKP will provide the ability to provide multiple encryption keys, mirroring the configuration for `kube-apiserver`. Rotating keys from a user perspective is therefore a three-step process:

1. Add a secondary encryption key to the `cluster.spec.encryptionConfiguration` structure.
2. When the secondary key is rolled out (visible via `Cluster` status), switch key positions.
3. When the key rotation is rolled out, remove the old key.

This approach will reduce complexity in the KKP implementation as it does not need to internally handle old keys that no longer exist in the `Cluster` spec, while giving end users maximum flexibility over the process. These steps will be documented in KKP's end user documentation appropriately.

Internally, KKP needs to track what phase of the key rotation (as per the upstream documentation linked above) a cluster is in. For that, it will use a status field (see later section [ClusterStatus API changes](#clusterstatus-api-changes). A dedicated `encryption_controller` will handle updates to the encryption phase and necessary actions for a specific phase.

As for necessary actions, apart from moving through phases, the `encryption_controller` will be spawning a Kubernetes `Job` that re-encrypts the encrypted resources with the updated key when necessary. The `Job` resource will make sure that data re-encryption is run only once and will not be interrupted by a restart of the `seed-controller-manager` (if the `Job` is interrupted, it should be restarted, but this is tracked as part of a `Job`'s typical behaviour).

During key rotation, certain changes to the `Cluster` spec should be prohibited to prevent resources encrypted with an unavailable key (due to rotating keys before re-encryption finished, for example). This will be handled by the existing `Cluster` validation and will use the status fields introduced as part of this proposal.

A similar process can be applied to decrypt (disable encryption) the data when encryption at rest is disabled.

### ClusterSpec API Changes

KKP's `Cluster` spec should offer a new API field that covers encryption at rest. It could look like this:

```yaml
spec:
    encryptionConfiguration:
        enabled: true
        secretbox:
            keys:
            - name: secretbox-key-1
              value: K7iFzARMM/VNtAbSSOqMEpMT5fzIsl46m5/uRe5OZ6o= # generated via head -c 32 /dev/urandom | base64
              # OR
              secretRef: # reference a Secret object on the Seed cluster that holds the static key
                name: cluster-encryption-key
                key: key
        kms:
            aws:
                keys:
                - key: "arn:aws:kms:us-west-2:111122223333:key/1234abcd-12ab-34cd-56ef-1234567890ab"
                  region: us-west-2
            gcp:
                keys:
                - key: "projects/<PROJECT_ID>/locations/<LOCATION>/keyRings/<KMS_KEY_RING>/cryptoKeys/<KMS_KEY>"
            azure:
                keys:
                - keyVault: keyvault
                  key: kkp-encryption-key
                  version: 1
[...]
```

The snippet above includes all provider configurations for demonstration. They are mutually exclusive to each other (which should be validated via webhook validation). The `kms` fields for `aws`, `gcp` and `azure` can only be set if the cluster's `spec.cloud.aws`, `spec.cloud.gcp` or `spec.cloud.azure` are set respectively. A realistic configuration would therefore look like this:

```yaml
spec:
  encryptionConfiguration:
    enabled: true
    kms:
      aws:
        keys:
        - key: "arn:aws:kms:us-west-2:111122223333:key/1234abcd-12ab-34cd-56ef-1234567890ab"
          region: us-west-2
```

### ClusterStatus API Changes

To support the potentially long-running process described in [Key Rotation](#key-rotation), additional status information should be stored and exposed to make sure that `seed-controller-manager` can pick up key rotation in case of pod termination. The following fields could be added:

```yaml
status:
  encryption:
    activeKey: "<string identifier based on key identity, e.g. 'secretbox:secretbox-key-1'>"
    phase: "(Pending|Failed|Active|EncryptionNeeded)"
  conditions:
    - kubermatic_version: <version>
      lastHeartbeatTime: "2021-11-22T07:17:12Z"
      lastTransitionTime: null
      status: "True"
      type: EncryptionInitialised
```

`status.encryption.activeKey` holds a string identifier/representation (based on what kind of encryption scheme is used, e.g. secretbox keys will be prefixed by `secretbox:`) of the currently active encryption key.

`status.encryption.phase` represents the current phase of the encryption "loop". It's set by `encryption_controller` based on changes to the `Cluster` spec and the state of the `kube-apiserver` Pod (is the `Secret` holding the `EncryptionConfiguration` up-to-date? is it mounted to the `kube-apiserver` Pods? Are there still `kube-apiserver` Pods that use the old configuration?)

### etcd Backups

Special consideration needs to go to the etcd backup and restore functionality that KKP provides. Upon restore, the same encryption key used to encrypt the data **at the time of the backup** needs to be provided, so the kube-apiserver can decrypt data from the restored etcd state. It is out of scope for KKP to back up the encryption keys itself, but it seems beneficial to include a "key hint" that helps administrator choose the correct key from a backup of old encryption keys.

Documentation for the encryption at rest feature needs to highlight the necessity to have old encryption keys around and recommend users to back up their encryption keys out of band.

Therefore, the `status.lastBackups` list of `EtcdBackupConfig` objects should include a `keyHint` field. This field is not used programmatically, but should include information like the ARN for the AWS KMS key that was active at the time of the backup. For static keys, the secret reference can be provided, but might not be meaningful.

The UI for restoring an etcd backup should offer the option (if encryption at rest is enabled) to set up the cluster with an (older) encryption key before applying the restore.

## Threat Model

This section discusses various threats to KKP systems, which can be prevented or mitigated by encryption at rest, some of them requiring well-managed controls over a KKP system.

### Prevented or mitigated risks

#### Stolen etcd backups

Creating backups of etcd via the available KKP feature might be necessary to restore etcd rings for user clusters that were damaged. However, since an external system (S3 or any S3-compatible object store) is used to store backups, there is a risk of attackers gaining access to the backup location and downloading stored etcd backups. Since the backups include all resources exposed by the Kubernetes API, that would include `Secrets` and their contents, which are usually credentials or other confidential data that must not be available in plain-text. Attackers in possession of an etcd backup might be able to use credentials extracted from `Secrets` to attack other systems.

Encryption at rest mitigates this threat by encrypting the contents of `Secrets` (and potentially, other sensitive resource types) while stored in etcd. Backups of etcd only include the encrypted data, which attackers cannot decrypt if they only had access to the backup file. Only systems or people in possession of both the etcd backup and the encryption key are able to decrypt the sensitive data in it. It is therefore vital that access to the backup location and the encryption key is strictly separated by appropriate controls.

S3 security has been problematic in the past and several high-profile breaches[^1][^2][^3] have occurred in the last few years. The probability of this threat depends on the administrator's experience with cloud services like S3 when setting up the KKP etcd backup functionality and the S3 bucket for it.

[^1] https://businessinsights.bitdefender.com/worst-amazon-breaches
[^2] https://www.computerweekly.com/news/252491842/Leaky-AWS-S3-bucket-once-again-at-centre-of-data-breach
[^3] https://securityboulevard.com/2021/03/another-s3-bucket-leads-to-breach-of-50k-patient-records/

#### Stolen or lost disk storage

Especially in private datacenters, the risk of unauthorized or unintended removal of data disks from the datacenter exists. Disks can be either physically removed by attackers or bought online if they are sold to the highest bidder. If those disks happen to include an etcd data directory (likely nested into a virtual disk present on the physical disk), attackers can extract sensitive information from `Secret` resources that were part of the etcd data and potentially attack other systems with extracted credentials. This is a similar threat as [stolen etcd backups](#stolen-etcd-backups) and encryption at rest mitigates against it in the same way: Secret data in the etcd data directory will be encrypted, and access to a disk that stores this data directory will not allow an attacker to decrypt sensitive data and use it for further attacks.

It is also possible that disk data is included in backup mechanisms that are set up by the datacenter provider or by another IT function. Similar to gaining access to a physical disk, a disk backup can be abused by an attacker in the same way if etcd data is not encrypted at rest. It is possible that disk images themselves are encrypted, but this is not necessarily the case.

### Partially mitigated or unmitigated risks

#### User cluster compromise

Since the etcd data storage is separated from the user cluster, a compromise of the user cluster does not allow an attacker to gain access to the etcd data storage. However, if the attacker gained sufficient privileges with the Kubernetes API, they can request `Secrets` resources in a decrypted state. Encryption at rest only secures the data stored on disk, a high access level to the Kubernetes API will look like the attacker has the permission to access data in its unencrypted state.

#### Seed cluster compromise

Seed clusters host the two critical components of encryption at rest, the Kubernetes API server and the etcd ring. The etcd ring only holds the encrypted data and is not aware of the key to decrypt it. The encryption and decryption is happening in the Kubernetes API itself. Since the encryption configuration in this proposal will be mounted as a `Secret` reference into the `kube-apiserver` Pod, either shell access to the `kube-apiserver` Pod or read access to the `Secret` is necessary for an attacker to recover the encryption key.

Even access to the encryption configuration `Secret` can be partially mitigated by using a KMS provider as the encryption configuration will only include the KMS key reference, not the actual private key. With that being said, there is a high probability that an attacker can extract cloud provider credentials if they have enough permissions to get the KMS key reference.

## Alternatives considered

Since Kubernetes does not offer another mechanism for data encryption at rest, alternatives are sparse. Considerations are mainly within the scope of "implementing the encryption configuration":

* Developing a Hashicorp Vault KMS plugin: For KKP admins that do not want to rely on cloud provider KMS services, Hashicorp Vault might be an interesting alternative to static key encryption. Since no KMS plugin exists, developing one is a consideration to keep in mind, depending on admin interest. For now, it seems out of scope. We should support cloud provider KMS plugins first.
* Support encryption schemes `aesgcm` and `aescbc`:
  * `aesgcm`: Must be rotated every 200k writes, which might be hard to track. Since we do not plan to start with an automated key rotation scheme, this is discouraged by upstream.
  * `aescbc`: Considered weak as it's vulnerable to padding oracle attacks. Therefore not considered for inclusion.
* Re-implement [kubeone's encryption support](https://docs.kubermatic.com/kubeone/v1.3/guides/encryption_providers/): `kubeone` already supports the data encryption feature of Kubernetes, but only provides `aescbc` as a turn-key solution. KMS plugins need to be installed and configured by administrators manually. KKP however should provide them out of the box for a better "managed" experience.

## Tasks & Effort

Initial estimates (not necessarily in the given order):

* Implement API changes (`ClusterSpec`, `ClusterStatus`, `EtcdBackupConfigurationStatus`) | 0,5d
* Support `EncryptionConfiguration` as a ConfigMap that is generated based on new API and passed to kube-apiserver | 2d
* Add static encryption scheme `secretbox` to supported encryption configurations | 2d
* Implement loops that pick up both status and spec's configured key references, rotate keys and re-encrypt data | 5d
* Add sidecars to kube-apiserver for KMS plugins based on `ClusterSpec` and `ClusterStatus` values and include them in `EncryptionConfiguration` | 5d
* Add e2e test cases (enabling/disabling encryption config, key rotation, backup restores) | 3-4d
* Update documentation to include guidelines for encryption configuration | 2d
