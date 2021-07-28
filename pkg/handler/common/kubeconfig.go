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

package common

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/securecookie"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/auth"
	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	kcerrors "k8c.io/kubermatic/v2/pkg/util/errors"

	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

const (
	initialPhase        = iota
	exchangeCodePhase   = iota
	kubeconfigGenerated = iota
)

const (
	csrfCookieName = "csrf_token"
	cookieMaxAge   = 180
)

var secureCookie *securecookie.SecureCookie

func GetAdminKubeconfigEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectID, clusterID string, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider) (interface{}, error) {
	clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
	cluster, err := GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID, nil)
	if err != nil {
		return nil, err
	}

	filePrefix := "admin"
	var adminClientCfg *clientcmdapi.Config

	adminUserInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	if adminUserInfo.IsAdmin {
		adminClientCfg, err = clusterProvider.GetAdminKubeconfigForCustomerCluster(cluster)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return &encodeKubeConifgResponse{clientCfg: adminClientCfg, filePrefix: filePrefix}, nil
	}

	userInfo, err := userInfoGetter(ctx, projectID)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}
	if strings.HasPrefix(userInfo.Group, "viewers") {
		filePrefix = "viewer"
		adminClientCfg, err = clusterProvider.GetViewerKubeconfigForCustomerCluster(cluster)
	} else {
		adminClientCfg, err = clusterProvider.GetAdminKubeconfigForCustomerCluster(cluster)
	}
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}
	return &encodeKubeConifgResponse{clientCfg: adminClientCfg, filePrefix: filePrefix}, nil
}

func GetOidcKubeconfigEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectID, clusterID string, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider) (interface{}, error) {
	clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

	cluster, err := GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID, nil)
	if err != nil {
		return nil, err
	}
	adminClientCfg, err := clusterProvider.GetAdminKubeconfigForCustomerCluster(cluster)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	clientCmdAuth := clientcmdapi.NewAuthInfo()
	clientCmdAuthProvider := &clientcmdapi.AuthProviderConfig{Config: map[string]string{}}
	clientCmdAuthProvider.Name = "oidc"
	clientCmdAuthProvider.Config["idp-issuer-url"] = cluster.Spec.OIDC.IssuerURL
	clientCmdAuthProvider.Config["client-id"] = cluster.Spec.OIDC.ClientID
	if cluster.Spec.OIDC.ClientSecret != "" {
		clientCmdAuthProvider.Config["client-secret"] = cluster.Spec.OIDC.ClientSecret
	}
	if cluster.Spec.OIDC.ExtraScopes != "" {
		clientCmdAuthProvider.Config["extra-scopes"] = cluster.Spec.OIDC.ExtraScopes
	}
	clientCmdAuth.AuthProvider = clientCmdAuthProvider

	adminClientCfg.AuthInfos = map[string]*clientcmdapi.AuthInfo{}
	adminClientCfg.AuthInfos["default"] = clientCmdAuth

	return &encodeKubeConifgResponse{clientCfg: adminClientCfg, filePrefix: "oidc"}, nil
}

func GetClusterOidcEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectID, clusterID string, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider) (interface{}, error) {
	cluster, err := GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID, nil)
	if err != nil {
		return nil, err
	}

	return apiv2.OIDCSpec{
		IssuerURL:    cluster.Spec.OIDC.IssuerURL,
		ClientID:     cluster.Spec.OIDC.ClientID,
		ClientSecret: cluster.Spec.OIDC.ClientSecret,
	}, nil
}

