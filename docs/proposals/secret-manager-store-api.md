# Secrets Manager Store Proposal

**Author**: Emmanuel Bakare (@emmanuel-kubermatic)

**Status**: Draft proposal; prototype in progress.

## Goals
The proposed fix for the issue [6671](https://github.com/kubermatic/kubermatic/issues/6671) 

- Define secret using Vault signed certificates to assist with SSH access

## Non-Goals


## Motivation and Background

Currently, there's a user ssh agent which manages credentials via fsnotify events and kubernetes secrets.

This approach is great but adds a limitation where the end user has to manually copy credentials to ssh secrets or add them via the API.

The new approach documented within this proposal removes the ability to hard-code authorized keys but rather, add another option to lease out ssh certificates via vault which can be used to access the necessary clusters.

This is possible via signed certificates which can be provided via vault with a lease time configurable by the user.


## Implementation

[Signed SSH Certificates](https://www.vaultproject.io/docs/secrets/ssh/signed-ssh-certificates) would be used to implement this.

Once provided, the configuration along with Vault access credentials (Vault address and token with access to that entry) would be used to create the desired token and access the cluster as needed.

Since the signing public key is to be replicated across multiple instances, we'd apply the same approach used previously for the SSH keys.

The keys would be updated but would require an update to the SSHd config written by the machine-controller on cluster creation. 

Updates to the SSH configuration required are detailed below:
```text
.../etc/ssh/sshd_config
TrustedUserCAKeys /etc/ssh/trusted-user-ca-keys.pem

# Needed for OpenSSH Server 8.1 and older, ssh-rsa is removed from the default supported CA algorithms
CASignatureAlgorithms ecdsa-sha2-nistp256,ecdsa-sha2-nistp384,ecdsa-sha2-nistp521,ssh-ed25519,rsa-sha2-512,rsa-sha2-256,ssh-rsa
```

Only the master-controller-manager would write these configurations.

During a revocation / rotation, the user would update the vault-ca secret configuration and that would be reconciled and updated in the cluster via a hostPath mount point.


## Alternatives considered

No alternatives as this process solves all desired requirements.


## Task & effort:
*Specify the tasks and the effort in days (samples unit 0.5days) e.g.*
* Define proposal and outline scope of the vault manager implementation - 3d
* Test POC of the process documented in the proposal - 3d
* Review changes and update process as desired - 1d
* Add SSH configuration to machine-controller cloud-init template - 0.1d