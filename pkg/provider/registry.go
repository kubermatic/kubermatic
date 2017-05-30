package provider

// KubernetesRegistry holds all known Kubernetes providers.
type KubernetesRegistry map[string]KubernetesProvider

// CloudRegistry stores all known cloud providers.
type CloudRegistry map[string]CloudProvider