func CreateOIDCKubeconfigEndpoint(ctx context.Context, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, oidcIssuerVerifier auth.OIDCIssuerVerifier, oidcCfg common.OIDCConfiguration, req CreateOIDCKubeconfigReq) (interface{}, error) {
	oidcIssuer := oidcIssuerVerifier.(auth.OIDCIssuer)
	oidcVerifier := oidcIssuerVerifier.(auth.TokenVerifier)
	clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

	if secureCookie == nil {
		secureCookie = securecookie.New([]byte(oidcCfg.CookieHashKey), nil)
	}

	cluster, err := getClusterForOIDCEndpoint(ctx, projectProvider, privilegedProjectProvider, req.ProjectID, req.ClusterID)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	// PHASE exchangeCode handles callback response from OIDC provider
	// and generates kubeconfig
	if req.phase == exchangeCodePhase {
		// validate the state
		if req.decodedState.Nonce != req.cookieNonceValue {
			return nil, kcerrors.NewBadRequest("incorrect value of state parameter = %s", req.decodedState.Nonce)
		}
		oidcTokens, err := oidcIssuer.Exchange(ctx, req.code)
		if err != nil {
			return nil, kcerrors.NewBadRequest("error while exchanging oidc code for token = %v", err)
		}
		if len(oidcTokens.RefreshToken) == 0 {
			return nil, kcerrors.NewBadRequest("the refresh token is missing but required, try setting/unsetting \"oidc-offline-access-as-scope\" command line flag")
		}

		claims, err := oidcVerifier.Verify(ctx, oidcTokens.IDToken)
		if err != nil {
			return nil, kcerrors.New(http.StatusUnauthorized, err.Error())
		}
		if len(claims.Email) == 0 {
			return nil, kcerrors.NewBadRequest("the token doesn't contain the mandatory \"email\" claim")
		}

		adminKubeConfig, err := clusterProvider.GetAdminKubeconfigForCustomerCluster(cluster)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		// create a kubeconfig that contains OIDC tokens
		oidcKubeCfg := clientcmdapi.NewConfig()
		{
			// grab admin kubeconfig to read the cluster info
			var clusterFromAdminKubeCfg *clientcmdapi.Cluster
			for clusterName, cluster := range adminKubeConfig.Clusters {
				if clusterName == req.ClusterID {
					clusterFromAdminKubeCfg = cluster
				}
			}
			if clusterFromAdminKubeCfg == nil {
				return nil, kcerrors.New(http.StatusInternalServerError, fmt.Sprintf("unable to construct kubeconfig because couldn't find %s cluster entry in existing kubecfg", req.ClusterID))
			}

			// create cluster entry
			clientCmdCluster := clientcmdapi.NewCluster()
			clientCmdCluster.Server = clusterFromAdminKubeCfg.Server
			clientCmdCluster.CertificateAuthorityData = clusterFromAdminKubeCfg.CertificateAuthorityData
			oidcKubeCfg.Clusters[req.ClusterID] = clientCmdCluster

			// create auth entry
			clientCmdAuth := clientcmdapi.NewAuthInfo()
			clientCmdAuthProvider := &clientcmdapi.AuthProviderConfig{Config: map[string]string{}}
			clientCmdAuthProvider.Name = "oidc"
			clientCmdAuthProvider.Config["id-token"] = oidcTokens.IDToken
			clientCmdAuthProvider.Config["refresh-token"] = oidcTokens.RefreshToken
			clientCmdAuthProvider.Config["idp-issuer-url"] = oidcCfg.URL
			clientCmdAuthProvider.Config["client-id"] = oidcCfg.ClientID
			clientCmdAuthProvider.Config["client-secret"] = oidcCfg.ClientSecret
			clientCmdAuth.AuthProvider = clientCmdAuthProvider
			oidcKubeCfg.AuthInfos[claims.Email] = clientCmdAuth

			// create default ctx
			clientCmdCtx := clientcmdapi.NewContext()
			clientCmdCtx.Cluster = req.ClusterID
			clientCmdCtx.AuthInfo = claims.Email
			oidcKubeCfg.Contexts["default"] = clientCmdCtx
			oidcKubeCfg.CurrentContext = "default"
		}

		// prepare final rsp that holds kubeconfig
		rsp := createOIDCKubeconfigRsp{}
		rsp.phase = kubeconfigGenerated
		rsp.oidcKubeConfig = oidcKubeCfg
		rsp.secureCookieMode = oidcCfg.CookieSecureMode
		return rsp, nil
	}

	// PHASE initial handles request from the end-user that wants to authenticate
	// and kicksoff the process of kubeconfig generation
	if req.phase != initialPhase {
		return nil, kcerrors.NewBadRequest(fmt.Sprintf("bad request unexpected phase = %d, expected phase = %d, did you forget to set the phase while decoding the request ?", req.phase, initialPhase))
	}

	rsp := createOIDCKubeconfigRsp{}
	scopes := []string{"openid", "email"}
	if oidcCfg.OfflineAccessAsScope {
		scopes = append(scopes, "offline_access")
	}

	// pass nonce
	nonce := rand.String(rand.IntnRange(10, 15))
	rsp.nonce = nonce
	rsp.secureCookieMode = oidcCfg.CookieSecureMode

	oidcState := OIDCState{
		Nonce:     nonce,
		ClusterID: req.ClusterID,
		ProjectID: req.ProjectID,
		UserID:    req.UserID,
	}
	rawState, err := json.Marshal(oidcState)
	if err != nil {
		return nil, err
	}
	encodedState := base64.StdEncoding.EncodeToString(rawState)
	urlSafeState := url.QueryEscape(encodedState)
	rsp.authCodeURL = oidcIssuer.AuthCodeURL(urlSafeState, oidcCfg.OfflineAccessAsScope, scopes...)

	return rsp, nil
}

