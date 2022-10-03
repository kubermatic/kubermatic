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
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/gorilla/securecookie"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/auth"
	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	apiserverserviceaccount "k8s.io/apiserver/pkg/authentication/serviceaccount"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	initialPhase        = iota
	exchangeCodePhase   = iota
	kubeconfigGenerated = iota
)

const (
	csrfCookieName = "csrf_token"
	cookieMaxAge   = 180
	oidc           = "oidc"

	// defaultCtx is the name of the default context in kubeconfig.
	defaultCtx = "default"
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
		adminClientCfg, err = clusterProvider.GetAdminKubeconfigForUserCluster(ctx, cluster)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return &encodeKubeConifgResponse{clientCfg: adminClientCfg, filePrefix: filePrefix}, nil
	}

	userInfo, err := userInfoGetter(ctx, projectID)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}
	if userInfo.Roles.Has("viewers") && userInfo.Roles.Len() == 1 {
		filePrefix = "viewer"
		adminClientCfg, err = clusterProvider.GetViewerKubeconfigForUserCluster(ctx, cluster)
	} else {
		adminClientCfg, err = clusterProvider.GetAdminKubeconfigForUserCluster(ctx, cluster)
	}
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}
	return &encodeKubeConifgResponse{clientCfg: adminClientCfg, filePrefix: filePrefix}, nil
}

func GetKubeconfigEndpoint(ctx context.Context, cluster *kubermaticv1.ExternalCluster, privilegedClusterProvider provider.PrivilegedExternalClusterProvider) (interface{}, error) {
	filePrefix := "external-cluster"

	kubeconfigReference := cluster.Spec.KubeconfigReference
	if kubeconfigReference == nil {
		return nil, fmt.Errorf("kubeconfig not available for the Cluster")
	}

	secretKeyGetter := provider.SecretKeySelectorValueFuncFactory(ctx, privilegedClusterProvider.GetMasterClient())

	rawKubeconfig, err := secretKeyGetter(kubeconfigReference, resources.KubeconfigSecretKey)
	if err != nil {
		return nil, err
	}

	cfg, err := clientcmd.Load([]byte(rawKubeconfig))
	if err != nil {
		return nil, fmt.Errorf("failed to parse kubeconfig: %w", err)
	}

	return &encodeKubeConifgResponse{clientCfg: cfg, filePrefix: filePrefix}, nil
}

// GetClusterSAKubeconigEndpoint returns the kubeconfig associated to a service account in the cluster.
func GetClusterSAKubeconigEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectID string, clusterID string, serviceAccountNamespace, serviceAccountName string, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider) (interface{}, error) {
	clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
	cluster, err := GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID, nil)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	client, err := common.GetClusterClient(ctx, userInfoGetter, clusterProvider, cluster, projectID)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	token, err := getServiceAccountToken(ctx, client, serviceAccountNamespace, serviceAccountName)
	if err != nil {
		return nil, err
	}

	adminKubeConfig, err := clusterProvider.GetAdminKubeconfigForUserCluster(ctx, cluster)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	// create a kubeconfig that contains service account token
	saKubeConfig := clientcmdapi.NewConfig()

	// grab admin kubeconfig to read the cluster info
	var clusterFromAdminKubeCfg *clientcmdapi.Cluster
	for clusterName, cluster := range adminKubeConfig.Clusters {
		if clusterName == clusterID {
			clusterFromAdminKubeCfg = cluster
		}
	}
	if clusterFromAdminKubeCfg == nil {
		return nil, utilerrors.New(http.StatusInternalServerError, fmt.Sprintf("unable to construct kubeconfig because couldn't find %s cluster entry in existing kubecfg", clusterID))
	}

	// create cluster entry
	clientCmdCluster := clientcmdapi.NewCluster()
	clientCmdCluster.Server = clusterFromAdminKubeCfg.Server
	clientCmdCluster.CertificateAuthorityData = clusterFromAdminKubeCfg.CertificateAuthorityData
	saKubeConfig.Clusters[clusterID] = clientCmdCluster

	// create auth entry
	userName := "sa-" + serviceAccountName
	clientCmdAuth := clientcmdapi.NewAuthInfo()
	clientCmdAuth.Token = token
	saKubeConfig.AuthInfos[userName] = clientCmdAuth

	// create context
	clientCmdCtx := clientcmdapi.NewContext()
	clientCmdCtx.Cluster = clusterID
	clientCmdCtx.AuthInfo = userName
	saKubeConfig.Contexts[defaultCtx] = clientCmdCtx
	saKubeConfig.CurrentContext = defaultCtx

	return &encodeKubeConifgResponse{clientCfg: saKubeConfig, filePrefix: userName}, nil
}

