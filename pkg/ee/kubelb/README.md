# KubeLB CCM integration

KKP runs KubeLB EE CCM v1.5.0 in the Seed cluster and manages its RBAC,
SyncSecret CRD and TenantWAFPolicy CRD in the user cluster. The CRDs are sourced
from the KubeLB EE v1.5.0 CCM chart. In mTLS mode, the tenant proxy runs in the
user cluster's `kube-system` namespace.

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