// CreateOIDCKubeconfigReq represent a request for creating kubeconfig for a cluster with OIDC credentials
// swagger:parameters createOIDCKubeconfig
type CreateOIDCKubeconfigReq struct {
	// in: query
	ClusterID string `json:"cluster_id,omitempty"`
	ProjectID string `json:"project_id,omitempty"`
	UserID    string `json:"user_id,omitempty"`

	// not exported so that they don't leak to swagger spec.
	code             string
	encodedState     string
	decodedState     OIDCState
	phase            int
	cookieNonceValue string
}

// OIDCState holds data that are send and retrieved from OIDC provider
type OIDCState struct {
	// nonce a random string that binds requests / responses of API server and OIDC provider
	// see https://tools.ietf.org/html/rfc6749#section-10.12
	Nonce     string `json:"nonce"`
	ClusterID string `json:"cluster_id"`
	ProjectID string `json:"project_id"`
	// UserID holds the ID of the user on behalf of which the request is being handled.
	UserID string `json:"user_id"`
}

type createOIDCKubeconfigRsp struct {
	// authCodeURL holds a URL to OpenID provider's consent page that asks for permissions for the required scopes explicitly.
	authCodeURL string
	// phase tells encoding function how to handle response
	phase int
	// oidcKubeConfig holds not serialized kubeconfig
	oidcKubeConfig *clientcmdapi.Config
	// nonce holds an arbitrary number storied in cookie to prevent Cross-site Request Forgery attack.
	nonce string
	// cookie received only with HTTPS, never with HTTP.
	secureCookieMode bool
}

