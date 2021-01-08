# Service Account Token Volume Projection Support

**Authors**: Rastislav Szabo (@rastislavs), Jiacheng Xu (@jiachengxu)

**Status**: Draft proposal.

**Issue**: https://github.com/kubermatic/kubermatic/issues/6191

## Goals
Add support for [Service Account Token Volume Projection](https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/#service-account-token-volume-projection)
in user clusters of KKP. Allow enabling / disabling it and configuring its parameters on a per-user-cluster basis.

## Non-Goals
This proposal does not cover full [Bound Service Account Token Volume](https://kubernetes.io/docs/reference/access-authn-authz/service-accounts-admin/#bound-service-account-token-volume)
support, as it is still in Alpha stage. Nevertheless, it aims to prepare a foundation for it.

## Prerequisites
`TokenRequest` and `TokenRequestProjection` Kubernetes feature gates enabled (enabled by default since Kubernetes v1.11 and v1.12 respectively).

## Motivation and Background
In current versions of Kubernetes, service account tokens are stored in Secrets, which provides a broad attack surface
for Kubernetes control-plane when powerful components are run. Any components that can see the service account’s Secret
can be at least as powerful as the other components within the service account. The tokens are not audience nor time bound.
Also, in the current Kubernetes model, a Secret is required per service account, which can lead to scalability issues by
many service accounts.

The [Bound Service Account Tokens KEP](https://github.com/kubernetes/enhancements/blob/master/keps/sig-auth/1205-bound-service-account-tokens/README.md)
aims to address these issues. Although the Bound Service Account Token Volume support is still in Alpha stage, part of
the proposal called Service Account Token Volume Projection is already GA in v1.20 and used by some applications.

### Benefits to KKP Users
By enabling service account token volume projection, we can provide KKP users the opportunity to use third party tokens
with scoped audience and expiration for more security considerations.

For example, as mentioned in [Istio Security Best Practices](https://istio.io/latest/docs/ops/best-practices/security/),
as of the Istio version 1.3, Istio uses third party tokens by default.

Similarly, for configuring [Konnectivity Service](https://kubernetes.io/docs/tasks/extend-kubernetes/setup-konnectivity/#configure-the-konnectivity-service)
in Kubernetes cluster, service account token volume projection feature has to be enabled.


## Implementation
To enable service account token volume projection, the following flags have to be specified to the kube-apiserver:

 - `--service-account-key-file`: is already set by KKP to `/etc/kubernetes/service-account-key/sa.key`,
 - `--service-account-signing-key-file`: should be set to `/etc/kubernetes/service-account-key/sa.key` in KKP,
 - `--service-account-issuer`: should be configurable by the cluster administrator to cover specific use-cases,
   can be set to `kubernetes.default.svc` if not specified.
 - `--service-account-api-audiences`: should be configurable by the cluster administrator to cover specific use-cases,
   equals to `--service-account-issuer` if not specified.

Since we want to control this feature on a per-user-cluster basis, we propose to add the following configuration
options into the Kubermatic Cluster CRD spec:

```go
// ClusterSpec specifies the data for a new cluster.
type ClusterSpec struct {
...
    ServiceAccountSettings *ServiceAccountSettings `json:"serviceAccountSettings,omitempty"`
...
}
```

```go
type ServiceAccountSettings struct {
	TokenVolumeProjectionEnabled bool `json:"tokenVolumeProjectionEnabled,omitempty"`
	Issuer string                     `json:"issuer,omitempty"`
	APIAudiences []string             `json:"apiAudiences,omitempty"`
}
```

If the config is not specified in the Cluster spec, the service account token volume projection will be disabled.
If it is enabled and the `Issuer` is not specified, it will be set to the default value `kubernetes.default.svc`.

The KKP seed controller manager will set the flags of the kube-apiserver of the given cluster according to this configuration.

Note that once the full support for [Bound Service Account Tokens](https://github.com/kubernetes/enhancements/blob/master/keps/sig-auth/1205-bound-service-account-tokens/README.md)
is going to be added, we may just extend this structure with more fields matching the respective kube-apiserver flags.

### Kubermatic API
To expose service account token volume projection settings via KKP API, we will extend the cluster API struct with `ServiceAccountsSettings`:

```go
// ClusterSpec defines the cluster specification
type ClusterSpec struct {
...
    ServiceAccountSettings *ServiceAccountSettings `json:"serviceAccountSettings,omitempty"`
...
}
```

### KKP Dashboard / UI integration
To allow an easy enablement of service account token volume projection for KKP users, we will expose an option to enable
it on the cluster creation page. If the user selects to enable the service account token volume projection, the
`ServiceAccountsSettings` will become available to them, with defaults filled in as discussed above.


## Alternatives Considered
One of the alternatives that we considered was allowing KKP users to specify any API server flags for the user clusters.
That would provide great flexibility and extensibility options even for any future extensions. This approach would
however have the following disadvantages:

 - The cluster admin users would need to type exact API server flags and their values, which requires more knowledge
   than just enabling a boolean flag and specifying some optional values, and is more prone to typo errors as well.
 - The cluster admin users could break their clusters by providing invalid API server flags.
 - The `--service-account-signing-key-file` flag should contain a value that is specific to Kubermatic and its value
   differs to what many available configuration guides (e.g. for Istio) are suggesting.
 - The `service-account-api-audiences` may potentially need to be modified with some of the KKP controllers for some
   KKP features/extensions (e.g. Konnectivity service). That can be done more easily and cleanly if the user specifies
   just their desired API audiences, not the flag and its value as whole.
   

## Tasks & Efforts
 - Implement service account token volume projection settings in Cluster CRD - _2 days_
 - Implement service account token volume projection settings in Cluster API - _2 days_
 - Integrate with KKP UI and Dashboard - _3 days?_
 - KKP Documentation - _1 day_
