package serviceaccount

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"unicode/utf8"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/serviceaccount"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/rand"
)

// CreateTokenEndpoint creates a token for the given service account
func CreateTokenEndpoint(projectProvider provider.ProjectProvider, serviceAccountProvider provider.ServiceAccountProvider, serviceAccountTokenProvider provider.ServiceAccountTokenProvider, tokenAuthenticator serviceaccount.TokenAuthenticator, tokenGenerator serviceaccount.TokenGenerator, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(addTokenReq)
		err := req.Validate()
		if err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}
		userInfo, err := userInfoGetter(ctx, req.ProjectID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		project, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		sa, err := serviceAccountProvider.Get(userInfo, req.ServiceAccountID, &provider.ServiceAccountGetOptions{RemovePrefix: false})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		// check if token name is already reserved for service account
		existingTokenList, err := serviceAccountTokenProvider.List(userInfo, project, sa, &provider.ServiceAccountTokenListOptions{TokenID: req.Body.Name})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		if len(existingTokenList) > 0 {
			return nil, errors.NewAlreadyExists("token", req.Body.Name)
		}

		tokenID := rand.String(10)

		token, err := tokenGenerator.Generate(serviceaccount.Claims(sa.Spec.Email, project.Name, tokenID))
		if err != nil {
			return nil, errors.New(http.StatusInternalServerError, "can not generate token data")
		}

		secret, err := serviceAccountTokenProvider.Create(userInfo, sa, project.Name, req.Body.Name, tokenID, token)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		externalToken, err := convertInternalTokenToPrivateExternal(secret, tokenAuthenticator)
		if err != nil {
			return nil, errors.New(http.StatusInternalServerError, err.Error())
		}

		return externalToken, nil
	}
}

// ListTokenEndpoint gets token for the service account
func ListTokenEndpoint(projectProvider provider.ProjectProvider, serviceAccountProvider provider.ServiceAccountProvider, serviceAccountTokenProvider provider.ServiceAccountTokenProvider, tokenAuthenticator serviceaccount.TokenAuthenticator, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		resultList := make([]*apiv1.PublicServiceAccountToken, 0)
		req := request.(commonTokenReq)
		err := req.Validate()
		if err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}
		userInfo, err := userInfoGetter(ctx, req.ProjectID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		project, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		sa, err := serviceAccountProvider.Get(userInfo, req.ServiceAccountID, &provider.ServiceAccountGetOptions{RemovePrefix: false})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		existingSecretList, err := serviceAccountTokenProvider.List(userInfo, project, sa, nil)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		var errorList []string
		for _, secret := range existingSecretList {

			externalToken, err := convertInternalTokenToPublicExternal(secret, tokenAuthenticator)
			if err != nil {
				errorList = append(errorList, err.Error())
				continue
			}
			resultList = append(resultList, externalToken)
		}

		if len(errorList) > 0 {
			return nil, errors.NewWithDetails(http.StatusInternalServerError, "failed to get some service account tokens, please examine details field for more info", errorList)
		}

		return resultList, nil
	}
}

// UpdateTokenEndpoint updates and regenerates the token for the given service account
func UpdateTokenEndpoint(projectProvider provider.ProjectProvider, serviceAccountProvider provider.ServiceAccountProvider, serviceAccountTokenProvider provider.ServiceAccountTokenProvider, tokenAuthenticator serviceaccount.TokenAuthenticator, tokenGenerator serviceaccount.TokenGenerator, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateTokenReq)
		err := req.Validate()
		if err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}
		userInfo, err := userInfoGetter(ctx, req.ProjectID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		secret, err := updateToken(projectProvider, serviceAccountProvider, serviceAccountTokenProvider, userInfo, tokenGenerator, req.ProjectID, req.ServiceAccountID, req.TokenID, req.Body.Name, true)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		externalToken, err := convertInternalTokenToPrivateExternal(secret, tokenAuthenticator)
		if err != nil {
			return nil, errors.New(http.StatusInternalServerError, err.Error())
		}

		return externalToken, nil
	}
}

