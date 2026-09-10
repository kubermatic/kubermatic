# KubeLB CCM integration

KKP runs KubeLB EE CCM v1.5.0 in the Seed cluster and manages its RBAC,
SyncSecret CRD and TenantWAFPolicy CRD in the user cluster. The CRDs are sourced
from the KubeLB EE v1.5.0 CCM chart. In mTLS mode, the tenant proxy runs in the
user cluster's `kube-system` namespace.

## Project defaults

Changes to `Project.spec.defaultTenantSpec` are applied to existing KubeLB
Tenants as well as new ones. KKP manages these defaults with Server-Side Apply;
removing a default removes fields owned only by KKP. Management-side settings
owned by other field managers are preserved. Conflicting edits to the same
field are reported as reconciliation errors and must be resolved by the
administrator; KKP does not force ownership.

Existing ownership from `seed-controller-manager` is migrated for the Tenant
spec and KKP labels. Tenants created by other tools retain their field ownership.

## Images

The xDS writer uses the configured CCM image. Envoy and shutdown-manager image
defaults are defined in [images.go](resources/seed-cluster/images.go).
`overwriteRegistry` applies to default images; an explicit CCM image repository
takes precedence. The installer mirrors all three images.

## Private registries

User-cluster nodes can use `Cluster.spec.imagePullSecret` for registry access.
Alternatively, create registry Secrets in user-cluster `kube-system` and reference
them through datacenter or cluster KubeLB settings:

```yaml
extraArgs:
  tenant-proxy-image-pull-secrets: "registry-credentials"
```

A cluster `extraArgs` map replaces the datacenter map, including when empty.
Custom images supplied through `extraArgs` must also be listed in
KubermaticConfiguration `spec.mirrorImages` for mirroring.
