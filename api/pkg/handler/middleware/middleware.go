package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/go-kit/kit/endpoint"
	transporthttp "github.com/go-kit/kit/transport/http"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/auth"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
	k8cerrors "github.com/kubermatic/kubermatic/api/pkg/util/errors"
	"github.com/kubermatic/kubermatic/api/pkg/util/hash"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
)

// contextKey defines a dedicated type for keys to use on contexts
type contextKey string

const (
	datacenterContextKey contextKey = "datacenter"

	// ClusterProviderContextKey key under which the current ClusterProvider is kept in the ctx
	ClusterProviderContextKey contextKey = "cluster-provider"

	// UserInfoContextKey key under which the current UserInfo is kept in the ctx
	UserInfoContextKey contextKey = "user-info"

	// UserCRContextKey key under which the current User (from the database) is kept in the ctx
	UserCRContextKey contextKey = "user-cr"

	// AuthenticatedUserContextKey key under which the current User (from OIDC provider) is kept in the ctx
	AuthenticatedUserContextKey contextKey = "authenticated-user"

	// rawTokenContextKey key under which the current token (OpenID ID Token) is kept in the ctx
	rawTokenContextKey contextKey = "raw-auth-token"
)

//DCGetter defines functionality to retrieve a datacenter name
type dCGetter interface {
	GetDC() string
}

// Datacenter is a middleware that injects the current ClusterProvider into the ctx
func Datacenter(clusterProviders map[string]provider.ClusterProvider, datacenters map[string]provider.DatacenterMeta) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			getter := request.(dCGetter)
			dc, exists := datacenters[getter.GetDC()]
			if !exists {
				return nil, errors.NewNotFound("datacenter", getter.GetDC())
			}
			ctx = context.WithValue(ctx, datacenterContextKey, dc)

			clusterProvider, exists := clusterProviders[getter.GetDC()]
			if !exists {
				return nil, errors.NewNotFound("cluster-provider", getter.GetDC())
			}
			ctx = context.WithValue(ctx, ClusterProviderContextKey, clusterProvider)
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
			return next(context.WithValue(ctx, UserCRContextKey, user), request)
		}
	}
}

// UserInfo is a middleware that creates UserInfo object from kubermaticapiv1.User (authenticated)
// and stores it in ctx under UserInfoContextKey key.
func UserInfo(userProjectMapper provider.ProjectMemberMapper) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			user, ok := ctx.Value(UserCRContextKey).(*kubermaticapiv1.User)
			if !ok {
				return nil, k8cerrors.New(http.StatusInternalServerError, "unable to get authenticated user object")
			}

			var projectID string
			prjIDGetter, ok := request.(common.ProjectIDGetter)
			if ok {
				projectID = prjIDGetter.GetProjectID()
			}

			uInfo, err := createUserInfo(user, projectID, userProjectMapper)
			if err != nil {
				return nil, common.KubernetesErrorToHTTPError(err)
			}

			return next(context.WithValue(ctx, UserInfoContextKey, uInfo), request)
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

			uInfo, err := createUserInfo(user, projectID, userProjectMapper)
			if err != nil {
				return nil, common.KubernetesErrorToHTTPError(err)
			}
			return next(context.WithValue(ctx, UserInfoContextKey, uInfo), request)
		}
	}
}

// Verifier knows how to verify an ID token from a request
func Verifier(o auth.OIDCVerifier) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			t := ctx.Value(rawTokenContextKey)
			token, ok := t.(string)
			if !ok || token == "" {
				return nil, k8cerrors.NewNotAuthorized()
			}

			verifyCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()
			claims, err := o.Verify(verifyCtx, token)
			if err != nil {
				return nil, k8cerrors.NewNotAuthorized()
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

// Extractor knows how to extract an ID token from the request
func Extractor(o auth.TokenExtractor) transporthttp.RequestFunc {
	return func(ctx context.Context, r *http.Request) context.Context {
		token := o.Extract(r)
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