// PatchTokenEndpoint patches the token name
func PatchTokenEndpoint(projectProvider provider.ProjectProvider, serviceAccountProvider provider.ServiceAccountProvider, serviceAccountTokenProvider provider.ServiceAccountTokenProvider, tokenAuthenticator serviceaccount.TokenAuthenticator, tokenGenerator serviceaccount.TokenGenerator, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(patchTokenReq)
		err := req.Validate()
		if err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}
		userInfo, err := userInfoGetter(ctx, req.ProjectID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		tokenReq := &apiv1.PublicServiceAccountToken{}
		if err := json.Unmarshal(req.Body, tokenReq); err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}
		if len(tokenReq.Name) == 0 {
			return nil, errors.NewBadRequest("new name can not be empty")
		}

		secret, err := updateToken(projectProvider, serviceAccountProvider, serviceAccountTokenProvider, userInfo, tokenGenerator, req.ProjectID, req.ServiceAccountID, req.TokenID, tokenReq.Name, false)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		externalToken, err := convertInternalTokenToPublicExternal(secret, tokenAuthenticator)
		if err != nil {
			return nil, errors.New(http.StatusInternalServerError, err.Error())
		}

		return externalToken, nil
	}
}

// DeleteTokenEndpoint deletes the token from service account
func DeleteTokenEndpoint(projectProvider provider.ProjectProvider, serviceAccountProvider provider.ServiceAccountProvider, serviceAccountTokenProvider provider.ServiceAccountTokenProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(deleteTokenReq)
		err := req.Validate()
		if err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}
		userInfo, err := userInfoGetter(ctx, req.ProjectID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		_, err = projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		_, err = serviceAccountProvider.Get(userInfo, req.ServiceAccountID, nil)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		if err := serviceAccountTokenProvider.Delete(userInfo, req.TokenID); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return nil, nil
	}
}

func updateToken(projectProvider provider.ProjectProvider, serviceAccountProvider provider.ServiceAccountProvider,
	serviceAccountTokenProvider provider.ServiceAccountTokenProvider, userInfo *provider.UserInfo, tokenGenerator serviceaccount.TokenGenerator,
	projectID, saID, tokenID, newName string, regenerateToken bool) (*v1.Secret, error) {

	project, err := projectProvider.Get(userInfo, projectID, &provider.ProjectGetOptions{})
	if err != nil {
		return nil, err
	}

	sa, err := serviceAccountProvider.Get(userInfo, saID, &provider.ServiceAccountGetOptions{RemovePrefix: false})
	if err != nil {
		return nil, err
	}

	existingSecret, err := serviceAccountTokenProvider.Get(userInfo, tokenID)
	if err != nil {
		return nil, err
	}
	existingName, ok := existingSecret.Labels["name"]
	if !ok {
		return nil, fmt.Errorf("can not find token name in secret %s", existingSecret.Name)
	}

	if newName == existingName && !regenerateToken {
		return existingSecret, nil
	}

	if newName != existingName {
		// check if token name is already reserved for service account
		existingTokenList, err := serviceAccountTokenProvider.List(userInfo, project, sa, &provider.ServiceAccountTokenListOptions{TokenID: newName})
		if err != nil {
			return nil, err
		}
		if len(existingTokenList) > 0 {
			return nil, errors.NewAlreadyExists("token", newName)
		}
		existingSecret.Labels["name"] = newName
	}

	if regenerateToken {
		token, err := tokenGenerator.Generate(serviceaccount.Claims(sa.Spec.Email, project.Name, existingSecret.Name))
		if err != nil {
			return nil, fmt.Errorf("can not generate token data")
		}

		existingSecret.Data["token"] = []byte(token)
	}

	secret, err := serviceAccountTokenProvider.Update(userInfo, existingSecret)
	if err != nil {
		return nil, err
	}

	return secret, nil
}

// addTokenReq defines HTTP request for addTokenToServiceAccount
// swagger:parameters addTokenToServiceAccount
type addTokenReq struct {
	commonTokenReq
	// in: body
	Body apiv1.ServiceAccountToken
}