// getServiceAccountToken returns the token associated to the k8s service account named serviceAccountID in serviceAccountNamespace.
// An error is returned for the following cases:
//   - service account does not exist
//   - service account does not have token associated. (ie no secret annoated with service account's name and uid)
//   - secret that stores token does not have key "token"
func getServiceAccountToken(ctx context.Context, client ctrlruntimeclient.Client, serviceAccountNamespace string, serviceAccountName string) (string, error) {
	serviceAccount := &corev1.ServiceAccount{}
	if err := client.Get(ctx, types.NamespacedName{Namespace: serviceAccountNamespace, Name: serviceAccountName}, serviceAccount); err != nil {
		return "", common.KubernetesErrorToHTTPError(err)
	}

	secretList := &corev1.SecretList{}
	if err := client.List(ctx, secretList, ctrlruntimeclient.InNamespace(serviceAccount.Namespace)); err != nil {
		return "", common.KubernetesErrorToHTTPError(err)
	}

	for _, secret := range secretList.Items {
		if apiserverserviceaccount.IsServiceAccountToken(&secret, serviceAccount) {
			token := secret.Data[corev1.ServiceAccountTokenKey]
			if len(token) == 0 {
				return "", utilerrors.New(http.StatusInternalServerError, "no token defined in the service account's secret")
			}
			return string(token), nil
		}
	}

	return "", utilerrors.New(http.StatusInternalServerError, "service account has no secret")
}

func GetOidcKubeconfigEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectID, clusterID string, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider) (interface{}, error) {
	clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

	cluster, err := GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID, nil)
	if err != nil {
		return nil, err
	}
	adminClientCfg, err := clusterProvider.GetAdminKubeconfigForUserCluster(ctx, cluster)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	clientCmdAuth := clientcmdapi.NewAuthInfo()
	clientCmdAuthProvider := &clientcmdapi.AuthProviderConfig{Config: map[string]string{}}
	clientCmdAuthProvider.Name = oidc
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

	return &encodeKubeConifgResponse{clientCfg: adminClientCfg, filePrefix: oidc}, nil
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
			return nil, utilerrors.NewBadRequest("incorrect value of state parameter: %s", req.decodedState.Nonce)
		}
		oidcTokens, err := oidcIssuer.Exchange(ctx, req.code, "")
		if err != nil {
			return nil, utilerrors.NewBadRequest("error while exchanging oidc code for token: %v", err)
		}
		if len(oidcTokens.RefreshToken) == 0 {
			return nil, utilerrors.NewBadRequest("the refresh token is missing but required, try setting/unsetting \"oidc-offline-access-as-scope\" command line flag")
		}

		claims, err := oidcVerifier.Verify(ctx, oidcTokens.IDToken)
		if err != nil {
			return nil, utilerrors.New(http.StatusUnauthorized, err.Error())
		}
		if len(claims.Email) == 0 {
			return nil, utilerrors.NewBadRequest("the token doesn't contain the mandatory \"email\" claim")
		}

		adminKubeConfig, err := clusterProvider.GetAdminKubeconfigForUserCluster(ctx, cluster)
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
				return nil, utilerrors.New(http.StatusInternalServerError, fmt.Sprintf("unable to construct kubeconfig because couldn't find %s cluster entry in existing kubecfg", req.ClusterID))
			}

			// create cluster entry
			clientCmdCluster := clientcmdapi.NewCluster()
			clientCmdCluster.Server = clusterFromAdminKubeCfg.Server
			clientCmdCluster.CertificateAuthorityData = clusterFromAdminKubeCfg.CertificateAuthorityData
			oidcKubeCfg.Clusters[req.ClusterID] = clientCmdCluster

			// create auth entry
			clientCmdAuth := clientcmdapi.NewAuthInfo()
			clientCmdAuthProvider := &clientcmdapi.AuthProviderConfig{Config: map[string]string{}}
			clientCmdAuthProvider.Name = oidc
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
			oidcKubeCfg.Contexts[defaultCtx] = clientCmdCtx
			oidcKubeCfg.CurrentContext = defaultCtx
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
		return nil, utilerrors.NewBadRequest(fmt.Sprintf("bad request unexpected phase %d, expected phase %d, did you forget to set the phase while decoding the request?", req.phase, initialPhase))
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
	rsp.authCodeURL = oidcIssuer.AuthCodeURL(urlSafeState, oidcCfg.OfflineAccessAsScope, "", scopes...)

	return rsp, nil
}

