package features

const (
	// All features for the kubermatic-api are defined below.

	// PrometheusEndpoint if enabled exposes cluster's metrics HTTP endpoint
	PrometheusEndpoint = "PrometheusEndpoint"

	// OIDCKubeCfgEndpoint if enabled exposes an HTTP endpoint for generating kubeconfig for a cluster that will contain OIDC tokens
	OIDCKubeCfgEndpoint = "OIDCKubeCfgEndpoint"
)
