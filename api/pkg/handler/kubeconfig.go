package handler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/go-kit/kit/endpoint"

	"github.com/kubermatic/kubermatic/api/pkg/provider"
	kcerrors "github.com/kubermatic/kubermatic/api/pkg/util/errors"

	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func getClusterKubeconfig(projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GetClusterReq)
		clusterProvider := ctx.Value(clusterProviderContextKey).(provider.ClusterProvider)
		userInfo := ctx.Value(userInfoContextKey).(*provider.UserInfo)
		_, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		cluster, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{})
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}
		return clusterProvider.GetAdminKubeconfigForCustomerCluster(cluster)
	}
}

func createOIDCKubeconfig(projectProvider provider.ProjectProvider, oidcIssuerVerifier OIDCIssuerVerifier, oidcCfg OIDCConfiguration) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		oidcIssuer := oidcIssuerVerifier.(OIDCIssuer)
		oidcVerifier := oidcIssuerVerifier.(OIDCVerifier)
		req := request.(CreateOIDCKubeconfigReq)
		clusterProvider := ctx.Value(clusterProviderContextKey).(provider.ClusterProvider)
		userInfo := ctx.Value(userInfoContextKey).(*provider.UserInfo)

		_, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}
		cluster, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{})
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		// PHASE exchangeCode handles callback response from OIDC provider
		// and generates kubeconfig
		if req.phase == exchangeCodePhase {
			// TODO: validate the state
			if req.decodedState.Nonce != "nonce=TODO" {
				return nil, kcerrors.NewBadRequest("incorrect value of state parameter = %s", req.decodedState.Nonce)
			}
			oidcTokens, err := oidcIssuer.Exchange(ctx, req.code)
			if err != nil {
				return nil, kcerrors.NewBadRequest("error while exchaning oidc code for token = %v", err)
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
				return nil, kubernetesErrorToHTTPError(err)
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
					return nil, kcerrors.New(http.StatusInternalServerError, fmt.Sprintf("unable to construct kubeconfig because couldn't find %s cluster enty in existing kubecfg", req.ClusterID))
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
				clientCmdAuthProvider.Config["id-token"] = oidcTokens.AccessToken
				clientCmdAuthProvider.Config["refresh-token"] = oidcTokens.RefreshToken
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
			return rsp, nil
		}

		// PHASE initial handles request from and end-user that want to authenticate
		// and kickoff the process of kubeconfig generation
		if req.phase != initialPhase {
			return nil, kcerrors.NewBadRequest("bad request unexpected ")
		}

		// TODO: pass nonce
		rsp := createOIDCKubeconfigRsp{}
		// TODO: Define a proper list of scopes, is offline_access only a scope ? (you can set it also via clientlib)
		scopes := []string{"openid", "email", "offline_access"}
		oidcState := state{
			Nonce:      "nonce=TODO",
			ClusterID:  req.ClusterID,
			ProjectID:  req.ProjectID,
			UserID:     req.UserID,
			Datacenter: req.Datacenter,
		}
		rawState, err := json.Marshal(oidcState)
		if err != nil {
			return nil, err
		}
		encodedState := base64.StdEncoding.EncodeToString(rawState)
		urlSafeState := url.QueryEscape(encodedState)
		rsp.authCodeURL = oidcIssuer.AuthCodeURL(urlSafeState, scopes...)

		return rsp, nil
	}
}

func encodeKubeconfig(c context.Context, w http.ResponseWriter, response interface{}) (err error) {
	cfg := response.(*clientcmdapi.Config)

	filename := "kubeconfig"

	if len(cfg.Contexts) > 0 {
		filename = fmt.Sprintf("%s-%s", filename, cfg.Contexts[cfg.CurrentContext].Cluster)
	}

	w.Header().Set("Content-Type", "application/yaml")
	w.Header().Set("Content-disposition", fmt.Sprintf("attachment; filename=%s", filename))
	w.Header().Add("Cache-Control", "no-cache")

	b, err := clientcmd.Write(*cfg)
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}

type createOIDCKubeconfigRsp struct {
	// authCodeURL holds a URL to OpenID provider's consent page that asks for permissions for the required scopes explicitly.
	authCodeURL string
	// phase tells encoding function how to handle response
	phase int
	// oidcKubeConfig holds not serialized kubeconfig
	oidcKubeConfig *clientcmdapi.Config
}

func encodeKubeconfigDoINeddAcditional(c context.Context, w http.ResponseWriter, response interface{}) (err error) {
	rsp := response.(createOIDCKubeconfigRsp)

	// handles kubeconfigGenerated PHASE
	// it means that kubeconfig was generated and we need to properly encode it.
	if rsp.phase == kubeconfigGenerated {
		return encodeKubeconfig(c, w, rsp.oidcKubeConfig)
	}

	// handles initialPhase
	// redirects request to OpenID provider's consent page
	w.Header().Add("Location", rsp.authCodeURL)
	w.Header().Add("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusSeeOther)
	return nil
}

func decodeGetClusterKubeconfig(c context.Context, r *http.Request) (interface{}, error) {
	req, err := decodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}

	return req, nil
}

// UserIDGetter knows how to get user ID from the request
type UserIDGetter interface {
	GetUserID() string
}

const (
	initialPhase        = iota
	exchangeCodePhase   = iota
	kubeconfigGenerated = iota
)

// state holds data that are send and retrieved from OIDC provider
type state struct {
	// nonce a random string that binds requests / responses of API server and OIDC provider
	// see https://tools.ietf.org/html/rfc6749#section-10.12
	Nonce     string `json:"nonce"`
	ClusterID string `json:"cluster_id"`
	ProjectID string `json:"project_id"`
	// UserID holds the ID of the user on behalf of which the request is being handled.
	UserID     string `json:"user_id"`
	Datacenter string `json:"datacenter"`
}

// CreateOIDCKubeconfigReq represent a request for creating kubeconfig for a cluster with OIDC credentials
// swagger:parameters createOIDCKubeconfig
type CreateOIDCKubeconfigReq struct {
	// in: query
	ClusterID  string
	ProjectID  string
	UserID     string
	Datacenter string

	// not exported so that they don't leak to swagger spec.
	code         string
	encodedState string
	decodedState state
	phase        int
}

func decodeCreateOIDCKubeconfig(c context.Context, r *http.Request) (interface{}, error) {
	req := CreateOIDCKubeconfigReq{}

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
		oidcState := state{}
		if err := json.Unmarshal(rawState, &oidcState); err != nil {
			return nil, kcerrors.NewBadRequest("incorrect value of state parameter, expected json encoded value, err = %v", err)
		}
		req.phase = exchangeCodePhase
		req.Datacenter = oidcState.Datacenter
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
	req.Datacenter = r.URL.Query().Get("datacenter")
	if len(req.ClusterID) == 0 || len(req.ProjectID) == 0 || len(req.UserID) == 0 || len(req.Datacenter) == 0 {
		return nil, errors.New("the following query parameters cluster_id, project_id, user_id and datacenter are mandatory, please make sure that all are set")
	}
	req.phase = initialPhase
	return req, nil
}

// GetUserID implements UserGetter interface
func (r CreateOIDCKubeconfigReq) GetUserID() string {
	return r.UserID
}

// GetDC implements DCGetter interface
func (r CreateOIDCKubeconfigReq) GetDC() string {
	return r.Datacenter
}

// GetProjectID implements ProjectGetter interface
func (r CreateOIDCKubeconfigReq) GetProjectID() string {
	return r.ProjectID
}