func CreateOIDCKubeconfigSecretEndpoint(ctx context.Context, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, oidcIssuerVerifier auth.OIDCIssuerVerifier, oidcCfg common.OIDCConfiguration, req CreateOIDCKubeconfigReq) (interface{}, error) {
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

	// Override default redirect uri
	redirectURI, err := oidcIssuerVerifier.GetRedirectURI(req.request.URL.Path)
	if err != nil {
		return nil, err
	}

	// PHASE exchangeCode handles callback response from OIDC provider
	// and generates kubeconfig
	if req.phase == exchangeCodePhase {
		// validate the state
		if req.decodedState.Nonce != req.cookieNonceValue {
			return nil, utilerrors.NewBadRequest("incorrect value of state parameter: %s", req.decodedState.Nonce)
		}
		oidcTokens, err := oidcIssuer.Exchange(ctx, req.code, redirectURI)
		if err != nil {
			return nil, utilerrors.NewBadRequest("error while exchanging oidc code for token: %v", err)
		}
		if len(oidcTokens.RefreshToken) == 0 {
			return nil, utilerrors.NewBadRequest("the refresh token is missing but required, try setting/unsetting \"oidc-offline-access-as-scope\" command line flag")
		}

		claims, err := oidcVerifier.Verify(ctx, oidcTokens.IDToken)
		if err != nil {
			return nil, utilerrors.New(http.StatusUnauthorized, err.Error())
		}
		if len(claims.Email) == 0 {
			return nil, utilerrors.NewBadRequest("the token doesn't contain the mandatory \"email\" claim")
		}

		adminKubeConfig, err := clusterProvider.GetAdminKubeconfigForUserCluster(ctx, cluster)
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
				return nil, utilerrors.New(http.StatusInternalServerError, fmt.Sprintf("unable to construct kubeconfig because couldn't find %s cluster entry in existing kubecfg", req.ClusterID))
			}

			// create cluster entry
			clientCmdCluster := clientcmdapi.NewCluster()
			clientCmdCluster.Server = clusterFromAdminKubeCfg.Server
			clientCmdCluster.CertificateAuthorityData = clusterFromAdminKubeCfg.CertificateAuthorityData
			oidcKubeCfg.Clusters[req.ClusterID] = clientCmdCluster

			// create auth entry
			clientCmdAuth := clientcmdapi.NewAuthInfo()
			clientCmdAuthProvider := &clientcmdapi.AuthProviderConfig{Config: map[string]string{}}
			clientCmdAuthProvider.Name = oidc
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
			oidcKubeCfg.Contexts[defaultCtx] = clientCmdCtx
			oidcKubeCfg.CurrentContext = defaultCtx
		}

		// prepare final rsp that holds kubeconfig
		rsp := createOIDCKubeconfigRsp{}
		rsp.phase = kubeconfigGenerated
		rsp.secureCookieMode = oidcCfg.CookieSecureMode
		client, err := clusterProvider.GetAdminClientForUserCluster(ctx, cluster)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		if err := createKubeconfigSecret(ctx, client, oidcKubeCfg, claims.Email); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return rsp, nil
	}

	// PHASE initial handles request from the end-user that wants to authenticate
	// and kicksoff the process of kubeconfig generation
	if req.phase != initialPhase {
		return nil, utilerrors.NewBadRequest(fmt.Sprintf("bad request unexpected phase %d, expected phase %d, did you forget to set the phase while decoding the request?", req.phase, initialPhase))
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
	rsp.authCodeURL = oidcIssuer.AuthCodeURL(urlSafeState, oidcCfg.OfflineAccessAsScope, redirectURI, scopes...)

	return rsp, nil
}

