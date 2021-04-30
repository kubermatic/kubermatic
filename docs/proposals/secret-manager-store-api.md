# Secrets Manager Store Proposal

**Author**: Emmanuel Bakare (@emmanuel-kubermatic)

**Status**: Draft proposal; prototype in progress.

## Goals
The proposed fix for the issue [6671](https://github.com/kubermatic/kubermatic/issues/6671) 

Adds support for vault related read access within the kubermatic source

## Non-Goals
Adds support for vault access and a central store for accessing data

## Motivation and Background

Currently, there's a user ssh agent which manages credentials via fsnotify events and kubernetes secrets.

This approach is great but adds a limitation where the end user has to manually copy credentials to ssh secrets or add them via the API.

Despite the ease of using that, there are some limitations:
1. Duplicate credentials across their environments.
2. We can have other secret managers in the future asides Vault 

The approach is to simplify the use / integration of separate secret manager APIs through easy interfaces.

Also, there could be more than 1 secrets manager in the future
e.g AWS Secrets Manager, Google Secret Manager etc

## Implementation

Define structs and interfaces for other secret managers

we would have interfaces like such

```go
package sm

type Store interface {
    write() error
    read(key string) error
    init(config interface{}) error
}

type K8SSecretStore struct{
	Store
	// other configs
}
```

The approach would this can affect the way we reconcile secrets, even  on Kubernetes as API access to the secrets store could be modified to use those interfaces.

A similar approach can be used already in here [user-ssh-agent](./pkg/controller/usersshkeysagent/usersshkeys_controller.go).

## Alternatives considered

For the case of vault, we can consider implementing the interface and use the [Vault Injector](https://www.vaultproject.io/docs/platform/k8s/injector) 
to manage secrets from vault as secrets.

This does limit the case of an on-prem vault installation as it has a dependence on a [vault-k8s](https://www.vaultproject.io/docs/platform/k8s/injector) deployment.
```text
This functionality is provided by the vault-k8s project and can be automatically installed and configured using the Vault Helm chart.```
```

## Task & effort:
*Specify the tasks and the effort in days (samples unit 0.5days) e.g.*
* Implement Store Interface - 0.1d
* Write Store APIs for Kubernetes Secrets Store - 1d
* Implement Vault integration APIs (Using the agent injector might have limitations) - 2d