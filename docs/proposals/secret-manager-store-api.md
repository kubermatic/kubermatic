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

The new approach documented within this proposal supports the ability to hard-code authorized keys but rather, alongside another option to trust SSH CA public keys.

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
 - Restart the SSH daemon to enable configuration on the cluster
 - Distribute the public key to the workers

> NOTE: This allows the rollout of new CA key pairs for security reasons e.g rotation, revocation etc
> Management of the CA keys are the responsibility of the admin managing the KKP cluster.

Since the signing public key is to be replicated across multiple instances, we'd apply the same approach used previously for the User SSH keys.

The keys would be updated but would require an update to the SSHd config written by the machine-controller on cluster creation. 

Only the master-controller-manager would write these configurations.

Distribution of the keys is out of scope from the master to user controller, the secret for the CA would be created on the cluster.

During a revocation / rotation, the user would update the vault-ca secret configuration and that would be reconciled and updated in the clusters trusted ca config in ssh config at `/etc/ssh/TrustedCA.pem` via a hostPath mount point.


## Alternatives considered

No alternatives as this process solves all desired requirements.


## Task & effort:
*Specify the tasks and the effort in days (samples unit 0.5days) e.g.*
* Define proposal and outline scope of the vault manager implementation - 3d
* Test POC of the process documented in the proposal - 3d
* Review changes and update process as desired - 1d
* Add SSH configuration to machine-controller cloud-init template - 0.1d