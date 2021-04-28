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

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticapiv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/auth"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	kubermaticcontext "k8c.io/kubermatic/v2/pkg/util/context"
	k8cerrors "k8c.io/kubermatic/v2/pkg/util/errors"
	"k8c.io/kubermatic/v2/pkg/util/hash"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	datacenterContextKey kubermaticcontext.Key = "datacenter"

	// RawTokenContextKey key under which the current token (OpenID ID Token) is kept in the ctx
	RawTokenContextKey kubermaticcontext.Key = "raw-auth-token"

	// TokenExpiryContextKey key under which the current token expiry (OpenID ID Token) is kept in the ctx
	TokenExpiryContextKey kubermaticcontext.Key = "auth-token-expiry"

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

	// ConstraintProviderContextKey key under which the current ConstraintProvider is kept in the ctx
	ConstraintProviderContextKey kubermaticcontext.Key = "constraint-provider"

	// PrivilegedConstraintProviderContextKey key under which the current PrivilegedConstraintProvider is kept in the ctx
	PrivilegedConstraintProviderContextKey kubermaticcontext.Key = "privileged-constraint-provider"

	// AlertmanagerProviderContextKey key under which the current AlertmanagerProvider is kept in the ctx
	AlertmanagerProviderContextKey kubermaticcontext.Key = "alertmanager_provider"


	UserCRContextKey                            = kubermaticcontext.UserCRContextKey
	SeedsGetterContextKey kubermaticcontext.Key = "seeds-getter"
)

//seedClusterGetter defines functionality to retrieve a seed name
type seedClusterGetter interface {
	GetSeedCluster() apiv1.SeedCluster
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
func TokenVerifier(tokenVerifier auth.TokenVerifier, userProvider provider.UserProvider) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			if rawTokenNotFoundErr := ctx.Value(noTokenFoundKey); rawTokenNotFoundErr != nil {
				tokenNotFoundErr, ok := rawTokenNotFoundErr.(error)
				if !ok {
					return nil, k8cerrors.NewNotAuthorized()
				}
				return nil, k8cerrors.NewWithDetails(http.StatusUnauthorized, "not authorized", []string{tokenNotFoundErr.Error()})
			}

			t := ctx.Value(RawTokenContextKey)
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

			if err := checkBlockedTokens(claims.Email, token, userProvider); err != nil {
				return nil, err
			}

			ctx = context.WithValue(ctx, TokenExpiryContextKey, claims.Expiry)
			return next(context.WithValue(ctx, AuthenticatedUserContextKey, user), request)
		}
	}
}

// Addons is a middleware that injects the current AddonProvider into the ctx
func Addons(clusterProviderGetter provider.ClusterProviderGetter, addonProviderGetter provider.AddonProviderGetter, seedsGetter provider.SeedsGetter) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			seedCluster := request.(seedClusterGetter).GetSeedCluster()

			addonProvider, err := getAddonProvider(clusterProviderGetter, addonProviderGetter, seedsGetter, seedCluster.SeedName, seedCluster.ClusterID)
			if err != nil {
				return nil, err
			}
			ctx = context.WithValue(ctx, AddonProviderContextKey, addonProvider)
			return next(ctx, request)
		}
	}
}

// PrivilegedAddons is a middleware that injects the current PrivilegedAddonProvider into the ctx
func PrivilegedAddons(clusterProviderGetter provider.ClusterProviderGetter, addonProviderGetter provider.AddonProviderGetter, seedsGetter provider.SeedsGetter) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			seedCluster := request.(seedClusterGetter).GetSeedCluster()
			addonProvider, err := getAddonProvider(clusterProviderGetter, addonProviderGetter, seedsGetter, seedCluster.SeedName, seedCluster.ClusterID)
			if err != nil {
				return nil, err
			}
			privilegedAddonProvider := addonProvider.(provider.PrivilegedAddonProvider)
			ctx = context.WithValue(ctx, PrivilegedAddonProviderContextKey, privilegedAddonProvider)
			return next(ctx, request)
		}
	}
}

