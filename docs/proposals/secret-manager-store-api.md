# Secrets Store Proposal

**Author**: Emmanuel Bakare (@emmanuel-kubermatic)

**Status**: Draft proposal; prototype in progress.

## Goals
The proposed fix for the issue [6671](https://github.com/kubermatic/kubermatic/issues/6671) 

- Define secret using Vault signed certificates to assist with SSH access

## Non-Goals


## Motivation and Background

A kubermatic user can assign ssh keys to clusters at creation time via the machine controller through cloud-init, but the user ssh agent also assists with updating the keys at runtime via the usersshkeys secret.

This approach is great but adds a limitation where the end user has to manually copy credentials to kubermatic usersshkey secrets or added via the API for each cluster's worker nodes authorized_keys.

The new approach documented within this proposal provides the ability to use [SSH CA certificates](https://www.vaultproject.io/docs/secrets/ssh/signed-ssh-certificates), avoiding the burden of copying each user key individually.

This enables authentication via signed keys which can be provided via vault with a lease time configurable by the user.

Vault provides the facility for storing and signing the keys used for authentication.


## Implementation

[Signed SSH Certificates](https://www.vaultproject.io/docs/secrets/ssh/signed-ssh-certificates) would be used to implement this.

This includes the following process
 - Generating a CA key pair 
 - Configure the SSH daemon to trust the public key via TrustedUserCAKeys configuration, example:
    ```text
    .../etc/ssh/sshd_config
    TrustedUserCAKeys /etc/ssh/trusted-user-ca-keys.pem
    
    # Needed for OpenSSH Server 8.1 and older, ssh-rsa is removed from the default supported CA algorithms
    CASignatureAlgorithms ecdsa-sha2-nistp256,ecdsa-sha2-nistp384,ecdsa-sha2-nistp521,ssh-ed25519,rsa-sha2-512,rsa-sha2-256,ssh-rsa
    ```
 - Starts the SSH daemon with the above configuration on the cluster to use only keys signed by the stated TrustedCA.
 - Distribute the public key to the workers upon further updates with the help of the `user-ssh-agent`.

> NOTE: This allows the rollout of new CA key pairs for security reasons e.g rotation, revocation etc
> Management of the CA keys are the responsibility of the admin managing the KKP cluster.

The machine controller would create the CA pem file and configure the SSH daemon to add the `TrustedCAKeys` configuration, provided the CA key is specified on cluster creation.

Distribution of the keys is out of scope from the master to user controller, the secret for the CA would be created on the cluster. The reason for this is to remove the issues around the secret being created after cluster creation.

Since the updates to use a TrustedCA require we update the SSH configuration and restart the SSH daemon, this would not be possible if the cluster is already created without the configuration.

During a revocation / rotation, the user would update the `vault-ca` secret configuration and that would be reconciled and updated in the clusters trusted ca config in ssh config at `/etc/ssh/TrustedCA.pem` via a hostPath mount point.

The update would now be handled by the `user-ssh-key-agent` controller, instead of the machine controller and reconciled on any changes to the `vault-ca` secret in the `kube-system` namespace.


## Alternatives considered

No alternatives as this process solves all desired requirements.


## Task & effort:
*Specify the tasks and the effort in days (samples unit 0.5days) e.g.*
* Define proposal and outline scope of the vault manager implementation - 3d
* Test POC of the process documented in the proposal - 3d
* Review changes and update process as desired - 1d
* Add SSH configuration to machine-controller cloud-init template - 0.1d
