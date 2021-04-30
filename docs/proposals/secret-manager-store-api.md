# Secrets Manager Store Proposal

**Author**: Emmanuel Bakare (@emmanuel-kubermatic)

**Status**: Draft proposal; prototype in progress.

## Goals
The proposed fix for the issue [6671](https://github.com/kubermatic/kubermatic/issues/6671) 

- Adds support for vault related read access within the kubermatic source
- Polling support for SSH and Secrets within the Kubermatic source
- Revocation and TTL-based access for SSH

## Non-Goals
Adds support for vault access and a central store for accessing data

## Motivation and Background

Currently, there's a user ssh agent which manages credentials via fsnotify events and kubernetes secrets.

This approach is great but adds a limitation where the end user has to manually copy credentials to ssh secrets or add them via the API.

The new approach documented within this proposal adds the ability to hard-code authorized keys but rather, add another option to lease out ssh certificates via vault which can be used to access the necessary clusters.

This is possible via signed certificates which can be provided via vault with a lease time configurable by the user.


## Implementation

[Signed SSH Certificates](https://www.vaultproject.io/docs/secrets/ssh/signed-ssh-certificates) would be used to implement this.

Once provided, the configuration along with Vault access credentials (Vault address and token with access to that entry) would be used to create the desired token and access the cluster as needed.

Since the signing public key is to be replicated across multiple instances, we'd apply the same approach used previously for the SSH keys.

The keys would be updated but an update to the SSHd config. 

```text
.../etc/ssh/sshd_config
TrustedUserCAKeys /etc/ssh/trusted-user-ca-keys.pem

# Needed for OpenSSH Server 8.1 and older, ssh-rsa is removed from the default supported CA algorithms
CASignatureAlgorithms ecdsa-sha2-nistp256,ecdsa-sha2-nistp384,ecdsa-sha2-nistp521,ssh-ed25519,rsa-sha2-512,rsa-sha2-256,ssh-rsa
```

Only the master-controller-manager would have access to the Vault instance deployed either within or outside the cluster and a PATH entry to the desired secret engine would be used to fetch it.

Once fetched, it would be used to create a [UserSSHKey](https://github.com/kubermatic/kubermatic/pkg/crd/kubermatic/v1/sshkeys.go) CRD along with the list of clusters it should be synced to and from there, replicated as needed.

During a revocation / rotation, the same process would be applied, and the previous CRD updated and replicated as before.


## Alternatives considered

No alternatives as this process solves all desired requirements.


## Task & effort:
*Specify the tasks and the effort in days (samples unit 0.5days) e.g.*
* Implement Store Interface - 0.1d
* Write Store APIs for Kubernetes Secrets Store - 1d
* Implement Vault integration APIs (Using the agent injector might have limitations) - 2d