// commonTokenReq defines HTTP request for listServiceAccountTokens
// swagger:parameters listServiceAccountTokens
type commonTokenReq struct {
	common.ProjectReq
	serviceAccountIDReq
}

// tokenIDReq represents a request that contains the token ID in the path
type tokenIDReq struct {
	// in: path
	TokenID string `json:"token_id"`
}

// updateTokenReq defines HTTP request for updateServiceAccountToken
// swagger:parameters updateServiceAccountToken
type updateTokenReq struct {
	commonTokenReq
	tokenIDReq
	// in: body
	Body apiv1.PublicServiceAccountToken
}

// patchTokenReq defines HTTP request for patchServiceAccountToken
// swagger:parameters patchServiceAccountToken
type patchTokenReq struct {
	commonTokenReq
	tokenIDReq
	// in: body
	Body []byte
}

// deleteTokenReq defines HTTP request for deleteServiceAccountToken
// swagger:parameters deleteServiceAccountToken
type deleteTokenReq struct {
	commonTokenReq
	tokenIDReq
}

// Validate validates addTokenReq request
func (r addTokenReq) Validate() error {
	if len(r.Body.Name) == 0 || len(r.ProjectID) == 0 || len(r.ServiceAccountID) == 0 {
		return fmt.Errorf("the name, service account ID and project ID cannot be empty")
	}
	if utf8.RuneCountInString(r.Body.Name) > 50 {
		return fmt.Errorf("the name is too long, max 50 chars")
	}

	return nil
}

// Validate validates commonTokenReq request
func (r commonTokenReq) Validate() error {
	if len(r.ProjectID) == 0 || len(r.ServiceAccountID) == 0 {
		return fmt.Errorf("service account ID and project ID cannot be empty")
	}

	return nil
}

// Validate validates updateTokenReq request
func (r updateTokenReq) Validate() error {
	if err := r.commonTokenReq.Validate(); err != nil {
		return err
	}
	if len(r.TokenID) == 0 {
		return fmt.Errorf("token ID cannot be empty")
	}
	if len(r.Body.Name) == 0 {
		return fmt.Errorf("new name can not be empty")
	}
	if r.TokenID != r.Body.ID {
		return fmt.Errorf("token ID mismatch, you requested to update token = %s but body contains token = %s", r.TokenID, r.Body.ID)
	}

	return nil
}

// Validate validates updateTokenReq request
func (r patchTokenReq) Validate() error {
	if err := r.commonTokenReq.Validate(); err != nil {
		return err
	}
	if len(r.TokenID) == 0 {
		return fmt.Errorf("token ID cannot be empty")
	}
	if len(r.Body) == 0 {
		return fmt.Errorf("body can not be empty")
	}

	return nil
}

// Validate validates updateTokenReq request
func (r deleteTokenReq) Validate() error {
	if err := r.commonTokenReq.Validate(); err != nil {
		return err
	}
	if len(r.TokenID) == 0 {
		return fmt.Errorf("token ID cannot be empty")
	}

	return nil
}

// DecodeAddReq  decodes an HTTP request into addReq
func DecodeAddTokenReq(c context.Context, r *http.Request) (interface{}, error) {
	var req addTokenReq

	rawReq, err := DecodeTokenReq(c, r)
	if err != nil {
		return nil, err
	}
	tokenReq := rawReq.(commonTokenReq)
	req.ServiceAccountID = tokenReq.ServiceAccountID
	req.ProjectID = tokenReq.ProjectID

	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}

	return req, nil
}

// DecodeTokenReq  decodes an HTTP request into addReq
func DecodeTokenReq(c context.Context, r *http.Request) (interface{}, error) {
	var req commonTokenReq

	prjReq, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, err

	}
	req.ProjectReq = prjReq.(common.ProjectReq)

	saIDReq, err := decodeServiceAccountIDReq(c, r)
	if err != nil {
		return nil, err
	}
	req.ServiceAccountID = saIDReq.ServiceAccountID

	return req, nil
}