func KubeconfigSecretName(userEmailID string) string {
	return fmt.Sprintf("kubeconfig-%s", userEmailID)
}

func createKubeconfigSecret(ctx context.Context, client ctrlruntimeclient.Client, config *clientcmdapi.Config, email string) error {
	// encode email address to unique ID for the secret name
	hasher := md5.New()
	hasher.Write([]byte(email))
	kubeconfigSecretName := KubeconfigSecretName(hex.EncodeToString(hasher.Sum(nil)))
	kubeconfig, err := clientcmd.Write(*config)
	if err != nil {
		return err
	}

	namespacedName := types.NamespacedName{Namespace: resources.KubeSystemNamespaceName, Name: kubeconfigSecretName}

	existingSecret := &corev1.Secret{}
	if err := client.Get(ctx, namespacedName, existingSecret); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to probe for secret %v: %w", namespacedName, err)
	}

	secretData := map[string][]byte{
		resources.KubeconfigSecretKey: kubeconfig,
	}

	// return if already exists
	if existingSecret.Name != "" {
		return nil
	}

	return createSecret(ctx, client, kubeconfigSecretName, email, secretData)
}

func createSecret(ctx context.Context, client ctrlruntimeclient.Client, name, email string, secretData map[string][]byte) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   resources.KubeSystemNamespaceName,
			Annotations: map[string]string{"user": email},
		},
		Type: corev1.SecretTypeOpaque,
		Data: secretData,
	}
	return client.Create(ctx, secret)
}

// CreateOIDCKubeconfigReq represent a request for creating kubeconfig for a cluster with OIDC credentials
// swagger:parameters createOIDCKubeconfig
type CreateOIDCKubeconfigReq struct {
	// in: query
	ClusterID string `json:"cluster_id,omitempty"`
	ProjectID string `json:"project_id,omitempty"`
	UserID    string `json:"user_id,omitempty"`

	// not exported so that they don't leak to swagger spec.
	// Embed the original request
	request          *http.Request
	code             string
	encodedState     string
	decodedState     OIDCState
	phase            int
	cookieNonceValue string
}

// OIDCState holds data that are send and retrieved from OIDC provider.
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
			return fmt.Errorf("the cookie can't be removed: %w", err)
		}
		return EncodeKubeconfig(c, w, &encodeKubeConifgResponse{clientCfg: rsp.oidcKubeConfig})
	}

	// handles initialPhase
	// redirects request to OpenID provider's consent page
	// and set cookie with nonce
	err = setCookie(w, rsp.nonce, rsp.secureCookieMode, cookieMaxAge)
	if err != nil {
		return fmt.Errorf("the cookie can't be created: %w", err)
	}
	w.Header().Add("Location", rsp.authCodeURL)
	w.Header().Add("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusSeeOther)
	return nil
}

