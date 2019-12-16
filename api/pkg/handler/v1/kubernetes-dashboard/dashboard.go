package kubernetesdashboard

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/go-kit/kit/endpoint"
	transporthttp "github.com/go-kit/kit/transport/http"
	"go.uber.org/zap"
	"k8s.io/client-go/tools/clientcmd/api"

	"github.com/kubermatic/kubermatic/api/pkg/handler/middleware"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/cluster"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	kubernetesdashboard "github.com/kubermatic/kubermatic/api/pkg/resources/kubernetes-dashboard"
	kubermaticerrors "github.com/kubermatic/kubermatic/api/pkg/util/errors"
)

// Minimal wrapper to implement the http.Handler interface
type dynamicHTTPHandler func(http.ResponseWriter, *http.Request)

// ServeHTTP implements http.Handler
func (dHandler dynamicHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	dHandler(w, r)
}

func ProxyEndpoint(
	log *zap.SugaredLogger,
	extractor transporthttp.RequestFunc,
	projectProvider provider.ProjectProvider,
	middlewares endpoint.Middleware) http.Handler {
	return dynamicHTTPHandler(func(w http.ResponseWriter, r *http.Request) {

		log := log.With("endpoint", "kubernetes-dashboard-proxy", "uri", r.URL.Path)
		ctx := extractor(r.Context(), r)
		request, err := common.DecodeGetClusterReq(ctx, r)
		if err != nil {
			common.WriteHTTPError(log, w, kubermaticerrors.New(http.StatusBadRequest, err.Error()))
			return
		}

		// The endpoint the middleware is called with is the innermost one, hence we must
		// define it as closure and pass it to the middleware() call below.
		ep := func(ctx context.Context, request interface{}) (interface{}, error) {
			// Simple redirect in case proxy call path does not end with trailing slash
			if strings.HasSuffix(r.URL.Path, "proxy") {
				http.Redirect(w, r, r.URL.Path+"/", http.StatusFound)
				return nil, nil
			}

			userCluster, clusterProvider, err := cluster.GetClusterProviderFromRequest(ctx, request, projectProvider)
			userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)
			if err != nil {
				common.WriteHTTPError(log, w, err)
				return nil, nil
			}

			adminClientCfg, err := clusterProvider.GetAdminKubeconfigForCustomerCluster(userCluster)
			if err != nil {
				common.WriteHTTPError(log, w, err)
				return nil, nil
			}

			if !strings.HasPrefix(userInfo.Group, "editors") && !strings.HasPrefix(userInfo.Group, "owners") {
				common.WriteHTTPError(log, w, kubermaticerrors.New(http.StatusBadRequest, fmt.Sprintf("user %q does not belong to the owners|editors group", userInfo.Email)))
				return nil, nil
			}

			token, err := extractBearerToken(adminClientCfg)
			if err != nil {
				common.WriteHTTPError(log, w, err)
				return nil, nil
			}

			log = log.With("cluster", userCluster.Name)

			// Ideally we would cache these to not open a port for every single request
			portforwarder, outBuffer, closeChan, err := common.GetPortForwarder(
				clusterProvider.GetSeedClusterAdminClient().CoreV1(),
				clusterProvider.SeedAdminConfig(),
				userCluster.Status.NamespaceName,
				kubernetesdashboard.AppLabel,
				kubernetesdashboard.ContainerPort)
			if err != nil {
				return nil, fmt.Errorf("failed to get portforwarder for console: %v", err)
			}
			defer func() {
				portforwarder.Close()
				close(closeChan)
			}()

			if err = common.ForwardPort(log, portforwarder); err != nil {
				common.WriteHTTPError(log, w, err)
				return nil, nil
			}

			port, err := common.GetLocalPortFromPortForwardOutput(outBuffer.String())
			if err != nil {
				common.WriteHTTPError(log, w, fmt.Errorf("failed to get backend port: %v", err))
				return nil, nil
			}

			proxyURL := &url.URL{
				Scheme: "http",
				Host:   fmt.Sprintf("127.0.0.1:%d", port),
			}

			// Override strict CSP policy for proxy
			w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; object-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self'; media-src 'self'; frame-ancestors 'self'; frame-src 'self'; connect-src 'self'; font-src 'self' data:")

			// Proxy the request
			proxy := httputil.NewSingleHostReverseProxy(proxyURL)
			proxy.Director = newDashboardProxyDirector(proxyURL, token, r).director()
			proxy.ServeHTTP(w, r)

			return nil, nil
		}

		if _, err := middlewares(ep)(ctx, request); err != nil {
			common.WriteHTTPError(log, w, err)
			return
		}
	})
}

// It's responsible for adjusting proxy request, so we can properly access Kubernetes Dashboard
type dashboardProxyDirector struct {
	proxyURL        *url.URL
	token           string
	originalRequest *http.Request
}

func (director *dashboardProxyDirector) director() func(*http.Request) {
	return func(req *http.Request) {
		req.URL.Scheme = director.proxyURL.Scheme
		req.URL.Host = director.proxyURL.Host
		req.URL.Path = director.getBasePath(director.originalRequest.URL.Path)

		req.Header.Set("Authorization", director.getAuthorizationHeader())
	}
}

func (director *dashboardProxyDirector) getAuthorizationHeader() string {
	return fmt.Sprintf("Bearer %s", director.token)
}

// We need to get proper path to Dashboard API and strip the URL from the Kubermatic API request part.
func (director *dashboardProxyDirector) getBasePath(path string) string {
	separator := "proxy"
	if !strings.Contains(path, separator) {
		return "/"
	}

	parts := strings.Split(path, separator)
	if len(parts) != 2 {
		return "/"
	}

	return parts[1]
}

func newDashboardProxyDirector(proxyURL *url.URL, token string, request *http.Request) *dashboardProxyDirector {
	return &dashboardProxyDirector{
		proxyURL:        proxyURL,
		token:           token,
		originalRequest: request,
	}
}

func extractBearerToken(kubeconfig *api.Config) (string, error) {
	for _, info := range kubeconfig.AuthInfos {
		if len(info.Token) > 0 {
			return info.Token, nil
		}
	}

	return "", errors.New("could not find bearer token in kubeconfig file")
}