func getAddonProvider(clusterProviderGetter provider.ClusterProviderGetter, addonProviderGetter provider.AddonProviderGetter, seedsGetter provider.SeedsGetter, seedName, clusterID string) (provider.AddonProvider, error) {
	seeds, err := seedsGetter()
	if err != nil {
		return nil, err
	}

	if clusterID != "" {
		for _, seed := range seeds {
			clusterProvider, err := clusterProviderGetter(seed)
			if err != nil {
				return nil, k8cerrors.NewNotFound("cluster-provider", clusterID)
			}
			if clusterProvider.IsCluster(clusterID) {
				seedName = seed.Name
				break
			}
		}
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
		return context.WithValue(ctx, RawTokenContextKey, token)
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
	getter, ok := request.(seedClusterGetter)
	if !ok {
		return nil, nil, fmt.Errorf("request is no dcGetter")
	}
	seeds, err := seedsGetter()
	if err != nil {
		return nil, ctx, k8cerrors.New(http.StatusInternalServerError, fmt.Sprintf("failed to list seeds: %v", err))
	}
	if getter.GetSeedCluster().ClusterID != "" {
		return getClusterProviderByClusterID(ctx, seeds, clusterProviderGetter, getter.GetSeedCluster().ClusterID)
	}

	seed, exists := seeds[getter.GetSeedCluster().SeedName]
	if !exists {
		return nil, ctx, k8cerrors.NewNotFound("seed", getter.GetSeedCluster().SeedName)
	}
	ctx = context.WithValue(ctx, datacenterContextKey, seed)

	clusterProvider, err := clusterProviderGetter(seed)
	if err != nil {
		return nil, ctx, k8cerrors.NewNotFound("cluster-provider", getter.GetSeedCluster().SeedName)
	}

	return clusterProvider, ctx, nil
}

func getClusterProviderByClusterID(ctx context.Context, seeds map[string]*kubermaticapiv1.Seed, clusterProviderGetter provider.ClusterProviderGetter, clusterID string) (provider.ClusterProvider, context.Context, error) {
	for _, seed := range seeds {
		clusterProvider, err := clusterProviderGetter(seed)
		if err != nil {
			return nil, ctx, k8cerrors.NewNotFound("cluster-provider", clusterID)
		}
		if clusterProvider.IsCluster(clusterID) {
			return clusterProvider, ctx, nil
		}
	}
	return nil, ctx, k8cerrors.NewNotFound("cluster-provider", clusterID)
}

func checkBlockedTokens(email, token string, userProvider provider.UserProvider) error {
	user, err := userProvider.UserByEmail(email)
	if err != nil {
		if err != provider.ErrNotFound {
			return common.KubernetesErrorToHTTPError(err)
		}
		return nil
	}
	blockedTokens, err := userProvider.GetUserBlacklistTokens(user)
	if err != nil {
		return common.KubernetesErrorToHTTPError(err)
	}
	tokenSet := sets.NewString(blockedTokens...)
	if tokenSet.Has(token) {
		return k8cerrors.NewNotAuthorized()
	}

	return nil
}

// SetSeedsGetter injects the current SeedsGetter into the ctx
func SetSeedsGetter(seedsGetter provider.SeedsGetter) transporthttp.RequestFunc {
	return func(ctx context.Context, r *http.Request) context.Context {
		return context.WithValue(ctx, SeedsGetterContextKey, seedsGetter)
	}
}

// Constraints is a middleware that injects the current ConstraintProvider into the ctx
func Constraints(clusterProviderGetter provider.ClusterProviderGetter, constraintProviderGetter provider.ConstraintProviderGetter, seedsGetter provider.SeedsGetter) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			seedCluster := request.(seedClusterGetter).GetSeedCluster()

			constraintProvider, err := getConstraintProvider(clusterProviderGetter, constraintProviderGetter, seedsGetter, seedCluster.SeedName, seedCluster.ClusterID)
			if err != nil {
				return nil, err
			}
			ctx = context.WithValue(ctx, ConstraintProviderContextKey, constraintProvider)
			return next(ctx, request)
		}
	}
}

// PrivilegedConstraints is a middleware that injects the current PrivilegedConstraintProvider into the ctx
func PrivilegedConstraints(clusterProviderGetter provider.ClusterProviderGetter, constraintProviderGetter provider.ConstraintProviderGetter, seedsGetter provider.SeedsGetter) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			seedCluster := request.(seedClusterGetter).GetSeedCluster()
			constraintProvider, err := getConstraintProvider(clusterProviderGetter, constraintProviderGetter, seedsGetter, seedCluster.SeedName, seedCluster.ClusterID)
			if err != nil {
				return nil, err
			}
			privilegedConstraintProvider := constraintProvider.(provider.PrivilegedConstraintProvider)
			ctx = context.WithValue(ctx, PrivilegedConstraintProviderContextKey, privilegedConstraintProvider)
			return next(ctx, request)
		}
	}
}

func getConstraintProvider(clusterProviderGetter provider.ClusterProviderGetter, constraintProviderGetter provider.ConstraintProviderGetter, seedsGetter provider.SeedsGetter, seedName, clusterID string) (provider.ConstraintProvider, error) {
	seeds, err := seedsGetter()
	if err != nil {
		return nil, err
	}

	if clusterID != "" {
		for _, seed := range seeds {
			clusterProvider, err := clusterProviderGetter(seed)
			if err != nil {
				return nil, k8cerrors.NewNotFound("cluster-provider", clusterID)
			}
			if clusterProvider.IsCluster(clusterID) {
				seedName = seed.Name
				break
			}
		}
	}

	seed, found := seeds[seedName]
	if !found {
		return nil, fmt.Errorf("couldn't find seed %q", seedName)
	}

	return constraintProviderGetter(seed)
}

// Alertmanagers is a middleware that injects the current AlertmanagerProvider into the ctx
func Alertmanagers(clusterProviderGetter provider.ClusterProviderGetter, alertmanagerProviderGetter provider.AlertmanagerProviderGetter, seedsGetter provider.SeedsGetter) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			seedCluster := request.(seedClusterGetter).GetSeedCluster()

			alertmanagerProvider, err := getAlertmanagerProvider(clusterProviderGetter, alertmanagerProviderGetter, seedsGetter, seedCluster.SeedName, seedCluster.ClusterID)
			if err != nil {
				return nil, err
			}
			ctx = context.WithValue(ctx, AlertmanagerProviderContextKey, alertmanagerProvider)
			return next(ctx, request)
		}
	}
}

func getAlertmanagerProvider(clusterProviderGetter provider.ClusterProviderGetter, alertmanagerProviderGetter provider.AlertmanagerProviderGetter, seedsGetter provider.SeedsGetter, seedName, clusterID string) (provider.AlertmanagerProvider, error) {
	seeds, err := seedsGetter()
	if err != nil {
		return nil, err
	}

	if clusterID != "" {
		for _, seed := range seeds {
			clusterProvider, err := clusterProviderGetter(seed)
			if err != nil {
				return nil, k8cerrors.NewNotFound("cluster-provider", clusterID)
			}
			if clusterProvider.IsCluster(clusterID) {
				seedName = seed.Name
				break
			}
		}
	}

	seed, found := seeds[seedName]
	if !found {
		return nil, fmt.Errorf("couldn't find seed %q", seedName)
	}

	return alertmanagerProviderGetter(seed)
}
