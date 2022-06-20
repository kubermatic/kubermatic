/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/handler/v2/cluster"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	kubernetesdashboard "k8c.io/kubermatic/v2/pkg/resources/kubernetes-dashboard"
)

const tokenCookieName = "proxy"
const csp = "style-src 'self' 'unsafe-inline';"

type proxyHandler struct {
	baseHandler

	requestFuncs              []httptransport.RequestFunc
	logger                    *zap.SugaredLogger
	settingsProvider          provider.SettingsProvider
	userInfoGetter            provider.UserInfoGetter
	privilegedProjectProvider provider.PrivilegedProjectProvider
	projectProvider           provider.ProjectProvider
}

func (this *proxyHandler) Middlewares(middlewares ...endpoint.Middleware) Handler {
	this.middlewares = middlewares
	return this
}

func (this *proxyHandler) RequestFuncs(middlewares ...httptransport.RequestFunc) Handler {
	this.requestFuncs = middlewares
	return this
}

func (this *proxyHandler) Options(options ...httptransport.ServerOption) Handler {
	this.options = options
	return this
}

func (this *proxyHandler) Install(router *mux.Router) {
	router.Methods(http.MethodGet).
		PathPrefix("/projects/{project_id}/clusters/{cluster_id}/dashboard/proxy").
		Queries("token", "{token}").
		Handler(this.storeTokenHandler())

	router.PathPrefix("/projects/{project_id}/clusters/{cluster_id}/dashboard/proxy").
		Handler(this)
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/dashboard/proxy
//
//	Stores the user cluster access token in a header cookie and redirects to a raw
//	proxy endpoint (without token query param).
//
//	Parameters:
//		+ name: project_id
//		  in: path
//		  required: true
//		  type: string
//		+ name: cluster_id
//		  in: path
//		  required: true
//		  type: string
//		+ name: token
//		  in: query
//		  required: true
//		  type: string
//
//	Responses:
//		default: empty
func (this *proxyHandler) storeTokenHandler() http.Handler {
	return httptransport.NewServer(
		this.chain(this.storeToken),
		this.decodeProxyRequest,
		this.encodeProxyResponse,
		this.options...,
	)
}

func (this *proxyHandler) decodeProxyRequest(_ context.Context, r *http.Request) (interface{}, error) {
	return NewProxyRequest(r), nil
}

func (this *proxyHandler) encodeProxyResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	proxyResponse := response.(*ProxyResponse)

	http.SetCookie(w, &http.Cookie{Name: tokenCookieName, Value: proxyResponse.token})
	http.Redirect(w, proxyResponse.Request, proxyResponse.Request.URL.Path, http.StatusSeeOther)
	return nil
}

func (this *proxyHandler) storeToken(ctx context.Context, request interface{}) (interface{}, error) {
	proxyRequest := request.(*ProxyRequest)

	// Make sure the global settings have the Dashboard integration enabled.
	if err := isEnabled(ctx, this.settingsProvider); err != nil {
		return nil, err
	}

	if len(proxyRequest.Token) < 1 {
		return nil, fmt.Errorf("required token query parameter is missing")
	}

	return &ProxyResponse{
		Request: proxyRequest.Request,
		token:   proxyRequest.Token,
	}, nil
}

// swagger:route GET /api/v2/projects/{project_id}/clusters/{cluster_id}/dashboard/proxy
// Implements http.Handler interface
//
//	Starts a simple reverse proxy to access the Kubernetes Dashboard installed inside the
//	user cluster
//
//	Responses:
//		default: empty
func (this *proxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	proxyRequest := NewProxyRequest(r)

	for _, requestFuncs := range this.requestFuncs {
		ctx = requestFuncs(ctx, r)
	}

	if _, err := this.chain(this.proxy(w, r))(ctx, proxyRequest); err != nil {
		common.WriteHTTPError(this.logger, w, err)
		return
	}
}

func (this *proxyHandler) proxy(w http.ResponseWriter, request *http.Request) endpoint.Endpoint {
	return func(ctx context.Context, _ interface{}) (interface{}, error) {
		// Make sure the global settings have the Dashboard integration enabled.
		if err := isEnabled(ctx, this.settingsProvider); err != nil {
			return nil, err
		}

		// Simple redirect in case proxy call path does not end with trailing slash.
		if strings.HasSuffix(request.URL.Path, "proxy") {
			http.Redirect(w, request, fmt.Sprintf("%s/", request.URL.Path), http.StatusFound)
			return nil, nil
		}

		token, err := this.getCookie(request, tokenCookieName)
		if err != nil {
			return nil, fmt.Errorf("required cookie %s missing: %w", tokenCookieName, err)
		}

		clusterRequest, err := cluster.DecodeGetClusterReq(ctx, request)
		if err != nil {
			return nil, err
		}

		userCluster, clusterProvider, err := cluster.GetClusterProviderFromRequest(ctx, clusterRequest, this.projectProvider, this.privilegedProjectProvider, this.userInfoGetter)
		if err != nil {
			return nil, err
		}

		proxyURL, closeChan, err := this.getProxyURL(ctx, clusterProvider, userCluster)
		if err != nil {
			return nil, err
		}

		defer func() {
			close(closeChan)
		}()

		// Override strict CSP policy for proxy
		w.Header().Set("Content-Security-Policy", csp)

		// Proxy the request
		proxy := httputil.NewSingleHostReverseProxy(proxyURL)
		proxy.Director = newDashboardProxyDirector(proxyURL, token, request).director()
		proxy.ServeHTTP(w, request)

		return nil, nil
	}
}

func (this *proxyHandler) getCookie(request *http.Request, name string) (string, error) {
	cookie, err := request.Cookie(name)
	if err != nil {
		return "", err
	}

	return cookie.Value, nil
}

func (this *proxyHandler) getProxyURL(ctx context.Context, clusterProvider *kubernetes.ClusterProvider, userCluster *kubermaticv1.Cluster) (proxyURL *url.URL, closeChan chan struct{}, err error) {
	// Ideally we would cache these to not open a port for every single request
	portforwarder, closeChan, err := common.GetPortForwarder(
		ctx,
		clusterProvider.GetSeedClusterAdminClient().CoreV1(),
		clusterProvider.SeedAdminConfig(),
		userCluster.Status.NamespaceName,
		kubernetesdashboard.AppLabel,
		kubernetesdashboard.ContainerPort)
	if err != nil {
		return proxyURL, closeChan, fmt.Errorf("failed to get portforwarder for console: %w", err)
	}

	if err = common.ForwardPort(this.logger, portforwarder); err != nil {
		return
	}

	ports, err := portforwarder.GetPorts()
	if err != nil {
		return proxyURL, closeChan, fmt.Errorf("failed to get backend port: %w", err)
	}

	if len(ports) != 1 {
		return proxyURL, closeChan, fmt.Errorf("didn't get exactly one port but %d", len(ports))
	}

	proxyURL = &url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("127.0.0.1:%d", ports[0].Local),
	}

	return
}

func NewProxyHandler(
	logger *zap.SugaredLogger,
	settingsProvider provider.SettingsProvider,
	projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider,
	userInfoGetter provider.UserInfoGetter) *proxyHandler {
	return &proxyHandler{
		logger:                    logger,
		settingsProvider:          settingsProvider,
		projectProvider:           projectProvider,
		privilegedProjectProvider: privilegedProjectProvider,
		userInfoGetter:            userInfoGetter,
	}
}