func EncodeOIDCKubeconfigSecret(c context.Context, w http.ResponseWriter, response interface{}) (err error) {
	rsp := response.(createOIDCKubeconfigRsp)

	// handles kubeconfig Generated PHASE
	// it means that kubeconfig was generated and we need to properly encode it.
	if rsp.phase == kubeconfigGenerated {
		// clear cookie by setting MaxAge<0
		err = setCookie(w, "", rsp.secureCookieMode, -1)
		if err != nil {
			return fmt.Errorf("the cookie can't be removed: %w", err)
		}
		return nil
	}

	// handles initialPhase
	// redirects request to OpenID provider's consent page
	// and set cookie with nonce
	err = setCookie(w, rsp.nonce, rsp.secureCookieMode, cookieMaxAge)
	if err != nil {
		return fmt.Errorf("the cookie can't be created: %w", err)
	}
	w.Header().Add("Location", rsp.authCodeURL)
	w.Header().Add("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusSeeOther)
	return nil
}

func DecodeCreateOIDCKubeconfig(_ context.Context, r *http.Request) (interface{}, error) {
	req := CreateOIDCKubeconfigReq{}
	req.request = r

	// handle OIDC errors
	{
		errType := r.URL.Query().Get("error")
		errMessage := r.URL.Query().Get("error_description")
		if len(errMessage) != 0 {
			return nil, fmt.Errorf("OIDC provider error %s: %s", errType, errMessage)
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
			return nil, utilerrors.NewBadRequest("incorrect value of state parameter, expected url encoded value: %v", err)
		}
		rawState, err := base64.StdEncoding.DecodeString(unescapedState)
		if err != nil {
			return nil, utilerrors.NewBadRequest("incorrect value of state parameter, expected base64 encoded value: %v", err)
		}
		oidcState := OIDCState{}
		if err := json.Unmarshal(rawState, &oidcState); err != nil {
			return nil, utilerrors.NewBadRequest("incorrect value of state parameter, expected json encoded value: %v", err)
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
				return nil, utilerrors.NewBadRequest("incorrect value of cookie or cookie not set: %v", err)
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
		return nil, errors.New("the following query parameters cluster_id, project_id, user_id are mandatory, please make sure that all are set")
	}
	req.phase = initialPhase
	return req, nil
}

// GetUserID implements UserGetter interface.
func (r CreateOIDCKubeconfigReq) GetUserID() string {
	return r.UserID
}

// GetSeedCluster returns the SeedCluster object.
func (r CreateOIDCKubeconfigReq) GetSeedCluster() apiv1.SeedCluster {
	return apiv1.SeedCluster{
		ClusterID: r.ClusterID,
	}
}

// GetProjectID implements ProjectGetter interface.
func (r CreateOIDCKubeconfigReq) GetProjectID() string {
	return r.ProjectID
}

// setCookie add cookie with random string value.
func setCookie(w http.ResponseWriter, nonce string, secureMode bool, maxAge int) error {
	encoded, err := secureCookie.Encode(csrfCookieName, nonce)
	if err != nil {
		return fmt.Errorf("the encode cookie failed: %w", err)
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
		return nil, utilerrors.New(http.StatusInternalServerError, "no cluster provider in request")
	}
	privilegedClusterProvider, ok := ctx.Value(middleware.PrivilegedClusterProviderContextKey).(provider.PrivilegedClusterProvider)
	if !ok {
		return nil, utilerrors.New(http.StatusInternalServerError, "no privileged cluster provider in request")
	}
	userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)

	project, err := getProjectForOIDCEndpoint(ctx, userInfo, projectProvider, privilegedProjectProvider, projectID)
	if err != nil {
		return nil, err
	}

	if userInfo.IsAdmin {
		return privilegedClusterProvider.GetUnsecured(ctx, project, clusterID, nil)
	}

	return clusterProvider.Get(ctx, userInfo, clusterID, &provider.ClusterGetOptions{})
}

func getProjectForOIDCEndpoint(ctx context.Context, userInfo *provider.UserInfo, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, projectID string) (*kubermaticv1.Project, error) {
	if userInfo.IsAdmin {
		// get any project for admin
		return privilegedProjectProvider.GetUnsecured(ctx, projectID, &provider.ProjectGetOptions{IncludeUninitialized: true})
	}

	return projectProvider.Get(ctx, userInfo, projectID, &provider.ProjectGetOptions{IncludeUninitialized: true})
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
