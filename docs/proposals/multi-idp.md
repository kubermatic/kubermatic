# Multi-IDP for share cluster

Multi-tenancy with third party identity providers for kubermatic managed clusters.

## Requirements

- [https://docs.kubermatic.io/advanced/oidc_auth/]()
- Multi-Tenant setup

## Case

When using kubermatic to provide a multi-customer environment with clusters
they usually do have their own IDP that should be connected to kubernetes. 

As of now the only two  authentication options in kubermatic managed clusters are:

- admin token
- central idp / oidc provider

These two options are great for single-tenant installations but provide additional
risk and complexity in multi-tenant environments. 

As of now every user in kubermatic would be able to authenticate against any cluster
if endpoints are known and a custom kubeconfig is created. They would not be able to
do things as auth-z is not configured for the user.

From a security standpoint this works - but as soon as customer start working with groups
there is intermediate to high risk of people authorizing people from unrelated companies.
Group names like `administrator` is just to common in current IT. 

## Proposal

When creating a cluster users are able to also provide endpoints, secrets and
attribute information. This will then be configured in the cluster. 

Additionally this will be used for the share cluster URL the oidc auth functionality
already provides.

This way people have the option to bring their own IDP, hosting companies are
able of providing multiple authentication realms for different corporate customers
but still keeping setups maintainable and in an unpaused state.

## Implementation

OIDC providers already can be configured for customer clusters in kubermatic.
This is just to be used as share cluster is already doing.

Share cluster functionality needs to be extended to allow kubermatic users to set
custom oidc provider settings.

Share cluster functionality needs to be extended to authenticate against third party IDP.

Share cluster functionality needs to be extended to allow setting properties like username
and group fields from the to be retrieved token.
