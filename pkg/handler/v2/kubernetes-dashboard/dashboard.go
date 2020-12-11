/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package kubernetesdashboard

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/go-kit/kit/endpoint"
	transporthttp "github.com/go-kit/kit/transport/http"
	"go.uber.org/zap"

	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/handler/v2/cluster"
	"k8c.io/kubermatic/v2/pkg/provider"
	kubernetesdashboard "k8c.io/kubermatic/v2/pkg/resources/kubernetes-dashboard"
	kubermaticerrors "k8c.io/kubermatic/v2/pkg/util/errors"
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
	privilegedProjectProvider provider.PrivilegedProjectProvider,
	userInfoGetter provider.UserInfoGetter,
	settingsProvider provider.SettingsProvider,
	middlewares endpoint.Middleware) http.Handler {
	return dynamicHTTPHandler(func(w http.ResponseWriter, r *http.Request) {
		log := log.With("endpoint", "kubernetes-dashboard-proxy", "uri", r.URL.Path)
		ctx := extractor(r.Context(), r)

		settings, err := settingsProvider.GetGlobalSettings()
		if err != nil {
			common.WriteHTTPError(log, w, kubermaticerrors.New(http.StatusInternalServerError, "could not read global settings"))
			return
		}

		if !settings.Spec.EnableDashboard {
			common.WriteHTTPError(log, w, kubermaticerrors.New(http.StatusForbidden, "Kubernetes Dashboard access is disabled by the global settings"))
			return
		}

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

			userCluster, clusterProvider, err := cluster.GetClusterProviderFromRequest(ctx, request, projectProvider, privilegedProjectProvider, userInfoGetter)
			if err != nil {
				common.WriteHTTPError(log, w, err)
				return nil, nil
			}
			req, ok := request.(cluster.GetClusterReq)
			if !ok {
				common.WriteHTTPError(log, w, kubermaticerrors.New(http.StatusBadRequest, "invalid request"))
				return nil, nil
			}
			userInfo, err := userInfoGetter(ctx, req.ProjectID)
			if err != nil {
				common.WriteHTTPError(log, w, kubermaticerrors.New(http.StatusInternalServerError, "couldn't get userInfo"))
				return nil, nil
			}

			token, err := clusterProvider.GetTokenForCustomerCluster(ctx, userInfo, userCluster)
			if err != nil {
				common.WriteHTTPError(log, w, kubermaticerrors.New(http.StatusBadRequest, fmt.Sprintf("error getting token for user %q: %v", userInfo.Email, err)))
				return nil, nil
			}

			log = log.With("cluster", userCluster.Name)

			// Ideally we would cache these to not open a port for every single request
			portforwarder, closeChan, err := common.GetPortForwarder(
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

			ports, err := portforwarder.GetPorts()
			if err != nil {
				common.WriteHTTPError(log, w, fmt.Errorf("failed to get backend port: %v", err))
				return nil, nil
			}
			if len(ports) != 1 {
				common.WriteHTTPError(log, w, fmt.Errorf("didn't get exactly one port but %d", len(ports)))
				return nil, nil
			}

			proxyURL := &url.URL{
				Scheme: "http",
				Host:   fmt.Sprintf("127.0.0.1:%d", ports[0].Local),
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
