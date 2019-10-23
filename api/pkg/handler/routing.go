package handler

import (
	"os"

	"github.com/go-kit/kit/log"
	httptransport "github.com/go-kit/kit/transport/http"
	prometheusapi "github.com/prometheus/client_golang/api"
	"go.uber.org/zap"

	"github.com/kubermatic/kubermatic/api/pkg/handler/auth"
	"github.com/kubermatic/kubermatic/api/pkg/handler/middleware"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/serviceaccount"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

// Routing represents an object which binds endpoints to http handlers.
type Routing struct {
	log                         *zap.SugaredLogger
	seedsGetter                 provider.SeedsGetter
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
	clusterProviderGetter       provider.ClusterProviderGetter
	addonProviderGetter         provider.AddonProviderGetter
	updateManager               common.UpdateManager
	prometheusClient            prometheusapi.Client
	projectMemberProvider       provider.ProjectMemberProvider
	userProjectMapper           provider.ProjectMemberMapper
	saTokenAuthenticator        serviceaccount.TokenAuthenticator
	saTokenGenerator            serviceaccount.TokenGenerator
	eventRecorderProvider       provider.EventRecorderProvider
	presetsManager              common.PresetsManager
	exposeStrategy              corev1.ServiceType
	accessibleAddons            sets.String
	userInfoGetter              provider.UserInfoGetter
}

// NewRouting creates a new Routing.
func NewRouting(
	logger *zap.SugaredLogger,
	seedsGetter provider.SeedsGetter,
	clusterProviderGetter provider.ClusterProviderGetter,
	addonProviderGetter provider.AddonProviderGetter,
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
	presetsManager common.PresetsManager,
	exposeStrategy corev1.ServiceType,
	accessibleAddons sets.String,
	userInfoGetter provider.UserInfoGetter,
) Routing {
	return Routing{
		log:                         logger,
		seedsGetter:                 seedsGetter,
		clusterProviderGetter:       clusterProviderGetter,
		addonProviderGetter:         addonProviderGetter,
		sshKeyProvider:              newSSHKeyProvider,
		userProvider:                userProvider,
		serviceAccountProvider:      serviceAccountProvider,
		serviceAccountTokenProvider: serviceAccountTokenProvider,
		projectProvider:             projectProvider,
		privilegedProjectProvider:   privilegedProject,
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
		presetsManager:              presetsManager,
		exposeStrategy:              exposeStrategy,
		accessibleAddons:            accessibleAddons,
		userInfoGetter:              userInfoGetter,
	}
}

func (r Routing) defaultServerOptions() []httptransport.ServerOption {
	return []httptransport.ServerOption{
		httptransport.ServerErrorLogger(r.logger),
		httptransport.ServerErrorEncoder(errorEncoder),
		httptransport.ServerBefore(middleware.TokenExtractor(r.tokenExtractors)),
	}
}
