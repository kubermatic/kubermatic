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

package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-kit/kit/endpoint"
	transporthttp "github.com/go-kit/kit/transport/http"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/auth"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	kubermaticcontext "github.com/kubermatic/kubermatic/api/pkg/util/context"
	k8cerrors "github.com/kubermatic/kubermatic/api/pkg/util/errors"
	"github.com/kubermatic/kubermatic/api/pkg/util/hash"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
)

const (
	datacenterContextKey kubermaticcontext.Key = "datacenter"

	// RawTokenContextKey key under which the current token (OpenID ID Token) is kept in the ctx
	rawTokenContextKey kubermaticcontext.Key = "raw-auth-token"

	// noTokenFoundKey key under which an error is kept when no suitable token has been found in a request
	noTokenFoundKey kubermaticcontext.Key = "no-token-found"

	// ClusterProviderContextKey key under which the current ClusterProvider is kept in the ctx
	ClusterProviderContextKey kubermaticcontext.Key = "cluster-provider"

	// PrivilegedClusterProviderContextKey key under which the current PrivilegedClusterProvider is kept in the ctx
	PrivilegedClusterProviderContextKey kubermaticcontext.Key = "privileged-cluster-provider"

	// UserInfoContextKey key under which the current UserInfoExtractor is kept in the ctx
	UserInfoContextKey kubermaticcontext.Key = "user-info"

	// AuthenticatedUserContextKey key under which the current User (from OIDC provider) is kept in the ctx
	AuthenticatedUserContextKey kubermaticcontext.Key = "authenticated-user"

	// AddonProviderContextKey key under which the current AddonProvider is kept in the ctx
	AddonProviderContextKey kubermaticcontext.Key = "addon-provider"

	// PrivilegedAddonProviderContextKey key under which the current PrivilegedAddonProvider is kept in the ctx
	PrivilegedAddonProviderContextKey kubermaticcontext.Key = "privileged-addon-provider"

	UserCRContextKey = kubermaticcontext.UserCRContextKey
)

//DCGetter defines functionality to retrieve a datacenter name
type dCGetter interface {
	GetDC() string
}

// SetClusterProvider is a middleware that injects the current ClusterProvider into the ctx
func SetClusterProvider(clusterProviderGetter provider.ClusterProviderGetter, seedsGetter provider.SeedsGetter) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			clusterProvider, ctx, err := getClusterProvider(ctx, request, seedsGetter, clusterProviderGetter)
			if err != nil {
				return nil, err
			}

			ctx = context.WithValue(ctx, ClusterProviderContextKey, clusterProvider)
			return next(ctx, request)
		}
	}
}

// SetPrivilegedClusterProvider is a middleware that injects the current ClusterProvider into the ctx
func SetPrivilegedClusterProvider(clusterProviderGetter provider.ClusterProviderGetter, seedsGetter provider.SeedsGetter) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			clusterProvider, ctx, err := getClusterProvider(ctx, request, seedsGetter, clusterProviderGetter)
			if err != nil {
				return nil, err
			}

			privilegedClusterProvider := clusterProvider.(provider.PrivilegedClusterProvider)
			ctx = context.WithValue(ctx, ClusterProviderContextKey, clusterProvider)
			ctx = context.WithValue(ctx, PrivilegedClusterProviderContextKey, privilegedClusterProvider)
			return next(ctx, request)
		}
	}
}

// UserSaver is a middleware that checks if authenticated user already exists in the database
// next it creates/retrieve an internal object (kubermaticv1.User) and stores it the ctx under UserCRContexKey
func UserSaver(userProvider provider.UserProvider) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			rawAuthenticatesUser := ctx.Value(AuthenticatedUserContextKey)
			if rawAuthenticatesUser == nil {
				return nil, k8cerrors.New(http.StatusInternalServerError, "no user in context found")
			}
			authenticatedUser := rawAuthenticatesUser.(apiv1.User)

			user, err := userProvider.UserByEmail(authenticatedUser.Email)
			if err != nil {
				if err != provider.ErrNotFound {
					return nil, common.KubernetesErrorToHTTPError(err)
				}
				// handling ErrNotFound
				user, err = userProvider.CreateUser(authenticatedUser.ID, authenticatedUser.Name, authenticatedUser.Email)
				if err != nil {
					if !kerrors.IsAlreadyExists(err) {
						return nil, common.KubernetesErrorToHTTPError(err)
					}
					if user, err = userProvider.UserByEmail(authenticatedUser.Email); err != nil {
						return nil, common.KubernetesErrorToHTTPError(err)
					}
				}
			}
			return next(context.WithValue(ctx, kubermaticcontext.UserCRContextKey, user), request)
		}
	}
}

// UserInfoUnauthorized tries to build userInfo for not authenticated (token) user
// instead it reads the user_id from the request and finds the associated user in the database
func UserInfoUnauthorized(userProjectMapper provider.ProjectMemberMapper, userProvider provider.UserProvider) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			userIDGetter, ok := request.(common.UserIDGetter)
			if !ok {
				return nil, k8cerrors.NewBadRequest("you can only use userInfoMiddlewareUnauthorized for endpoints that accepts user ID")
			}
			prjIDGetter, ok := request.(common.ProjectIDGetter)
			if !ok {
				return nil, k8cerrors.NewBadRequest("you can only use userInfoMiddlewareUnauthorized for endpoints that accepts project ID")
			}
			userID := userIDGetter.GetUserID()
			projectID := prjIDGetter.GetProjectID()
			user, err := userProvider.UserByID(userID)
			if err != nil {
				return nil, common.KubernetesErrorToHTTPError(err)
			}

			if user.Spec.IsAdmin {
				uInfo := &provider.UserInfo{Email: user.Spec.Email, IsAdmin: true}
				return next(context.WithValue(ctx, UserInfoContextKey, uInfo), request)
			}
			uInfo, err := createUserInfo(user, projectID, userProjectMapper)
			if err != nil {
				return nil, common.KubernetesErrorToHTTPError(err)
			}
			return next(context.WithValue(ctx, UserInfoContextKey, uInfo), request)
		}
	}
}

