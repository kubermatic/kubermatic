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

package openshift

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/go-kit/kit/endpoint"
	transporthttp "github.com/go-kit/kit/transport/http"
	"go.uber.org/zap"

	openshiftresources "k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/openshift/resources"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/v1/cluster"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	kubermaticerrors "k8c.io/kubermatic/v2/pkg/util/errors"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// Minimal wrapper to implement the http.Handler interface
type dynamicHTTPHandler func(http.ResponseWriter, *http.Request)

// ServeHTTP implements http.Handler
func (dHandler dynamicHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	dHandler(w, r)
}

// ConsoleLoginEndpoint is an endpoint that gets an oauth token for the user from the openshift
// oauth service, then redirects back to the openshift console
func ConsoleLoginEndpoint(
	log *zap.SugaredLogger,
	extractor transporthttp.RequestFunc,
	projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider,
	userInfoGetter provider.UserInfoGetter,
	middlewares endpoint.Middleware) http.Handler {
	return dynamicHTTPHandler(func(w http.ResponseWriter, r *http.Request) {

		log := log.With("endpoint", "openshift-console-login", "uri", r.URL.Path)
		ctx := extractor(r.Context(), r)
		request, err := common.DecodeGetClusterReq(ctx, r)
		if err != nil {
			common.WriteHTTPError(log, w, kubermaticerrors.New(http.StatusBadRequest, err.Error()))
			return
		}

		// The endpoint the middleware is called with is the innermost one, hence we must
		// define it as closure and pass it to the middleware() call below.
		endpoint := func(ctx context.Context, request interface{}) (interface{}, error) {
			cluster, clusterProvider, err := cluster.GetClusterProviderFromRequest(ctx, request, projectProvider, privilegedProjectProvider, userInfoGetter)
			if err != nil {
				common.WriteHTTPError(log, w, err)
				return nil, nil
			}
			log = log.With("cluster", cluster.Name)
			req, ok := request.(common.GetClusterReq)
			if !ok {
				common.WriteHTTPError(log, w, kubermaticerrors.New(http.StatusBadRequest, "invalid request"))
				return nil, nil
			}
			userInfo, err := userInfoGetter(ctx, req.ProjectID)
			if err != nil {
				common.WriteHTTPError(log, w, kubermaticerrors.New(http.StatusInternalServerError, "couldn't get userInfo"))
				return nil, nil
			}
			if strings.HasPrefix(userInfo.Group, "editors") || strings.HasPrefix(userInfo.Group, "owners") {
				consoleLogin(ctx, log, w, cluster, clusterProvider.GetSeedClusterAdminRuntimeClient(), r)
			} else {
				common.WriteHTTPError(log, w, kubermaticerrors.New(http.StatusBadRequest, fmt.Sprintf("user %q does not belong to the editors group", userInfo.Email)))
			}

			return nil, nil
		}
		if _, err := middlewares(endpoint)(ctx, request); err != nil {
			common.WriteHTTPError(log, w, err)
			return
		}
	})
}

// ConsoleProxyEndpoint is an endpoint that proxies to the Openshift console running
// in the seed. It also performs authentication on the users behalf. Currently, it only supports
// login as cluster-admin user, so this must not be accessible for users that are not cluster admin.
func ConsoleProxyEndpoint(
	log *zap.SugaredLogger,
	extractor transporthttp.RequestFunc,
	projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider,
	userInfoGetter provider.UserInfoGetter,
	middlewares endpoint.Middleware) http.Handler {
	return dynamicHTTPHandler(func(w http.ResponseWriter, r *http.Request) {

		log := log.With("endpoint", "openshift-console-proxy", "uri", r.URL.Path)
		ctx := extractor(r.Context(), r)
		request, err := common.DecodeGetClusterReq(ctx, r)
		if err != nil {
			common.WriteHTTPError(log, w, kubermaticerrors.New(http.StatusBadRequest, err.Error()))
			return
		}

		// The endpoint the middleware is called with is the innermost one, hence we must
		// define it as closure and pass it to the middleware() call below.
		endpoint := func(ctx context.Context, request interface{}) (interface{}, error) {
			cluster, clusterProvider, err := cluster.GetClusterProviderFromRequest(ctx, request, projectProvider, privilegedProjectProvider, userInfoGetter)
			if err != nil {
				common.WriteHTTPError(log, w, err)
				return nil, nil
			}
			log = log.With("cluster", cluster.Name)

			// Ideally we would cache these to not open a port for every single request
			portforwarder, closeChan, err := common.GetPortForwarder(
				clusterProvider.GetSeedClusterAdminClient().CoreV1(),
				clusterProvider.SeedAdminConfig(),
				cluster.Status.NamespaceName,
				// TODO: Export the labelselector from the openshift resources
				"app=openshift-console",
				openshiftresources.ConsoleListenPort)
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

			// The Openshift console needs script-src: unsafe-inline and sryle-src: unsafe-inline.
			// The header here overwrites the setting on the main router, which is more strict.
			w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; object-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self'; media-src 'self'; frame-ancestors 'self'; frame-src 'self'; connect-src 'self'")
			// Proxy the request
			proxy := httputil.NewSingleHostReverseProxy(proxyURL)
			proxy.ServeHTTP(w, r)

			return nil, nil
		}

		if _, err := middlewares(endpoint)(ctx, request); err != nil {
			common.WriteHTTPError(log, w, err)
			return
		}
	})

}

