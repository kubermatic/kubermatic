package common

import (
	"context"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/presets"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/version"
	"github.com/kubermatic/machine-controller/pkg/providerconfig"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ResourceMetricsInfo is a struct that holds the node metrics
type ResourceMetricsInfo struct {
	Name      string
	Metrics   corev1.ResourceList
	Available corev1.ResourceList
}

// OIDCConfiguration is a struct that holds
// OIDC provider configuration data, read from command line arguments
type OIDCConfiguration struct {
	// URL holds OIDC Issuer URL address
	URL string
	// ClientID holds OIDC ClientID
	ClientID string
	// ClientSecret holds OIDC ClientSecret
	ClientSecret string
	// CookieHashKey is required, used to authenticate the cookie value using HMAC
	// It is recommended to use a key with 32 or 64 bytes.
	CookieHashKey string
	// CookieSecureMode if true then cookie received only with HTTPS otherwise with HTTP.
	CookieSecureMode bool
	// OfflineAccessAsScope if true then "offline_access" scope will be used
	// otherwise 'access_type=offline" query param will be passed
	OfflineAccessAsScope bool
}

// UpdateManager specifies a set of methods to handle cluster versions & updates
type UpdateManager interface {
	GetVersion(from, clusterType string) (*version.Version, error)
	GetVersions(string) ([]*version.Version, error)
	GetDefault() (*version.Version, error)
	GetPossibleUpdates(from, clusterType string) ([]*version.Version, error)
}

// PresetsManager specifies a set of methods to handle presets for specific provider
type PresetsManager interface {
	GetPresets() *presets.Presets
	SetCloudCredentials(credentialName string, cloud kubermaticv1.CloudSpec, dc *kubermaticv1.Datacenter) (*kubermaticv1.CloudSpec, error)
}

// ServerMetrics defines metrics used by the API.
type ServerMetrics struct {
	HTTPRequestsTotal          *prometheus.CounterVec
	HTTPRequestsDuration       *prometheus.HistogramVec
	InitNodeDeploymentFailures *prometheus.CounterVec
}

// IsBringYourOwnProvider determines whether the spec holds BringYourOwn provider
func IsBringYourOwnProvider(spec kubermaticv1.CloudSpec) (bool, error) {
	providerName, err := provider.ClusterCloudProviderName(spec)
	if err != nil {
		return false, err
	}
	return providerName == provider.BringYourOwnCloudProvider, nil
}

type CredentialsData struct {
	Ctx               context.Context
	KubermaticCluster *kubermaticv1.Cluster
	Client            ctrlruntimeclient.Client
}

func (d CredentialsData) Cluster() *kubermaticv1.Cluster {
	return d.KubermaticCluster
}

func (d CredentialsData) GetGlobalSecretKeySelectorValue(configVar *providerconfig.GlobalSecretKeySelector, key string) (string, error) {
	if configVar.Name != "" && configVar.Namespace != "" && key != "" {
		secret := &corev1.Secret{}
		name := types.NamespacedName{Namespace: configVar.Namespace, Name: configVar.Name}
		if err := d.Client.Get(d.Ctx, name, secret); err != nil {
			return "", fmt.Errorf("error retrieving secret %q from namespace %q: %v", configVar.Name, configVar.Namespace, err)
		}

		if val, ok := secret.Data[key]; ok {
			return string(val), nil
		}
		return "", fmt.Errorf("secret %q in namespace %q has no key %q", configVar.Name, configVar.Namespace, key)
	}
	return "", nil
}
