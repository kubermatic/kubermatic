/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package serviceaccount

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"unicode/utf8"

	"github.com/go-kit/kit/endpoint"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticapiv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/serviceaccount"
	"k8c.io/kubermatic/v2/pkg/util/errors"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/rand"
)

// CreateTokenEndpoint creates a token for the given main service account
func CreateTokenEndpoint(serviceAccountProvider provider.ServiceAccountProvider, privilegedServiceAccountTokenProvider provider.PrivilegedServiceAccountTokenProvider, tokenAuthenticator serviceaccount.TokenAuthenticator, tokenGenerator serviceaccount.TokenGenerator, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(addTokenReq)
		err := req.Validate()
		if err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, err
		}

		sa, err := serviceAccountProvider.GetMainServiceAccount(userInfo, req.ServiceAccountID, nil)
		if err != nil {
			return nil, err
		}

		// check if token name is already reserved for service account
		existingTokenList, err := listSAToken(privilegedServiceAccountTokenProvider, sa, "", req.Body.Name)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		if len(existingTokenList) > 0 {
			return nil, errors.NewAlreadyExists("token", req.Body.Name)
		}

		tokenID := rand.String(10)

		token, err := tokenGenerator.Generate(serviceaccount.Claims(sa.Spec.Email, "", tokenID))
		if err != nil {
			return nil, errors.New(http.StatusInternalServerError, "can not generate token data")
		}

		secret, err := createSAToken(privilegedServiceAccountTokenProvider, sa, req.Body.Name, tokenID, token)
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

func listSAToken(privilegedServiceAccountTokenProvider provider.PrivilegedServiceAccountTokenProvider, sa *kubermaticapiv1.User, tokenID, tokenName string) ([]*v1.Secret, error) {
	options := &provider.ServiceAccountTokenListOptions{}
	options.TokenID = tokenID
	options.TokenName = tokenName

	options.ServiceAccountID = sa.Name

	return privilegedServiceAccountTokenProvider.ListUnsecured(options)
}

func createSAToken(privilegedServiceAccountTokenProvider provider.PrivilegedServiceAccountTokenProvider, sa *kubermaticapiv1.User, tokenName, tokenID, tokenData string) (*v1.Secret, error) {

	return privilegedServiceAccountTokenProvider.CreateUnsecured(sa, "", tokenName, tokenID, tokenData)
}

// addTokenReq defines HTTP request for addTokenToMainServiceAccount
// swagger:parameters addTokenToMainServiceAccount
type addTokenReq struct {
	commonTokenReq
	// in: body
	Body apiv1.ServiceAccountToken
}

type commonTokenReq struct {
	serviceAccountIDReq
}

// Validate validates addTokenReq request
func (r addTokenReq) Validate() error {
	if len(r.Body.Name) == 0 || len(r.ServiceAccountID) == 0 {
		return fmt.Errorf("the name and service account ID cannot be empty")
	}
	if utf8.RuneCountInString(r.Body.Name) > 50 {
		return fmt.Errorf("the name is too long, max 50 chars")
	}

	return nil
}

// Validate validates commonTokenReq request
func (r commonTokenReq) Validate() error {
	if len(r.ServiceAccountID) == 0 {
		return fmt.Errorf("service account ID cannot be empty")
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

	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}

	return req, nil
}

// DecodeTokenReq  decodes an HTTP request into addReq
func DecodeTokenReq(c context.Context, r *http.Request) (interface{}, error) {
	var req commonTokenReq

	saIDReq, err := decodeServiceAccountIDReq(c, r)
	if err != nil {
		return nil, err
	}
	req.ServiceAccountID = saIDReq.ServiceAccountID

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