// TokenVerifier knows how to verify a token from the incoming request
func TokenVerifier(tokenVerifier auth.TokenVerifier) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			if rawTokenNotFoundErr := ctx.Value(noTokenFoundKey); rawTokenNotFoundErr != nil {
				tokenNotFoundErr, ok := rawTokenNotFoundErr.(error)
				if !ok {
					return nil, k8cerrors.NewNotAuthorized()
				}
				return nil, k8cerrors.NewWithDetails(http.StatusUnauthorized, "not authorized", []string{tokenNotFoundErr.Error()})
			}

			t := ctx.Value(rawTokenContextKey)
			token, ok := t.(string)
			if !ok || token == "" {
				return nil, k8cerrors.NewNotAuthorized()
			}

			verifyCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()
			claims, err := tokenVerifier.Verify(verifyCtx, token)
			if err != nil {
				return nil, k8cerrors.New(http.StatusUnauthorized, fmt.Sprintf("access denied due to an invalid token, details = %v", err))
			}

			if claims.Subject == "" {
				return nil, k8cerrors.NewNotAuthorized()
			}

			id, err := hash.GetUserID(claims.Subject)
			if err != nil {
				return nil, k8cerrors.NewNotAuthorized()
			}

			user := apiv1.User{
				ObjectMeta: apiv1.ObjectMeta{
					ID:   id,
					Name: claims.Name,
				},
				Email: claims.Email,
			}

			if user.ID == "" {
				return nil, k8cerrors.NewNotAuthorized()
			}

			return next(context.WithValue(ctx, AuthenticatedUserContextKey, user), request)
		}
	}
}

// Addons is a middleware that injects the current AddonProvider into the ctx
func Addons(addonProviderGetter provider.AddonProviderGetter, seedsGetter provider.SeedsGetter) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			seedName := request.(dCGetter).GetDC()
			addonProvider, err := getAddonProvider(addonProviderGetter, seedsGetter, seedName)
			if err != nil {
				return nil, err
			}
			ctx = context.WithValue(ctx, AddonProviderContextKey, addonProvider)
			return next(ctx, request)
		}
	}
}

// PrivilegedAddons is a middleware that injects the current PrivilegedAddonProvider into the ctx
func PrivilegedAddons(addonProviderGetter provider.AddonProviderGetter, seedsGetter provider.SeedsGetter) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			seedName := request.(dCGetter).GetDC()
			addonProvider, err := getAddonProvider(addonProviderGetter, seedsGetter, seedName)
			if err != nil {
				return nil, err
			}
			privilegedAddonProvider := addonProvider.(provider.PrivilegedAddonProvider)
			ctx = context.WithValue(ctx, PrivilegedAddonProviderContextKey, privilegedAddonProvider)
			return next(ctx, request)
		}
	}
}

func getAddonProvider(addonProviderGetter provider.AddonProviderGetter, seedsGetter provider.SeedsGetter, seedName string) (provider.AddonProvider, error) {
	seeds, err := seedsGetter()
	if err != nil {
		return nil, err
	}

	seed, found := seeds[seedName]
	if !found {
		return nil, fmt.Errorf("couldn't find seed %q", seedName)
	}

	return addonProviderGetter(seed)
}

// TokenExtractor knows how to extract a token from the incoming request
func TokenExtractor(o auth.TokenExtractor) transporthttp.RequestFunc {
	return func(ctx context.Context, r *http.Request) context.Context {
		token, err := o.Extract(r)
		if err != nil {
			return context.WithValue(ctx, noTokenFoundKey, err)
		}
		return context.WithValue(ctx, rawTokenContextKey, token)
	}
}

func createUserInfo(user *kubermaticapiv1.User, projectID string, userProjectMapper provider.ProjectMemberMapper) (*provider.UserInfo, error) {
	var group string
	if projectID != "" {
		var err error
		group, err = userProjectMapper.MapUserToGroup(user.Spec.Email, projectID)
		if err != nil {
			return nil, err
		}
	}

	return &provider.UserInfo{Email: user.Spec.Email, Group: group}, nil
}

func getClusterProvider(ctx context.Context, request interface{}, seedsGetter provider.SeedsGetter, clusterProviderGetter provider.ClusterProviderGetter) (provider.ClusterProvider, context.Context, error) {
	getter, ok := request.(dCGetter)
	if !ok {
		return nil, nil, fmt.Errorf("request is no dcGetter")
	}
	seeds, err := seedsGetter()
	if err != nil {
		return nil, ctx, k8cerrors.New(http.StatusInternalServerError, fmt.Sprintf("failed to list seeds: %v", err))
	}
	seed, exists := seeds[getter.GetDC()]
	if !exists {
		return nil, ctx, k8cerrors.NewNotFound("datacenter", getter.GetDC())
	}
	ctx = context.WithValue(ctx, datacenterContextKey, seed)

	clusterProvider, err := clusterProviderGetter(seed)
	if err != nil {
		return nil, ctx, k8cerrors.NewNotFound("cluster-provider", getter.GetDC())
	}

	return clusterProvider, ctx, nil
}
