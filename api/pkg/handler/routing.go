package handler

import (
	"os"

	"github.com/go-kit/kit/log"
	httptransport "github.com/go-kit/kit/transport/http"
	prometheusapi "github.com/prometheus/client_golang/api"

	"github.com/kubermatic/kubermatic/api/pkg/handler/auth"
	"github.com/kubermatic/kubermatic/api/pkg/handler/middleware"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/serviceaccount"
	"github.com/kubermatic/kubermatic/api/pkg/version"

	corev1 "k8s.io/api/core/v1"
)

// UpdateManager specifies a set of methods to handle cluster versions & updates
type UpdateManager interface {
	GetVersion(from, clusterType string) (*version.MasterVersion, error)
	GetMasterVersions(clusterType string) ([]*version.MasterVersion, error)
	GetDefault() (*version.MasterVersion, error)
	AutomaticUpdate(from, clusterType string) (*version.MasterVersion, error)
	GetPossibleUpdates(from, clusterType string) ([]*version.MasterVersion, error)
}

// Routing represents an object which binds endpoints to http handlers.
type Routing struct {
	datacenters                 map[string]provider.DatacenterMeta
	cloudProviders              provider.CloudRegistry
	sshKeyProvider              provider.SSHKeyProvider
	userProvider                provider.UserProvider
	serviceAccountProvider      provider.ServiceAccountProvider
	serviceAccountTokenProvider provider.ServiceAccountTokenProvider
	projectProvider             provider.ProjectProvider
	privilegedProjectProvider   provider.PrivilegedProjectProvider
	logger                      log.Logger
	oidcIssuerVerifier          auth.OIDCIssuerVerifier
	tokenVerifiers              auth.TokenVerifier
	tokenExtractors             auth.TokenExtractor
	clusterProviders            map[string]provider.ClusterProvider
	updateManager               common.UpdateManager
	prometheusClient            prometheusapi.Client
	projectMemberProvider       provider.ProjectMemberProvider
	userProjectMapper           provider.ProjectMemberMapper
	saTokenAuthenticator        serviceaccount.TokenAuthenticator
	saTokenGenerator            serviceaccount.TokenGenerator
	eventRecorderProvider       provider.EventRecorderProvider
	credentialManager           common.CredentialManager
	exposeStrategy              corev1.ServiceType
}

// NewRouting creates a new Routing.
func NewRouting(
	datacenters map[string]provider.DatacenterMeta,
	newClusterProviders map[string]provider.ClusterProvider,
	cloudProviders map[string]provider.CloudProvider,
	newSSHKeyProvider provider.SSHKeyProvider,
	userProvider provider.UserProvider,
	serviceAccountProvider provider.ServiceAccountProvider,
	serviceAccountTokenProvider provider.ServiceAccountTokenProvider,
	projectProvider provider.ProjectProvider,
	privilegedProject provider.PrivilegedProjectProvider,
	oidcIssuerVerifier auth.OIDCIssuerVerifier,
	tokenVerifiers auth.TokenVerifier,
	tokenExtractors auth.TokenExtractor,
	updateManager common.UpdateManager,
	prometheusClient prometheusapi.Client,
	projectMemberProvider provider.ProjectMemberProvider,
	userProjectMapper provider.ProjectMemberMapper,
	saTokenAuthenticator serviceaccount.TokenAuthenticator,
	saTokenGenerator serviceaccount.TokenGenerator,
	eventRecorderProvider provider.EventRecorderProvider,
	credentialManager common.CredentialManager,
	exposeStrategy corev1.ServiceType,
) Routing {
	return Routing{
		datacenters:                 datacenters,
		clusterProviders:            newClusterProviders,
		sshKeyProvider:              newSSHKeyProvider,
		userProvider:                userProvider,
		serviceAccountProvider:      serviceAccountProvider,
		serviceAccountTokenProvider: serviceAccountTokenProvider,
		projectProvider:             projectProvider,
		privilegedProjectProvider:   privilegedProject,
		cloudProviders:              cloudProviders,
		logger:                      log.NewLogfmtLogger(os.Stderr),
		oidcIssuerVerifier:          oidcIssuerVerifier,
		tokenVerifiers:              tokenVerifiers,
		tokenExtractors:             tokenExtractors,
		updateManager:               updateManager,
		prometheusClient:            prometheusClient,
		projectMemberProvider:       projectMemberProvider,
		userProjectMapper:           userProjectMapper,
		saTokenAuthenticator:        saTokenAuthenticator,
		saTokenGenerator:            saTokenGenerator,
		eventRecorderProvider:       eventRecorderProvider,
		credentialManager:           credentialManager,
		exposeStrategy:              exposeStrategy,
	}
}

func (r Routing) defaultServerOptions() []httptransport.ServerOption {
	return []httptransport.ServerOption{
		httptransport.ServerErrorLogger(r.logger),
		httptransport.ServerErrorEncoder(errorEncoder),
		httptransport.ServerBefore(middleware.TokenExtractor(r.tokenExtractors)),
	}
}