// DecodeUpdateTokenReq  decodes an HTTP request into updateTokenReq
func DecodeUpdateTokenReq(c context.Context, r *http.Request) (interface{}, error) {
	var req updateTokenReq

	rawReq, err := DecodeTokenReq(c, r)
	if err != nil {
		return nil, err
	}
	tokenReq := rawReq.(commonTokenReq)
	req.ServiceAccountID = tokenReq.ServiceAccountID
	req.ProjectID = tokenReq.ProjectID

	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}

	tokenID, err := decodeTokenIDReq(c, r)
	if err != nil {
		return nil, err
	}

	req.TokenID = tokenID.TokenID

	return req, nil
}

// DecodePatchTokenReq  decodes an HTTP request into patchTokenReq
func DecodePatchTokenReq(c context.Context, r *http.Request) (interface{}, error) {
	var req patchTokenReq

	rawReq, err := DecodeTokenReq(c, r)
	if err != nil {
		return nil, err
	}
	tokenReq := rawReq.(commonTokenReq)
	req.ServiceAccountID = tokenReq.ServiceAccountID
	req.ProjectID = tokenReq.ProjectID

	req.Body, err = ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	tokenID, err := decodeTokenIDReq(c, r)
	if err != nil {
		return nil, err
	}

	req.TokenID = tokenID.TokenID

	return req, nil
}

// DecodeDeleteTokenReq  decodes an HTTP request into deleteTokenReq
func DecodeDeleteTokenReq(c context.Context, r *http.Request) (interface{}, error) {
	var req deleteTokenReq

	rawReq, err := DecodeTokenReq(c, r)
	if err != nil {
		return nil, err
	}
	tokenReq := rawReq.(commonTokenReq)
	req.ServiceAccountID = tokenReq.ServiceAccountID
	req.ProjectID = tokenReq.ProjectID

	tokenID, err := decodeTokenIDReq(c, r)
	if err != nil {
		return nil, err
	}

	req.TokenID = tokenID.TokenID

	return req, nil
}

func decodeTokenIDReq(c context.Context, r *http.Request) (tokenIDReq, error) {
	var req tokenIDReq

	tokenID, ok := mux.Vars(r)["token_id"]
	if !ok {
		return req, fmt.Errorf("'token_id' parameter is required")
	}
	req.TokenID = tokenID

	return req, nil
}

func convertInternalTokenToPrivateExternal(internal *v1.Secret, authenticator serviceaccount.TokenAuthenticator) (*apiv1.ServiceAccountToken, error) {
	externalToken := &apiv1.ServiceAccountToken{}
	public, err := convertInternalTokenToPublicExternal(internal, authenticator)
	if err != nil {
		return nil, err
	}
	externalToken.PublicServiceAccountToken = *public
	token, ok := internal.Data["token"]
	if !ok {
		return nil, fmt.Errorf("can not find token data in secret %s", internal.Name)
	}
	externalToken.Token = string(token)
	return externalToken, nil
}

func convertInternalTokenToPublicExternal(internal *v1.Secret, authenticator serviceaccount.TokenAuthenticator) (*apiv1.PublicServiceAccountToken, error) {
	externalToken := &apiv1.PublicServiceAccountToken{}
	token, ok := internal.Data["token"]
	if !ok {
		return nil, fmt.Errorf("can not find token data")
	}

	publicClaim, _, err := authenticator.Authenticate(string(token))
	if err != nil {
		return nil, fmt.Errorf("unable to create a token for %s due to %v", internal.Name, err)
	}

	externalToken.Expiry = apiv1.NewTime(publicClaim.Expiry.Time())
	externalToken.ID = internal.Name
	name, ok := internal.Labels["name"]
	if !ok {
		return nil, fmt.Errorf("can not find token name in secret %s", internal.Name)
	}
	externalToken.Name = name

	externalToken.CreationTimestamp = apiv1.NewTime(internal.CreationTimestamp.Time)
	return externalToken, nil
}