func EncodeOIDCKubeconfig(c context.Context, w http.ResponseWriter, response interface{}) (err error) {
	rsp := response.(createOIDCKubeconfigRsp)

	// handles kubeconfig Generated PHASE
	// it means that kubeconfig was generated and we need to properly encode it.
	if rsp.phase == kubeconfigGenerated {
		// clear cookie by setting MaxAge<0
		err = setCookie(w, "", rsp.secureCookieMode, -1)
		if err != nil {
			return fmt.Errorf("the cookie can't be removed, err = %v", err)
		}
		return EncodeKubeconfig(c, w, &encodeKubeConifgResponse{clientCfg: rsp.oidcKubeConfig})
	}

	// handles initialPhase
	// redirects request to OpenID provider's consent page
	// and set cookie with nonce
	err = setCookie(w, rsp.nonce, rsp.secureCookieMode, cookieMaxAge)
	if err != nil {
		return fmt.Errorf("the cookie can't be created, err = %v", err)
	}
	w.Header().Add("Location", rsp.authCodeURL)
	w.Header().Add("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusSeeOther)
	return nil
}

func DecodeCreateOIDCKubeconfig(c context.Context, r *http.Request) (interface{}, error) {
	req := CreateOIDCKubeconfigReq{}

	// handle OIDC errors
	{
		errType := r.URL.Query().Get("error")
		errMessage := r.URL.Query().Get("error_description")
		if len(errMessage) != 0 {
			return nil, fmt.Errorf("OIDC provider error type = %s, description = %s", errType, errMessage)
		}
	}

	// if true - then this is a callback from OIDC provider and the next step is
	// to exchange the given code and generate kubeconfig
	// note: state is decoded here so that the middlewares can load providers (cluster) into the ctx.
	req.code = r.URL.Query().Get("code")
	req.encodedState = r.URL.Query().Get("state")
	if len(req.code) != 0 && len(req.encodedState) != 0 {
		unescapedState, err := url.QueryUnescape(req.encodedState)
		if err != nil {
			return nil, kcerrors.NewBadRequest("incorrect value of state parameter, expected url encoded value, err = %v", err)
		}
		rawState, err := base64.StdEncoding.DecodeString(unescapedState)
		if err != nil {
			return nil, kcerrors.NewBadRequest("incorrect value of state parameter, expected base64 encoded value, err = %v", err)
		}
		oidcState := OIDCState{}
		if err := json.Unmarshal(rawState, &oidcState); err != nil {
			return nil, kcerrors.NewBadRequest("incorrect value of state parameter, expected json encoded value, err = %v", err)
		}
		// handle cookie when new endpoint is created and secureCookie was initialized
		if secureCookie != nil {
			// cookie should be set in initial code phase
			if cookie, err := r.Cookie(csrfCookieName); err == nil {
				var value string
				if err = secureCookie.Decode(csrfCookieName, cookie.Value, &value); err == nil {
					req.cookieNonceValue = value
				}
			} else {
				return nil, kcerrors.NewBadRequest("incorrect value of cookie or cookie not set, err = %v", err)
			}
		}
		req.phase = exchangeCodePhase
		req.ProjectID = oidcState.ProjectID
		req.UserID = oidcState.UserID
		req.ClusterID = oidcState.ClusterID
		req.decodedState = oidcState
		return req, nil
	}

	// initial flow an end-user wants to authenticate using OIDC provider
	req.ClusterID = r.URL.Query().Get("cluster_id")
	req.ProjectID = r.URL.Query().Get("project_id")
	req.UserID = r.URL.Query().Get("user_id")
	if len(req.ClusterID) == 0 || len(req.ProjectID) == 0 || len(req.UserID) == 0 {
		return nil, errors.New("the following query parameters cluster_id, project_id, user_id and datacenter are mandatory, please make sure that all are set")
	}
	req.phase = initialPhase
	return req, nil
}

// GetUserID implements UserGetter interface
func (r CreateOIDCKubeconfigReq) GetUserID() string {
	return r.UserID
}

// GetSeedCluster returns the SeedCluster object
func (r CreateOIDCKubeconfigReq) GetSeedCluster() apiv1.SeedCluster {
	return apiv1.SeedCluster{
		ClusterID: r.ClusterID,
	}
}

// GetProjectID implements ProjectGetter interface
func (r CreateOIDCKubeconfigReq) GetProjectID() string {
	return r.ProjectID
}

// setCookie add cookie with random string value
func setCookie(w http.ResponseWriter, nonce string, secureMode bool, maxAge int) error {

	encoded, err := secureCookie.Encode(csrfCookieName, nonce)
	if err != nil {
		return fmt.Errorf("the encode cookie failed, err = %v", err)
	}
	cookie := &http.Cookie{
		Name:     csrfCookieName,
		Value:    encoded,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   secureMode,
		SameSite: http.SameSiteLaxMode,
	}

	http.SetCookie(w, cookie)
	return nil
}

func getClusterForOIDCEndpoint(ctx context.Context, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, projectID, clusterID string) (*kubermaticv1.Cluster, error) {
	clusterProvider, ok := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
	if !ok {
		return nil, kcerrors.New(http.StatusInternalServerError, "no cluster provider in request")
	}
	privilegedClusterProvider, ok := ctx.Value(middleware.PrivilegedClusterProviderContextKey).(provider.PrivilegedClusterProvider)
	if !ok {
		return nil, kcerrors.New(http.StatusInternalServerError, "no privileged cluster provider in request")
	}
	userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)

	project, err := getProjectForOIDCEndpoint(userInfo, projectProvider, privilegedProjectProvider, projectID)
	if err != nil {
		return nil, err
	}

	if userInfo.IsAdmin {
		return privilegedClusterProvider.GetUnsecured(project, clusterID, nil)
	}

	return clusterProvider.Get(userInfo, clusterID, &provider.ClusterGetOptions{})
}

func getProjectForOIDCEndpoint(userInfo *provider.UserInfo, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, projectID string) (*kubermaticv1.Project, error) {
	if userInfo.IsAdmin {
		// get any project for admin
		return privilegedProjectProvider.GetUnsecured(projectID, &provider.ProjectGetOptions{IncludeUninitialized: true})
	}

	return projectProvider.Get(userInfo, projectID, &provider.ProjectGetOptions{IncludeUninitialized: true})
}

type encodeKubeConifgResponse struct {
	clientCfg  *clientcmdapi.Config
	filePrefix string
}

func EncodeKubeconfig(c context.Context, w http.ResponseWriter, response interface{}) (err error) {
	rsp := response.(*encodeKubeConifgResponse)
	cfg := rsp.clientCfg
	filename := "kubeconfig"

	if len(rsp.filePrefix) > 0 {
		filename = fmt.Sprintf("%s-%s", filename, rsp.filePrefix)
	}

	if len(cfg.Contexts) > 0 {
		filename = fmt.Sprintf("%s-%s", filename, cfg.Contexts[cfg.CurrentContext].Cluster)
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-disposition", fmt.Sprintf("attachment; filename=%s", filename))
	w.Header().Add("Cache-Control", "no-cache")

	b, err := clientcmd.Write(*cfg)
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}