// consoleLogin loggs an user into the console by doing the oauth login, then returning a redirect.
// This is not done by the user themselves, because:
// * The openshift OAuth server is under the same URL as the kubermatic UI but doesn't have a
//   certificate signed by a CA the browser has. This mean that if HSTS is enabled, the browser
//   won't allow the user to visit that URL.
// * It is poor UX to require the User to login twice.
func consoleLogin(
	ctx context.Context,
	log *zap.SugaredLogger,
	w http.ResponseWriter,
	cluster *kubermaticv1.Cluster,
	seedClient ctrlruntimeclient.Client,
	initialRequest *http.Request) {

	log.Debug("Login request received")

	oauthServiceName := types.NamespacedName{
		Namespace: cluster.Status.NamespaceName,
		Name:      openshiftresources.OAuthServiceName,
	}
	oauthService := &corev1.Service{}
	if err := seedClient.Get(ctx, oauthServiceName, oauthService); err != nil {
		common.WriteHTTPError(log, w, fmt.Errorf("failed to retrieve oauth service: %v", err))
		return
	}
	if n := len(oauthService.Spec.Ports); n != 1 {
		common.WriteHTTPError(log, w, fmt.Errorf("OAuth service doesn't have exactly one port but %d", n))
		return
	}
	oauthPort := oauthService.Spec.Ports[0].NodePort

	oauthPasswordSecretName := types.NamespacedName{
		Namespace: cluster.Status.NamespaceName,
		Name:      openshiftresources.ConsoleAdminPasswordSecretName,
	}
	oauthPasswordSecret := &corev1.Secret{}
	if err := seedClient.Get(ctx, oauthPasswordSecretName, oauthPasswordSecret); err != nil {
		common.WriteHTTPError(log, w, fmt.Errorf("failed to get OAuth credential secret: %v", err))
		return
	}
	oauthPassword := string(oauthPasswordSecret.Data[openshiftresources.ConsoleAdminUserName])
	if oauthPassword == "" {
		common.WriteHTTPError(log, w, errors.New("no OAuth password found"))
		return
	}

	oauthStateValue, err := generateRandomOauthState()
	if err != nil {
		common.WriteHTTPError(log, w, fmt.Errorf("failed to get oauth state token: %v", err))
		return
	}

	queryArgs := url.Values{
		"client_id":     []string{"console"},
		"response_type": []string{"code"},
		"scope":         []string{"user:full"},
		"state":         []string{oauthStateValue},
	}
	// TODO: Should we put that into cluster.Address?
	oauthURL, err := url.Parse(fmt.Sprintf("https://%s:%d/oauth/authorize", cluster.Address.ExternalName, oauthPort))
	if err != nil {
		common.WriteHTTPError(log, w, fmt.Errorf("failed to parse oauth url: %v", err))
		return
	}
	oauthURL.RawQuery = queryArgs.Encode()

	oauthRequest, err := http.NewRequest(http.MethodGet, oauthURL.String(), nil)
	if err != nil {
		common.WriteHTTPError(log, w, fmt.Errorf("failed to construct query for oauthRequest: %v", err))
		return
	}
	oauthRequest.SetBasicAuth(openshiftresources.ConsoleAdminUserName, oauthPassword)

	resp, err := httpRequestOAuthClient().Do(oauthRequest)
	if err != nil {
		common.WriteHTTPError(log, w, fmt.Errorf("failed to get oauth code: %v", err))
		return
	}
	defer resp.Body.Close()

	redirectURL, err := resp.Location()
	if err != nil {
		common.WriteHTTPError(log, w, fmt.Errorf("failed to get redirectURL: %v", err))
		return
	}

	oauthCode := redirectURL.Query().Get("code")
	if oauthCode == "" {
		common.WriteHTTPError(log, w, errors.New("did not get an OAuth code back from Openshift OAuth server"))
		return
	}
	// We don't check this here again. If something is wrong with it, Openshift will complain
	returnedOAuthState := redirectURL.Query().Get("state")
	http.SetCookie(w, &http.Cookie{Name: "state-token", Value: returnedOAuthState})

	redirectQueryArgs := url.Values{
		"state": []string{returnedOAuthState},
		"code":  []string{oauthCode},
	}
	// Leave the Host unset, http.Redirect will fill it with the host from the original request
	redirectTargetURLRaw := strings.Replace(initialRequest.URL.Path, "login", "proxy/auth/callback", 1)
	redirectTargetURL, err := url.Parse(redirectTargetURLRaw)
	if err != nil {
		common.WriteHTTPError(log, w, fmt.Errorf("failed to parse target redirect URL: %v", err))
		return
	}
	redirectTargetURL.RawQuery = redirectQueryArgs.Encode()

	http.Redirect(w, initialRequest, redirectTargetURL.String(), http.StatusFound)
}

// generateRandomOauthState generates a random string that is being used when performing the
// oauth request. The Openshift console checks that the query param on the request it received
// matches a cookie:
// https://github.com/openshift/console/blob/5c80c44d31e244b01dd9bbb4c8b1adec18e3a46b/auth/auth.go#L375
func generateRandomOauthState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to get entropy: %v", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// httpRequestOAuthClient is used to perform the OAuth request.
// it needs some special settings.
func httpRequestOAuthClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			// TODO: Fetch the CA instead and use it for verification
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
		// We must not follow the redirect
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}
